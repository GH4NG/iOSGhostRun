package services

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	wintunDownloadURL   = "https://www.wintun.net/builds/wintun-0.14.1.zip"
	wintunArchiveSHA256 = "07c256185d6ee3652e09fa55c0b673e2624b565e02c4b9091c79ca7d2f24ef51"
	maxWintunSize       = 16 << 20
)

// 官方 0.14.1 签名 DLL 的摘要；本地导入也执行相同校验。
var wintunDLLHashes = map[string]string{
	"amd64": "e5da8447dc2c320edc0fc52fa01885c103de8c118481f683643cacc3220dafce",
	"arm64": "f7ba89005544be9d85231a9e0d5f23b2d15b3311667e2dad0debd344918a3f80",
	"386":   "d694fa46ab4cfebcb2632d094c7aa97278eef2f8052438621766d863ae98a931",
	"arm":   "daad267411ecdc70a0535e274d2c3e9da3d0084bdac7662cb8424dd4a031b4d9",
}

type WintunStatus struct {
	Supported bool   `json:"supported"`
	Installed bool   `json:"installed"`
	Arch      string `json:"arch"`
	Path      string `json:"path"`
	Message   string `json:"message"`
}

func (s *SettingsService) GetWintunStatus() WintunStatus {
	status := WintunStatus{Supported: runtime.GOOS == "windows", Arch: runtime.GOARCH}
	if !status.Supported {
		status.Message = "当前系统无需安装 Wintun，开发者镜像会在选择设备时自动下载。"
		return status
	}
	path, err := findWintunDLL()
	status.Path = path
	status.Installed = err == nil
	if err != nil {
		status.Message = err.Error()
	} else {
		status.Message = "Wintun 已就绪，可以连接 iOS 17 及以上设备。"
	}
	return status
}

func (s *SettingsService) DownloadAndInstallWintun() (WintunStatus, error) {
	if runtime.GOOS != "windows" {
		return s.GetWintunStatus(), fmt.Errorf("当前系统无需安装 Wintun")
	}
	if !s.runtimeMu.TryLock() {
		return WintunStatus{}, fmt.Errorf("正在安装 Wintun，请稍候")
	}
	defer s.runtimeMu.Unlock()
	if status := s.GetWintunStatus(); status.Installed {
		return status, nil
	}
	Log.Info("Settings", "正在从 Wintun 官网下载运行库")
	client := newDownloadHTTPClient(2 * time.Minute)
	defer client.CloseIdleConnections()
	response, err := client.Get(wintunDownloadURL)
	if err != nil {
		return WintunStatus{}, fmt.Errorf("下载 Wintun 失败，可重试或导入本地文件: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return WintunStatus{}, fmt.Errorf("下载 Wintun 失败: HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxWintunSize+1))
	if err != nil {
		return WintunStatus{}, fmt.Errorf("读取 Wintun 安装包失败: %w", err)
	}
	if err := installWintunFile("wintun.zip", data, s.wintunLicense); err != nil {
		return WintunStatus{}, err
	}
	Log.Info("Settings", "Wintun 下载、校验和安装完成")
	return s.GetWintunStatus(), nil
}

// encodedData 来自用户通过系统文件选择器选中的文件。
func (s *SettingsService) ImportWintun(filename, encodedData string) (WintunStatus, error) {
	if runtime.GOOS != "windows" {
		return s.GetWintunStatus(), fmt.Errorf("当前系统无需安装 Wintun")
	}
	if !s.runtimeMu.TryLock() {
		return WintunStatus{}, fmt.Errorf("正在安装 Wintun，请稍候")
	}
	defer s.runtimeMu.Unlock()
	if len(encodedData) > base64.StdEncoding.EncodedLen(maxWintunSize) {
		return WintunStatus{}, fmt.Errorf("Wintun 文件不能超过 16 MB")
	}
	data, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		return WintunStatus{}, fmt.Errorf("读取导入文件失败: %w", err)
	}
	if err := installWintunFile(filename, data, s.wintunLicense); err != nil {
		return WintunStatus{}, err
	}
	Log.Info("Settings", "Wintun 本地文件校验和导入完成")
	return s.GetWintunStatus(), nil
}

func installWintunFile(filename string, data, fallbackLicense []byte) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序目录失败: %w", err)
	}
	return installWintunToDirectory(filename, data, fallbackLicense, runtime.GOARCH, filepath.Dir(exe))
}

