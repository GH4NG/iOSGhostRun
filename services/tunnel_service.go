package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/tunnel"
)

// TunnelService 隧道管理服务
type TunnelService struct {
	mu         sync.Mutex
	tm         *tunnel.TunnelManager
	httpServer *http.Server
	cancel     context.CancelFunc
}

var globalTunnelManager *tunnel.TunnelManager
var defaultTunnelInfoPort = 49151

// startTunnel 启动 tunnel
func (t *TunnelService) startTunnel() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.tm != nil {
		return nil
	}

	pairRecordPath := ResolveAppDir("pairrecords")

	pm, err := tunnel.NewPairRecordManager(pairRecordPath)
	if err != nil {
		return err
	}

	userspaceTUN := ios.CheckRoot() != nil
	tm := tunnel.NewTunnelManager(pm, userspaceTUN)
	t.tm = tm
	globalTunnelManager = tm

	// 更新协程
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = tm.UpdateTunnels(ctx)
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
		json.NewEncoder(w).Encode(tunnels)
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
				json.NewEncoder(w).Encode(tinfo)
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
		json.NewEncoder(w).Encode(tunnels)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", defaultTunnelInfoPort),
		Handler: mux,
	}

	// 启动 HTTP 服务器
	t.httpServer = server

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			Log.Error("TunnelService", "隧道 HTTP 服务器错误: "+err.Error())
		}
	}()

	// 尝试立即更新一次 tunnel 状态
	go func() {
		_ = tm.UpdateTunnels(context.Background())
		time.Sleep(500 * time.Millisecond)
		_ = tm.UpdateTunnels(context.Background())
	}()

	return nil
}

// StopTunnel 停止 tunnel
func (t *TunnelService) StopTunnel() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	if t.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = t.httpServer.Shutdown(shutdownCtx)
		t.httpServer = nil
	}
	if t.tm != nil {
		_ = t.tm.Close()
		t.tm = nil
		globalTunnelManager = nil
	}
	return nil
}

// GetTunnelForDevice 获取指定设备的隧道信息
func GetTunnelForDevice(udid string) (tunnel.Tunnel, error) {
	return tunnel.TunnelInfoForDevice(udid, "localhost", defaultTunnelInfoPort)
}

// ListRunningTunnels 列出所有运行中的隧道
func ListRunningTunnels() ([]tunnel.Tunnel, error) {
	return tunnel.ListRunningTunnels("localhost", defaultTunnelInfoPort)
}
