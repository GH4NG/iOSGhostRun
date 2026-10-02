package services

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	pac "github.com/phlipse/go-pac"
)

var downloadProxy = &proxyResolver{}
var imageTransportOnce sync.Once
var downloadBaseTransport *http.Transport
var originalProxyEnvironment []string

type proxyResolver struct {
	once   sync.Once
	pac    *pac.PACProxy
	manual *url.URL
	err    error
}

func ConfigureDownloadProxy() {
	imageTransportOnce.Do(func() {
		originalProxyEnvironment = os.Environ()
		base, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			Log.Warn("Network", "默认 HTTP 传输层类型异常，无法配置镜像下载代理")
			return
		}
		downloadBaseTransport = base.Clone()
		base.Proxy = imageProxyForRequest
	})
}

func imageProxyForRequest(req *http.Request) (*url.URL, error) {
	if isImageDownloadHost(req.URL.Hostname()) {
		return downloadProxy.proxyForRequest(req)
	}
	if downloadBaseTransport.Proxy != nil {
		return downloadBaseTransport.Proxy(req)
	}
	return nil, nil
}

func isImageDownloadHost(host string) bool {
	host = strings.ToLower(host)
	for _, domain := range []string{"github.com", "githubusercontent.com", "deviceboxhq.com", "gs.apple.com"} {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func newDownloadHTTPClient(timeout time.Duration) *http.Client {
	base := downloadBaseTransport
	if base == nil {
		base, _ = http.DefaultTransport.(*http.Transport)
	}
	if base == nil {
		return &http.Client{Timeout: timeout}
	}
	transport := base.Clone()
	transport.Proxy = downloadProxy.proxyForRequest
	return &http.Client{Transport: transport, Timeout: timeout}
}

func (r *proxyResolver) proxyForRequest(req *http.Request) (*url.URL, error) {
	if isLocalHost(req.URL.Hostname()) {
		return nil, nil
	}
	if hasProxyEnvironment(req.URL.Scheme) {
		return http.ProxyFromEnvironment(req)
	}
	r.once.Do(r.load)
	if r.err != nil {
		return nil, r.err
	}
	if r.pac != nil {
		return r.pac.ProxyFunc()(req)
	}
	return r.manual, nil
}

func isLocalHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func hasProxyEnvironment(scheme string) bool {
	keys := []string{"HTTPS_PROXY", "https_proxy"}
	if scheme == "http" {
		keys = []string{"HTTP_PROXY", "http_proxy"}
	}
	for _, key := range keys {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}

func (r *proxyResolver) load() {
	pacURL, err := configuredPACURL()
	if err != nil {
		r.err = fmt.Errorf("读取 PAC 代理配置失败: %w", err)
		Log.Warn("Network", r.err.Error())
		return
	}
	if pacURL != nil {
		if pacURL.Scheme != "http" && pacURL.Scheme != "https" {
			r.err = fmt.Errorf("PAC 地址仅支持 HTTP 或 HTTPS: %s", pacURL.Scheme)
			Log.Warn("Network", r.err.Error())
			return
		}
		client := &http.Client{Transport: downloadBaseTransport, Timeout: 10 * time.Second}
		r.pac, r.err = pac.NewPACProxy(pacURL, &pac.PACProxyConfig{Client: client})
		if r.err != nil {
			r.err = fmt.Errorf("加载 PAC 代理脚本失败: %w", r.err)
			Log.Warn("Network", r.err.Error())
			return
		}
		Log.Info("Network", "下载请求将按 PAC 脚本选择代理")
		return
	}

	proxy, err := detectSystemProxy()
	if err != nil {
		Log.Warn("Network", fmt.Sprintf("检测系统代理失败，下载将直连: %v", err))
		return
	}
	if proxy == "" {
		Log.Info("Network", "未检测到系统代理，下载将直连")
		return
	}
	r.manual, err = url.Parse(proxy)
	if err != nil || r.manual.Host == "" {
		r.err = fmt.Errorf("系统代理地址无效: %s", proxy)
		return
	}
	Log.Info("Network", "下载请求将使用系统代理")
}

func configuredPACURL() (*url.URL, error) {
	raw := strings.TrimSpace(os.Getenv("IOSGHOSTRUN_PAC_URL"))
	if raw == "" {
		switch runtime.GOOS {
		case "windows":
			value, err := readRegValue(`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`, "AutoConfigURL")
			if err == nil {
				raw = value
			}
		case "darwin":
			fields, err := macProxySettings()
			if err != nil {
				return nil, err
			}
			if fields["ProxyAutoConfigEnable"] == "1" {
				raw = fields["ProxyAutoConfigURLString"]
			}
		case "linux":
			modeCmd := exec.Command("gsettings", "get", "org.gnome.system.proxy", "mode")
			modeCmd.Env = originalProxyEnvironment
			mode, err := modeCmd.Output()
			if err == nil && strings.Trim(string(mode), " \r\n'\"") == "auto" {
				urlCmd := exec.Command("gsettings", "get", "org.gnome.system.proxy", "autoconfig-url")
				urlCmd.Env = originalProxyEnvironment
				value, err := urlCmd.Output()
				if err != nil {
					return nil, err
				}
				raw = strings.Trim(string(value), " \r\n'\"")
			}
		}
	}
	if raw == "" {
		return nil, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return nil, fmt.Errorf("PAC 地址无效: %s", raw)
	}
	return parsed, nil
}

func detectSystemProxy() (string, error) {
	switch runtime.GOOS {
	case "windows":
		key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
		enable, err := readRegValue(key, "ProxyEnable")
		if err != nil || !strings.EqualFold(strings.TrimSpace(enable), "0x1") {
			return "", nil
		}
		server, err := readRegValue(key, "ProxyServer")
		if err != nil {
			return "", fmt.Errorf("读取 ProxyServer 失败: %w", err)
		}
		return parseProxyServer(server), nil
	case "darwin":
		fields, err := macProxySettings()
		if err != nil {
			return "", err
		}
		if fields["HTTPSEnable"] == "1" {
			if p := normalizeProxyHost(fields["HTTPSProxy"] + ":" + fields["HTTPSPort"]); p != "" {
				return p, nil
			}
		}
		if fields["HTTPEnable"] == "1" {
			return normalizeProxyHost(fields["HTTPProxy"] + ":" + fields["HTTPPort"]), nil
		}
	}
	return "", nil
}

func macProxySettings() (map[string]string, error) {
	out, err := exec.Command("scutil", "--proxy").Output()
	if err != nil {
		return nil, fmt.Errorf("执行 scutil --proxy 失败: %w", err)
	}
	fields := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, ":"); idx > 0 {
			fields[strings.TrimSpace(line[:idx])] = strings.TrimSpace(line[idx+1:])
		}
	}
	return fields, nil
}

func readRegValue(key, name string) (string, error) {
	out, err := exec.Command("reg", "query", key, "/v", name).Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && strings.EqualFold(fields[0], name) {
			return fields[len(fields)-1], nil
		}
	}
	return "", fmt.Errorf("注册表值 %s 不存在", name)
}

func parseProxyServer(server string) string {
	server = strings.TrimSpace(server)
	if server == "" {
		return ""
	}
	if !strings.Contains(server, "=") && !strings.Contains(server, ";") {
		return normalizeProxyHost(server)
	}
	var httpProxy, httpsProxy string
	for _, part := range strings.Split(server, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(kv[0])) {
		case "https":
			httpsProxy = strings.TrimSpace(kv[1])
		case "http":
			httpProxy = strings.TrimSpace(kv[1])
		}
	}
	if httpsProxy != "" {
		return normalizeProxyHost(httpsProxy)
	}
	return normalizeProxyHost(httpProxy)
}

func normalizeProxyHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	u, err := url.Parse(host)
	if err != nil || u.Host == "" {
		return ""
	}
	return "http://" + u.Host
}
