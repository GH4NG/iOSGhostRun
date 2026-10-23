package services

import (
	"crypto/tls"
	"crypto/x509"
	"embed"
	"net/http"
	"runtime"
)

//go:embed certs/apple_root_ca.pem certs/apple_root_ca_g3.pem
var appleRootCerts embed.FS

func init() {
	installAppleRootCAs()
}

func installAppleRootCAs() {

	if runtime.GOOS == "darwin" {
		return
	}

	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		Log.Warn("AppleCerts", "DefaultTransport 类型异常，跳过根证书注入")
		return
	}
	tr = tr.Clone()

	if tr.TLSClientConfig == nil {
		tr.TLSClientConfig = &tls.Config{}
	}
	tlsConfig := tr.TLSClientConfig
	var pool *x509.CertPool
	if tlsConfig.RootCAs != nil {
		pool = tlsConfig.RootCAs.Clone()
	} else {
		var err error
		pool, err = x509.SystemCertPool()
		if err != nil {
			Log.Warn("AppleCerts", "读取系统根证书失败: "+err.Error())
		}
		if pool == nil {
			pool = x509.NewCertPool()
		}
	}

	files, err := appleRootCerts.ReadDir("certs")
	if err != nil {
		Log.Warn("AppleCerts", "读取内置根证书目录失败: "+err.Error())
		return
	}
	for _, f := range files {
		pemBytes, err := appleRootCerts.ReadFile("certs/" + f.Name())
		if err != nil {
			Log.Warn("AppleCerts", "读取内置根证书 "+f.Name()+" 失败: "+err.Error())
			continue
		}
		if !pool.AppendCertsFromPEM(pemBytes) {
			Log.Warn("AppleCerts", "内置根证书 "+f.Name()+" 解析失败")
		}
	}

	tlsConfig.RootCAs = pool
	tr.TLSClientConfig = tlsConfig
	http.DefaultTransport = tr
}
