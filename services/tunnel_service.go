package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/tunnel"
)

const defaultTunnelInfoPort = 49151

var globalTunnelManager *tunnel.TunnelManager
var globalTunnelCancel context.CancelFunc
var tunnelStateMu sync.Mutex

// StartTunnel 启动 tunnel
func StartTunnel(ctx context.Context) error {
	pairRecordPath := ResolveAppDir("pairrecords")

	pm, err := tunnel.NewPairRecordManager(pairRecordPath)
	if err != nil {
		return err
	}

	userspaceTUN := tunnel.CheckPermissions() != nil
	tm := tunnel.NewTunnelManager(pm, userspaceTUN)
	tunnelCtx, cancelTunnel := context.WithCancel(ctx)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", defaultTunnelInfoPort))
	if err != nil {
		cancelTunnel()
		tm.Close()
		return fmt.Errorf("启动 tunnel HTTP 服务失败: %w", err)
	}

	tunnelStateMu.Lock()
	if globalTunnelManager != nil {
		tunnelStateMu.Unlock()
		_ = listener.Close()
		tm.Close()
		cancelTunnel()
		return nil
	}
	globalTunnelManager = tm
	globalTunnelCancel = cancelTunnel
	tunnelStateMu.Unlock()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-tunnelCtx.Done():
				return
			case <-ticker.C:
				_ = tm.UpdateTunnels(tunnelCtx)
			}
		}
	}()

	// HTTP 接口
	mux := http.NewServeMux()

	// 列出所有隧道
	mux.HandleFunc("/tunnels", func(w http.ResponseWriter, r *http.Request) {
		tunnels, err := tm.ListTunnels()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tunnels)
	})

	// 获取指定设备的隧道
	mux.HandleFunc("/tunnel/", func(w http.ResponseWriter, r *http.Request) {
		udid := r.URL.Path[len("/tunnel/"):]
		if udid == "" {
			http.Error(w, "需要 UDID", http.StatusBadRequest)
			return
		}
		tunnels, err := tm.ListTunnels()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, tinfo := range tunnels {
			if tinfo.Udid == udid {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tinfo)
				return
			}
		}
		http.Error(w, "隧道未找到", http.StatusNotFound)
	})

	// 根路径返回所有隧道
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		tunnels, err := tm.ListTunnels()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tunnels)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", defaultTunnelInfoPort),
		Handler: mux,
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			Log.Error("TunnelService", "隧道 HTTP 服务器错误: "+err.Error())
		}
	}()

	<-tunnelCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	_ = listener.Close()
	tm.Close()

	tunnelStateMu.Lock()
	globalTunnelManager = nil
	globalTunnelCancel = nil
	tunnelStateMu.Unlock()
	return nil
}

// StopTunnel 停止 tunnel
func StopTunnel() error {
	tunnelStateMu.Lock()
	cancel := globalTunnelCancel
	tunnelStateMu.Unlock()

	if cancel != nil {
		cancel()
	}
	return nil
}

func ensureTunnelReady(udid string) error {
	tunnelStateMu.Lock()
	needStart := globalTunnelManager == nil
	tunnelStateMu.Unlock()

	if needStart {
		go func() {
			if err := StartTunnel(context.Background()); err != nil {
				Log.Error("TunnelService", "启动隧道服务失败: "+err.Error())
			}
		}()
	}

	if err := waitTunnelDeviceReady(udid, 15*time.Second); err != nil {
		return fmt.Errorf("等待隧道就绪失败: %w", err)
	}

	return nil
}

func waitTunnelDeviceReady(udid string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		if _, err := getTunnelDevice(udid); err == nil {
			return nil
		} else {
			lastErr = err
		}

		time.Sleep(300 * time.Millisecond)
	}

	if lastErr != nil {
		return fmt.Errorf("设备隧道未就绪: %w", lastErr)
	}

	return fmt.Errorf("设备隧道未就绪")
}

// GetTunnelForDevice 获取指定设备的隧道信息
func getTunnelDevice(udid string) (*ios.DeviceEntry, error) {
	tunnelInfo, err := tunnel.TunnelInfoForDevice(udid, "localhost", defaultTunnelInfoPort)
	if err != nil {
		if strings.Contains(err.Error(), "invalid character") {
			return nil, fmt.Errorf("隧道服务尚未就绪")
		}
		return nil, fmt.Errorf("获取 tunnel 信息失败: %w", err)
	}

	// 先获取基础设备信息
	baseDevice, err := ios.GetDevice(udid)
	if err != nil {
		return nil, fmt.Errorf("获取设备信息失败: %w", err)
	}

	// 设置 userspace TUN 信息
	baseDevice.UserspaceTUN = tunnelInfo.UserspaceTUN
	baseDevice.UserspaceTUNPort = tunnelInfo.UserspaceTUNPort
	baseDevice.UserspaceTUNHost = "localhost"

	// 连接到 RSD 服务获取服务端口映射
	rsdService, err := ios.NewWithAddrPortDevice(tunnelInfo.Address, tunnelInfo.RsdPort, baseDevice)
	if err != nil {
		return nil, fmt.Errorf("连接 RSD 服务失败: %w", err)
	}
	defer rsdService.Close()

	// 执行握手获取 RsdPortProvider
	rsdProvider, err := rsdService.Handshake()
	if err != nil {
		return nil, fmt.Errorf("RSD 握手失败: %w", err)
	}

	// 使用 RsdPortProvider 获取带 tunnel 信息的设备
	device, err := ios.GetDeviceWithAddress(udid, tunnelInfo.Address, rsdProvider)
	if err != nil {
		return nil, fmt.Errorf("通过 tunnel 获取设备失败: %w", err)
	}

	// 保留 userspace TUN 设置
	device.UserspaceTUN = tunnelInfo.UserspaceTUN
	device.UserspaceTUNPort = tunnelInfo.UserspaceTUNPort
	device.UserspaceTUNHost = "localhost"

	return &device, nil
}
