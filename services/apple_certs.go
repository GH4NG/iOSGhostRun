package services

import (
	"crypto/tls"
	"crypto/x509"
	"embed"
	"log"
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

	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}

	files, err := appleRootCerts.ReadDir("certs")
	if err != nil {
		log.Printf("[AppleCerts] 读取内置根证书目录失败: %v", err)
		return
	}
	for _, f := range files {
		pemBytes, err := appleRootCerts.ReadFile("certs/" + f.Name())
		if err != nil {
			log.Printf("[AppleCerts] 读取内置根证书 %s 失败: %v", f.Name(), err)
			continue
		}
		if !pool.AppendCertsFromPEM(pemBytes) {
			log.Printf("[AppleCerts] 内置根证书 %s 解析失败", f.Name())
		}
	}

	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		log.Printf("[AppleCerts] DefaultTransport 类型异常，跳过根证书注入")
		return
	}
	tr.TLSClientConfig = &tls.Config{RootCAs: pool}
}