func installWintunToDirectory(filename string, data, fallbackLicense []byte, arch, dir string) error {
	dll, license, err := readWintunFile(filename, data, arch)
	if err != nil {
		return err
	}
	if len(license) == 0 {
		license = fallbackLicense
	}
	if len(license) != 0 {
		if err := writeFileAtomically(filepath.Join(dir, "WINTUN-LICENSE.txt"), license, 0644); err != nil {
			return fmt.Errorf("保存 Wintun 许可证失败，请将程序移到可写目录: %w", err)
		}
	}
	path := filepath.Join(dir, "wintun.dll")
	// 已存在相同的文件时无需替换，兼容 DLL 已加载的情况。
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, dll) {
		return nil
	}
	if err := writeFileAtomically(path, dll, 0644); err != nil {
		return fmt.Errorf("安装 Wintun 失败，请将程序移到可写目录；若运行库正在使用，请重启后导入: %w", err)
	}
	return nil
}

func readWintunFile(filename string, data []byte, arch string) ([]byte, []byte, error) {
	if len(data) == 0 || len(data) > maxWintunSize {
		return nil, nil, fmt.Errorf("Wintun 文件为空或超过 16 MB")
	}
	var dll, license []byte
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".zip":
		if fmt.Sprintf("%x", sha256.Sum256(data)) != wintunArchiveSHA256 {
			return nil, nil, fmt.Errorf("安装包 SHA-256 校验失败，请使用 Wintun 官网的 0.14.1 ZIP 安装包")
		}
		archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, nil, fmt.Errorf("解压 Wintun 失败: %w", err)
		}
		zipArch := arch
		if arch == "386" {
			zipArch = "x86"
		}
		for _, file := range archive.File {
			if file.Name != "wintun/bin/"+zipArch+"/wintun.dll" && file.Name != "wintun/LICENSE.txt" {
				continue
			}
			reader, err := file.Open()
			if err != nil {
				return nil, nil, err
			}
			contents, readErr := io.ReadAll(io.LimitReader(reader, maxWintunSize+1))
			reader.Close()
			if readErr != nil || len(contents) > maxWintunSize {
				return nil, nil, fmt.Errorf("读取 Wintun 压缩包内容失败")
			}
			if strings.HasSuffix(file.Name, ".dll") {
				dll = contents
			} else {
				license = contents
			}
		}
	case ".dll":
		dll = data
	default:
		return nil, nil, fmt.Errorf("请选择官方 Wintun ZIP 安装包或 wintun.dll")
	}
	if err := validateWintunDLL(dll, arch); err != nil {
		return nil, nil, err
	}
	expectedHash, ok := wintunDLLHashes[arch]
	if !ok || fmt.Sprintf("%x", sha256.Sum256(dll)) != expectedHash {
		return nil, nil, fmt.Errorf("DLL SHA-256 校验失败，请使用官方 Wintun 0.14.1 的 %s 版本", arch)
	}
	return dll, license, nil
}

func validateWintunDLL(data []byte, arch string) error {
	machine := map[string]uint16{"amd64": pe.IMAGE_FILE_MACHINE_AMD64, "arm64": pe.IMAGE_FILE_MACHINE_ARM64, "386": pe.IMAGE_FILE_MACHINE_I386, "arm": pe.IMAGE_FILE_MACHINE_ARMNT}[arch]
	file, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("运行库不是有效的 Windows DLL")
	}
	defer file.Close()
	if machine == 0 || file.Machine != machine || file.Characteristics&pe.IMAGE_FILE_DLL == 0 {
		return fmt.Errorf("运行库架构不匹配，请选择 %s 版本的 wintun.dll", arch)
	}
	return nil
}

func findWintunDLL() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// 与 wintun-go 的 DLL 搜索顺序保持一致：程序目录优先于 System32。
	paths := []string{filepath.Join(filepath.Dir(exe), "wintun.dll")}
	if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
		paths = append(paths, filepath.Join(systemRoot, "System32", "wintun.dll"))
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return path, fmt.Errorf("读取运行库失败: %w", err)
		}
		if err := validateWintunDLL(data, runtime.GOARCH); err != nil {
			return path, err
		}
		return path, nil
	}
	return paths[0], fmt.Errorf("尚未安装 Wintun，请点击一键下载安装或导入本地文件")
}
