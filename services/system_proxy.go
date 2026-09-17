package services

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	proxyOnce sync.Once
	proxyURL  atomic.Pointer[url.URL]
)

// applySystemProxy 检测系统代理
func applySystemProxy() {
	proxyOnce.Do(installProxyHook)

	if os.Getenv("HTTPS_PROXY") != "" || os.Getenv("https_proxy") != "" ||
		os.Getenv("HTTP_PROXY") != "" || os.Getenv("http_proxy") != "" {
		Log.Info("ImageService", "已通过环境变量配置代理，跳过系统代理检测")
		return
	}

	proxy, err := detectSystemProxy()
	if err != nil {
		Log.Warn("ImageService", fmt.Sprintf("检测系统代理失败: %v", err))
		return
	}
	if proxy == "" {
		Log.Info("ImageService", "未检测到系统代理，直连下载")
		return
	}

	u, err := url.Parse(proxy)
	if err != nil || u.Host == "" {
		Log.Warn("ImageService", fmt.Sprintf("系统代理地址无效: %s", proxy))
		return
	}

	proxyURL.Store(u)
	_ = os.Setenv("HTTP_PROXY", proxy)
	_ = os.Setenv("HTTPS_PROXY", proxy)
	_ = os.Setenv("http_proxy", proxy)
	_ = os.Setenv("https_proxy", proxy)
	Log.Info("ImageService", fmt.Sprintf("检测到系统代理 %s，镜像下载将使用该代理", proxy))
}

func detectSystemProxy() (string, error) {
	switch runtime.GOOS {
	case "windows":
		key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

		enable, err := readRegValue(key, "ProxyEnable")
		if err != nil {
			return "", nil
		}
		if !strings.EqualFold(strings.TrimSpace(enable), "0x1") {
			return "", nil
		}

		server, err := readRegValue(key, "ProxyServer")
		if err != nil {
			return "", fmt.Errorf("读取 ProxyServer 失败: %w", err)
		}
		return parseProxyServer(server), nil
	case "darwin":
		out, err := exec.Command("scutil", "--proxy").Output()
		if err != nil {
			return "", fmt.Errorf("执行 scutil --proxy 失败: %w", err)
		}

		fields := map[string]string{}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if idx := strings.Index(line, ":"); idx > 0 {
				fields[strings.TrimSpace(line[:idx])] = strings.TrimSpace(line[idx+1:])
			}
		}

		if fields["HTTPSEnable"] == "1" {
			if p := normalizeProxyHost(fields["HTTPSProxy"] + ":" + fields["HTTPSPort"]); p != "" {
				return p, nil
			}
		}
		if fields["HTTPEnable"] == "1" {
			if p := normalizeProxyHost(fields["HTTPProxy"] + ":" + fields["HTTPPort"]); p != "" {
				return p, nil
			}
		}
		return "", nil
	default:
		return "", nil
	}
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
	return "", fmt.Errorf("reg query 输出中未找到 %s", name)
}

func installProxyHook() {
	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		Log.Warn("ImageService", "DefaultTransport 类型异常，无法应用系统代理")
		return
	}
	tr.Proxy = func(req *http.Request) (*url.URL, error) {
		if u := proxyURL.Load(); u != nil {
			return u, nil
		}
		return http.ProxyFromEnvironment(req)
	}
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
