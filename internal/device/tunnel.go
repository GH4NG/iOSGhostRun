package device

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/tunnel"

	"iOSGhostRun/internal/storage"
)

// 隧道配置常量
const (
	defaultTunnelInfoPort = 49151
	tunnelUpdateInterval  = 2 * time.Second
)

var globalTunnelManager *tunnel.TunnelManager

// StartTunnelService 启动 iOS 17+ 隧道服务
func StartTunnelService(ctx context.Context) error {
	pairRecordPath := storage.ResolveAppDir("pairrecords")

	pm, err := tunnel.NewPairRecordManager(pairRecordPath)
	if err != nil {
		return err
	}

	userspaceTUN := ios.CheckRoot() != nil

	tm := tunnel.NewTunnelManager(pm, userspaceTUN)
	globalTunnelManager = tm

	// 启动隧道更新协程
	go func() {
		ticker := time.NewTicker(tunnelUpdateInterval)
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
			http.Error(w, "udid required", http.StatusBadRequest)
			return
		}

		tunnels, err := tm.ListTunnels()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, t := range tunnels {
			if t.Udid == udid {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(t)
				return
			}
		}

		http.Error(w, "tunnel not found", http.StatusNotFound)
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
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {

		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
	tm.Close()
	globalTunnelManager = nil

	return nil
}

// GetTunnelForDevice 获取指定设备的隧道
func GetTunnelForDevice(udid string) (tunnel.Tunnel, error) {
	return tunnel.TunnelInfoForDevice(udid, "localhost", defaultTunnelInfoPort)
}

// ListRunningTunnels 列出所有运行中的隧道
func ListRunningTunnels() ([]tunnel.Tunnel, error) {
	return tunnel.ListRunningTunnels("localhost", defaultTunnelInfoPort)
}
