package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type mapRoundTripper func(*http.Request) (*http.Response, error)

func (f mapRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func setupMapService(t *testing.T) *SettingsService {
	t.Helper()
	previous := dataDirectory
	dataDirectory = t.TempDir()
	s := NewSettingsService(nil)
	t.Cleanup(func() { _ = s.ServiceShutdown(); dataDirectory = previous })
	if err := s.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestMapProxyLifecycle(t *testing.T) {
	s := setupMapService(t)
	status := s.GetMapProxyStatus()
	if !status.Running || status.State != "unconfigured" || !strings.HasPrefix(status.Address, "http://127.0.0.1:") {
		t.Fatalf("initial status: %+v", status)
	}
	if _, err := s.SaveMapSettings(MapSettings{Key: "key", SecurityJsCode: "secret"}); err != nil {
		t.Fatal(err)
	}
	config, err := s.GetMapRuntimeConfig()
	if err != nil || config.Key != "key" || !strings.HasPrefix(config.ServiceHost, status.Address+"/") {
		t.Fatalf("runtime config: %+v, %v", config, err)
	}
	endpoint, err := url.Parse(config.ServiceHost)
	if err != nil {
		t.Fatal(err)
	}
	segments := strings.Split(strings.Trim(endpoint.Path, "/"), "/")
	if len(segments) != 2 || segments[0] != "_AMapService" || len(segments[1]) != 48 {
		t.Fatalf("JSAPI requires _AMapService as the first route, followed by a random capability: %s", endpoint.Path)
	}
	encoded, _ := json.Marshal(config)
	if strings.Contains(string(encoded), "secret") {
		t.Fatal("runtime config leaked secret")
	}
	oldURL := config.ServiceHost + "/v3/place/text"
	status = s.RestartMapProxy()
	config, _ = s.GetMapRuntimeConfig()
	if !status.Running || status.State != "running" || config.ServiceHost+"/v3/place/text" == oldURL {
		t.Fatal("restart did not create a new service endpoint")
	}
	if err := s.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	if status := s.GetMapProxyStatus(); status.Running || status.State != "stopped" {
		t.Fatalf("shutdown status: %+v", status)
	}
	if _, err := s.GetMapRuntimeConfig(); err == nil {
		t.Fatal("stopped service must not return a usable map configuration")
	}
	if _, err := http.Get(config.ServiceHost + "/v3/place/text"); err == nil {
		t.Fatal("listener still accepts requests after shutdown")
	}
}

func TestMapProxyRoutesAndCredentialIsolation(t *testing.T) {
	s := setupMapService(t)
	_, _ = s.SaveMapSettings(MapSettings{Key: "configured-key", SecurityJsCode: "configured-secret"})
	s.proxyMu.RLock()
	p := s.mapProxy
	s.proxyMu.RUnlock()
	var received []*http.Request
	// Inject a transport before requests begin; no real credentials leave the test.
	p.client.Transport = mapRoundTripper(func(r *http.Request) (*http.Response, error) {
		received = append(received, r.Clone(r.Context()))
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}, "Access-Control-Allow-Origin": {"*"}, "Set-Cookie": {"upstream=secret"}}, Body: io.NopCloser(strings.NewReader(`{"status":"1"}`))}, nil
	})
	for _, path := range []string{"v3/place/text", "v4/map/styles", "v4/map/styles2"} {
		req, _ := http.NewRequest("GET", p.serviceHost+"/"+path+"?key=attacker&jscode=wrong&callback=callback1&keywords=test", nil)
		req.Header.Set("Origin", "http://wails.localhost")
		req.Header.Set("Cookie", "local=private")
		req.Header.Set("Authorization", "Bearer private")
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 200 || response.Header.Get("Access-Control-Allow-Origin") != "http://wails.localhost" || response.Header.Get("Set-Cookie") != "" {
			t.Fatalf("bad proxy response headers: %v", response.Header)
		}
	}
	if len(received) != 3 || received[0].URL.Host != "restapi.amap.com" || received[1].URL.Host != "webapi.amap.com" || received[2].URL.Host != "webapi.amap.com" {
		t.Fatalf("incorrect upstream routing: %+v", received)
	}
	for _, r := range received {
		if r.URL.Scheme != "https" || r.URL.Query().Get("key") != "configured-key" || r.URL.Query().Get("jscode") != "configured-secret" || r.URL.Query().Get("callback") != "callback1" {
			t.Fatalf("incorrect upstream query: %s", r.URL)
		}
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("local credentials forwarded")
		}
	}
	if status := s.GetMapProxyStatus(); status.Requests != 3 || status.State != "running" {
		t.Fatalf("bad request accounting: %+v", status)
	}
	// A saved credential change is used immediately without restarting the listener.
	_, _ = s.SaveMapSettings(MapSettings{Key: "rotated-key", SecurityJsCode: "rotated-secret"})
	response, err := http.Get(p.serviceHost + "/v3/place/text")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if received[3].URL.Query().Get("jscode") != "rotated-secret" {
		t.Fatal("proxy kept stale credentials")
	}
}

func TestMapProxyRejectsUntrustedRequestsAndRedactsErrors(t *testing.T) {
	s := setupMapService(t)
	_, _ = s.SaveMapSettings(MapSettings{Key: "key", SecurityJsCode: "private-secret"})
	s.proxyMu.RLock()
	p := s.mapProxy
	s.proxyMu.RUnlock()
	forwarded := 0
	p.client.Transport = mapRoundTripper(func(r *http.Request) (*http.Response, error) {
		forwarded++
		return nil, errors.New("cannot reach " + r.URL.String())
	})
	for _, tc := range []struct {
		path, method, origin, host string
		want                       int
	}{
		{"/_AMapService/v3/place/text", "GET", "", "", 404},
		{p.prefix + "v3/place/text", "GET", "https://untrusted.example", "", 403},
		{p.prefix + "v3/place/text", "GET", "", "untrusted.example", 404},
		{p.prefix + "v3/place/text", "CONNECT", "", "", 404},
		{p.prefix + "v3/place/text", "DELETE", "", "", 405},
		{p.prefix + "../v3/place/text", "GET", "", "", 400},
	} {
		req := httptest.NewRequest(tc.method, p.address+tc.path, nil)
		req.Header.Set("Origin", tc.origin)
		if tc.host != "" {
			req.Host = tc.host
		}
		w := httptest.NewRecorder()
		s.mapProxyHandler(p, strings.TrimPrefix(p.address, "http://")).ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, w.Code, tc.want)
		}
	}
	if forwarded != 0 {
		t.Fatal("untrusted requests reached upstream")
	}
	req, _ := http.NewRequest("OPTIONS", p.serviceHost+"/v3/place/text", nil)
	req.Header.Set("Origin", "http://wails.localhost")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 204 || response.Header.Get("Access-Control-Allow-Private-Network") != "true" {
		t.Fatal("preflight failed")
	}
	response, err = http.Get(p.serviceHost + "/v3/place/text")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	status := s.GetMapProxyStatus()
	if response.StatusCode != 502 || status.State != "error" || strings.Contains(string(body)+status.LastError, "private-secret") {
		t.Fatalf("network error handling leaked data or lost state: %+v", status)
	}
	// A later successful request clears the connection error.
	p.client.Transport = mapRoundTripper(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})
	response, err = http.Get(p.serviceHost + "/v3/place/text")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if s.GetMapProxyStatus().State != "running" {
		t.Fatal("proxy did not recover after a successful request")
	}
}
