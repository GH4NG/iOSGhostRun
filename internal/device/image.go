package device

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/amfi"
	"github.com/danielpaulus/go-ios/ios/imagemounter"
)

var (
	ErrNoDeviceSelected = errors.New("未选择设备")
	ErrImageNotMounted  = errors.New("开发者镜像未挂载")
	ErrDevModeDisabled  = errors.New("开发者模式未启用")
)

// GitHub 镜像代理列表
var githubMirrors = []string{
	"",                            // 原始 GitHub
	"https://ghfast.top/",         // 镜像1
	"https://gh-proxy.com/",       // 镜像2
	"https://mirror.ghproxy.com/", // 镜像3
}

const (
	imageFile     = "DeveloperDiskImage.dmg"
	signatureFile = "DeveloperDiskImage.dmg.signature"
	deviceboxURL  = "https://deviceboxhq.com/"
	xcode15_4_ddi = "ddi-15F31d"
)

// getDeviceContext 获取设备上下文
func (m *Manager) getDeviceContext() (*ios.DeviceEntry, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.deviceEntry == nil {
		return nil, "", ErrNoDeviceSelected
	}
	return m.deviceEntry, m.imageBasedir, nil
}

// newImageMounter 创建镜像挂载器
func (m *Manager) newImageMounter(deviceEntry ios.DeviceEntry) (imagemounter.ImageMounter, error) {
	mounter, err := imagemounter.NewImageMounter(deviceEntry)
	if err != nil {
		return nil, fmt.Errorf("连接镜像服务失败: %w", err)
	}
	return mounter, nil
}

// CheckDevModeEnabled 检查开发者模式是否启用
func (m *Manager) CheckDevModeEnabled() (bool, error) {
	deviceEntry, _, err := m.getDeviceContext()
	if err != nil {
		return false, err
	}

	enabled, err := imagemounter.IsDevModeEnabled(*deviceEntry)
	if err != nil {
		m.log.Debug("设备管理", "检查开发者模式失败: %v", err)
		return true, nil
	}

	if enabled {
		m.log.Debug("设备管理", "开发者模式已启用")
	} else {
		m.log.Warn("设备管理", "开发者模式未启用，正在尝试自动开启...")
		if err := m.EnableDevMode(); err != nil {
			m.log.Error("设备管理", "自动开启开发者模式失败: %v", err)
			return false, err
		}
		m.log.Info("设备管理", "开发者模式已开启，请在设备上确认并重启设备")
	}

	return enabled, nil
}

// EnableDevMode 启用开发者模式
func (m *Manager) EnableDevMode() error {
	deviceEntry, _, err := m.getDeviceContext()
	if err != nil {
		return err
	}

	m.log.Info("设备管理", "正在启用开发者模式...")

	// enablePostRestart=true 表示设备重启后自动完成启用
	if err := amfi.EnableDeveloperMode(*deviceEntry, true); err != nil {
		return fmt.Errorf("启用开发者模式失败: %w", err)
	}

	m.log.Info("设备管理", "开发者模式启用请求已发送，请在设备上确认并重启")
	return nil
}

// CheckDeveloperImage 检查开发者镜像是否已挂载
func (m *Manager) CheckDeveloperImage() (bool, error) {
	deviceEntry, _, err := m.getDeviceContext()
	if err != nil {
		return false, err
	}

	m.log.Debug("设备管理", "检查开发者镜像挂载状态...")

	mounter, err := m.newImageMounter(*deviceEntry)
	if err != nil {
		m.log.Error("设备管理", "%v", err)
		return false, err
	}
	defer mounter.Close()

	signatures, err := mounter.ListImages()
	if err != nil {
		m.log.Error("设备管理", "获取镜像列表失败: %v", err)
		return false, fmt.Errorf("获取镜像列表失败: %w", err)
	}

	mounted := len(signatures) > 0
	if mounted {
		m.log.Info("设备管理", "开发者镜像已挂载")
	} else {
		m.log.Info("设备管理", "开发者镜像未挂载")
	}

	return mounted, nil
}

