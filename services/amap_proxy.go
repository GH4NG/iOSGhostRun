package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type MapProxyStatus struct {
	State      string `json:"state"`
	Running    bool   `json:"running"`
	Configured bool   `json:"configured"`
	Address    string `json:"address"`
	Requests   uint64 `json:"requests"`
	LastError  string `json:"lastError"`
	Message    string `json:"message"`
}

type amapProxy struct {
	server      *http.Server
	client      *http.Client
	address     string
	serviceHost string
	prefix      string
	mu          sync.RWMutex
	requests    uint64
	lastError   string
}

// Wails 管理服务生命周期；代理启动失败时仍打开 GUI，让用户查看并重试。
func (s *SettingsService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	s.RestartMapProxy()
	return nil
}

func (s *SettingsService) ServiceShutdown() error {
	s.proxyControlMu.Lock()
	defer s.proxyControlMu.Unlock()
	return s.stopMapProxy()
}

func (s *SettingsService) RestartMapProxy() MapProxyStatus {
	s.proxyControlMu.Lock()
	defer s.proxyControlMu.Unlock()
	_ = s.stopMapProxy()
	s.startMapProxy()
	return s.GetMapProxyStatus()
}

func (s *SettingsService) startMapProxy() {
	token := make([]byte, 24)
	if _, err := rand.Read(token); err != nil {
		s.setProxyError("无法生成本地地图服务凭据")
		return
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		s.setProxyError("无法启动本地地图服务，请检查本机网络权限")
		return
	}
	address := "http://" + listener.Addr().String()
	// JSAPI 要求 _AMapService 是一级路由；随机凭据放在它后面。
	prefix := "/_AMapService/" + hex.EncodeToString(token) + "/"
	p := &amapProxy{
		address:     address,
		serviceHost: address + strings.TrimSuffix(prefix, "/"),
		prefix:      prefix,
		client:      newDownloadHTTPClient(25 * time.Second),
	}
	p.server = &http.Server{
		Handler:           s.mapProxyHandler(p, listener.Addr().String()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	s.proxyMu.Lock()
	s.mapProxy = p
	s.proxyError = ""
	s.proxyMu.Unlock()
	go func() {
		_ = p.server.Serve(listener)
		s.proxyMu.Lock()
		if s.mapProxy == p {
			s.mapProxy = nil
			s.proxyError = "本地地图服务意外停止，请点击重启服务"
		}
		s.proxyMu.Unlock()
		p.client.CloseIdleConnections()
	}()
	Log.Info("MapService", "内置高德安全代理已启动: "+address)
}

func (s *SettingsService) setProxyError(message string) {
	s.proxyMu.Lock()
	s.proxyError = message
	s.proxyMu.Unlock()
	Log.Error("MapService", message)
}

func (s *SettingsService) stopMapProxy() error {
	s.proxyMu.Lock()
	p := s.mapProxy
	s.mapProxy = nil
	s.proxyError = ""
	s.proxyMu.Unlock()
	if p == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := p.server.Shutdown(ctx)
	if err != nil {
		_ = p.server.Close()
	}
	p.client.CloseIdleConnections()
	return err
}

func (s *SettingsService) GetMapProxyStatus() MapProxyStatus {
	settings, settingsErr := s.readMapSettings()
	s.proxyMu.RLock()
	p, startError := s.mapProxy, s.proxyError
	s.proxyMu.RUnlock()
	status := MapProxyStatus{Configured: settings.Key != "" && settings.SecurityJsCode != ""}
	if p != nil {
		status.Running = true
		status.Address = p.address
		p.mu.RLock()
		status.Requests, status.LastError = p.requests, p.lastError
		p.mu.RUnlock()
	}
	switch {
	case settingsErr != nil:
		status.State, status.Message = "error", "读取地图设置失败，请打开设置重新保存"
	case startError != "":
		status.State, status.Message = "error", startError
		status.LastError = startError
	case !status.Running:
		status.State, status.Message = "stopped", "本地地图服务已停止"
	case !status.Configured:
		status.State, status.Message = "unconfigured", "服务已启动，请填写 JS API Key 和安全密钥"
	case status.LastError != "":
		status.State, status.Message = "error", status.LastError
	default:
		status.State, status.Message = "running", "本地安全代理运行中"
	}
	return status
}

// 地图 SDK 仅取得客户端 Key 和本地代理地址，不取得安全密钥。
func (s *SettingsService) GetMapRuntimeConfig() (MapRuntimeConfig, error) {
	settings, err := s.readMapSettings()
	if err != nil {
		return MapRuntimeConfig{}, err
	}
	if settings.Key == "" || settings.SecurityJsCode == "" {
		return MapRuntimeConfig{}, nil
	}
	s.proxyMu.RLock()
	defer s.proxyMu.RUnlock()
	if s.mapProxy == nil {
		return MapRuntimeConfig{}, fmt.Errorf("本地地图服务未启动，请打开设置并重启服务")
	}
	return MapRuntimeConfig{Key: settings.Key, ServiceHost: s.mapProxy.serviceHost}, nil
}

func allowedMapOrigin(origin string) bool {
	// 自定义 WebView scheme 的跨域请求可能使用 null；每次启动的随机路径另行鉴权。
	if origin == "" || origin == "null" || origin == "wails://wails" || origin == "wails://localhost" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == "" &&
		(u.Hostname() == "wails.localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1")
}

func (s *SettingsService) mapProxyHandler(p *amapProxy, host string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host || !strings.HasPrefix(r.URL.Path, p.prefix) {
			http.NotFound(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if !allowedMapOrigin(origin) {
			http.Error(w, "Origin not allowed", http.StatusForbidden)
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST, OPTIONS")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, p.prefix)
		if path == "" || strings.Contains(path, "..") || strings.Contains(path, "//") || strings.Contains(path, "\\") {
			http.Error(w, "Invalid map service path", http.StatusBadRequest)
			return
		}
		settings, err := s.readMapSettings()
		if err != nil || settings.Key == "" || settings.SecurityJsCode == "" {
			http.Error(w, "请先配置高德 JS API Key 和安全密钥", http.StatusServiceUnavailable)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		upstream := "restapi.amap.com"
		if path == "v4/map/styles" || path == "v4/map/styles2" {
			upstream = "webapi.amap.com"
		}
		p.mu.Lock()
		p.requests++
		p.mu.Unlock()
		proxy := &httputil.ReverseProxy{
			Rewrite: func(request *httputil.ProxyRequest) {
				request.SetURL(&url.URL{Scheme: "https", Host: upstream})
				request.Out.URL.Path = "/" + path
				request.Out.URL.RawPath = ""
				query := request.In.URL.Query()
				query.Set("key", settings.Key)
				query.Set("jscode", settings.SecurityJsCode)
				request.Out.URL.RawQuery = query.Encode()
				// 不向上游传递本地请求中的 Cookie、Authorization 等信息。
				request.Out.Header = make(http.Header)
				for _, name := range []string{"Accept", "Content-Type", "Referer"} {
					if value := request.In.Header.Get(name); value != "" {
						request.Out.Header.Set(name, value)
					}
				}
			},
			Transport: p.client.Transport,
			ModifyResponse: func(response *http.Response) error {
				// 不把上游重定向（可能包含鉴权参数）交给浏览器。
				if response.StatusCode >= 300 && response.StatusCode < 400 {
					return errors.New("unexpected upstream redirect")
				}
				for name := range response.Header {
					if strings.HasPrefix(strings.ToLower(name), "access-control-") || strings.EqualFold(name, "Set-Cookie") {
						response.Header.Del(name)
					}
				}
				response.Header.Set("Cache-Control", "no-store")
				p.mu.Lock()
				p.lastError = ""
				if response.StatusCode >= 400 {
					p.lastError = fmt.Sprintf("高德服务返回 HTTP %d，请检查配置或稍后重试", response.StatusCode)
				}
				p.mu.Unlock()
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
				// 错误文本不包含带有安全密钥的上游 URL。
				message := "连接高德服务失败，请检查网络后重试"
				p.mu.Lock()
				p.lastError = message
				p.mu.Unlock()
				http.Error(w, message, http.StatusBadGateway)
			},
		}
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		proxy.ServeHTTP(w, r.WithContext(ctx))
	})
}