// MountDeveloperImage 挂载开发者镜像
func (m *Manager) MountDeveloperImage() error {
	deviceEntry, basedir, err := m.getDeviceContext()
	if err != nil {
		return err
	}

	m.log.Info("设备管理", "开始挂载开发者镜像...")

	// 检查并自动启用开发者模式
	if enabled, err := imagemounter.IsDevModeEnabled(*deviceEntry); err == nil && !enabled {
		m.log.Warn("设备管理", "开发者模式未启用，正在尝试自动开启...")
		if err := amfi.EnableDeveloperMode(*deviceEntry, true); err != nil {
			m.log.Error("设备管理", "自动开启开发者模式失败: %v", err)
			return ErrDevModeDisabled
		}
		m.log.Info("设备管理", "开发者模式启用请求已发送，请在设备上确认并重启后重试")
		return ErrDevModeDisabled
	}

	mounter, err := m.newImageMounter(*deviceEntry)
	if err != nil {
		m.log.Error("设备管理", "%v", err)
		return err
	}

	signatures, err := mounter.ListImages()
	if err != nil {
		mounter.Close()
		m.log.Error("设备管理", "获取镜像列表失败: %v", err)
		return fmt.Errorf("获取镜像列表失败: %w", err)
	}

	if len(signatures) > 0 {
		mounter.Close()
		m.log.Info("设备管理", "开发者镜像已挂载，跳过")
		return nil
	}

	// 下载镜像
	m.log.Info("设备管理", "正在下载开发者镜像，目录: %s", basedir)
	imagePath, err := m.downloadDeveloperImage(*deviceEntry, basedir)
	if err != nil {
		mounter.Close()
		m.log.Error("设备管理", "下载开发者镜像失败: %v", err)
		return fmt.Errorf("下载开发者镜像失败: %w", err)
	}
	m.log.Info("设备管理", "下载完成，镜像路径: %s", imagePath)

	// 挂载镜像
	m.log.Info("设备管理", "正在挂载开发者镜像...")
	if err := mounter.MountImage(imagePath); err != nil {
		mounter.Close()
		m.log.Error("设备管理", "挂载开发者镜像失败: %v", err)
		return fmt.Errorf("挂载开发者镜像失败: %w", err)
	}

	mounter.Close()
	m.log.Info("设备管理", "开发者镜像挂载成功！")
	return nil
}

// UnMountDeveloperImage 卸载开发者镜像
func (m *Manager) UnMountDeveloperImage() error {
	deviceEntry, _, err := m.getDeviceContext()
	if err != nil {
		return err
	}

	m.log.Info("设备管理", "正在卸载开发者镜像...")

	mounter, err := m.newImageMounter(*deviceEntry)
	if err != nil {
		m.log.Error("设备管理", "%v", err)
		return err
	}
	defer mounter.Close()

	if err := mounter.UnmountImage(); err != nil {
		m.log.Error("设备管理", "卸载开发者镜像失败: %v", err)
		return fmt.Errorf("卸载开发者镜像失败: %w", err)
	}

	m.log.Info("设备管理", "开发者镜像卸载成功！")
	return nil
}

// EnsureDeveloperImage 确保开发者镜像已挂载
func (m *Manager) EnsureDeveloperImage() error {
	mounted, err := m.CheckDeveloperImage()
	if err != nil {
		return err
	}

	if !mounted {
		return m.MountDeveloperImage()
	}

	return nil
}

// downloadDeveloperImage 下载开发者镜像
func (m *Manager) downloadDeveloperImage(device ios.DeviceEntry, basedir string) (string, error) {
	// 获取设备版本
	allValues, err := ios.GetValues(device)
	if err != nil {
		return "", fmt.Errorf("获取设备信息失败: %w", err)
	}

	parsedVersion, err := semver.NewVersion(allValues.Value.ProductVersion)
	if err != nil {
		return "", fmt.Errorf("解析版本失败: %w", err)
	}

	// iOS 17+ 使用 devicebox
	if parsedVersion.GreaterThan(semver.MustParse("17.0.0")) || parsedVersion.Equal(semver.MustParse("17.0.0")) {
		return m.download17Plus(basedir)
	}

	// iOS 16 及以下使用 GitHub
	return m.downloadLegacyImage(basedir, allValues.Value.ProductVersion)
}

// download17Plus 下载 iOS 17+ 镜像
func (m *Manager) download17Plus(basedir string) (string, error) {
	// 检查是否已下载
	extractedPath := path.Join(basedir, xcode15_4_ddi)
	restorePath := path.Join(extractedPath, "Restore")
	if _, err := os.Stat(restorePath); err == nil {
		m.log.Info("设备管理", "使用已下载的镜像: %s", restorePath)
		return restorePath, nil
	}

	// iOS 17+ 从 devicebox 下载
	downloadURL := deviceboxURL + xcode15_4_ddi + ".zip"
	zipPath := path.Join(basedir, xcode15_4_ddi+".zip")

	m.log.Info("设备管理", "下载 iOS 17+ 镜像: %s", downloadURL)
	if err := m.downloadFile(zipPath, downloadURL); err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}

	// 解压
	m.log.Info("设备管理", "解压镜像...")
	if _, _, err := ios.Unzip(zipPath, extractedPath); err != nil {
		return "", fmt.Errorf("解压失败: %w", err)
	}

	return restorePath, nil
}

// downloadLegacyImage 下载 iOS 16 及以下镜像
func (m *Manager) downloadLegacyImage(basedir string, productVersion string) (string, error) {
	version := imagemounter.MatchAvailable(productVersion)
	versionDir := strings.Split(version, " (")[0]
	imageDir := path.Join(basedir, versionDir)
	imagePath := path.Join(imageDir, imageFile)
	signaturePath := path.Join(imageDir, signatureFile)

	// 检查是否已下载
	if _, err := os.Stat(imagePath); err == nil {
		if _, err := os.Stat(signaturePath); err == nil {
			m.log.Info("设备管理", "使用已下载的镜像: %s", imagePath)
			return imagePath, nil
		}
	}

	// 创建目录
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 构建原始 URL
	baseURL := fmt.Sprintf("https://github.com/mspvirajpatel/Xcode_Developer_Disk_Images/raw/master/Developer%%20Disk%%20Image/%s", version)
	imageURL := baseURL + "/" + imageFile
	signatureURL := baseURL + "/" + signatureFile

	// 尝试使用不同镜像下载
	var lastErr error
	for _, mirror := range githubMirrors {
		mirrorImageURL := mirror + imageURL
		mirrorSignatureURL := mirror + signatureURL

		if mirror == "" {
			m.log.Info("设备管理", "尝试从 GitHub 原始地址下载...")
		} else {
			m.log.Info("设备管理", "尝试从镜像站下载: %s", mirror)
		}

		// 下载镜像文件
		if err := m.downloadFile(imagePath, mirrorImageURL); err != nil {
			lastErr = err
			m.log.Warn("设备管理", "下载镜像失败: %v,尝试下一个镜像...", err)
			continue
		}

		// 下载签名文件
		if err := m.downloadFile(signaturePath, mirrorSignatureURL); err != nil {
			lastErr = err
			os.Remove(imagePath) // 清理已下载的镜像
			m.log.Warn("设备管理", "下载签名失败: %v,尝试下一个镜像...", err)
			continue
		}

		m.log.Info("设备管理", "下载成功!")
		return imagePath, nil
	}

	return "", fmt.Errorf("所有镜像下载失败: %w", lastErr)
}

// downloadFile 下载文件
func (m *Manager) downloadFile(filepath string, url string) error {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
