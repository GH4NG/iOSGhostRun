package services

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/amfi"
	"github.com/danielpaulus/go-ios/ios/imagemounter"
)

var (
	githubMirrors = []string{
		"",
		"https://ghfast.top/",
		"https://gh-proxy.com/",
		"https://mirror.ghproxy.com/",
	}

	deviceboxURL  = "https://deviceboxhq.com/"
	imageFile     = "DeveloperDiskImage.dmg"
	signatureFile = "DeveloperDiskImage.dmg.signature"
	xcode15_4_ddi = "ddi-15F31d"
)

type ImageService struct{}

// MountDeveloperImage 挂载镜像
func (i *ImageService) MountDeveloperImage(udid string) error {
	device, err := ios.GetDevice(udid)
	if err != nil {
		return fmt.Errorf("获取设备失败: %w", err)
	}

	// 检查并尝试启用开发者模式
	if enabled, err := imagemounter.IsDevModeEnabled(device); err == nil && !enabled {
		Log.Warn("ImageService", "设备开发者模式未启用，尝试请求启用")
		if err := amfi.EnableDeveloperMode(device, true); err != nil {
			Log.Error("ImageService", fmt.Sprintf("启用开发者模式失败: %v", err))
			return fmt.Errorf("开发者模式未启用: %w", err)
		}
		Log.Info("ImageService", "已请求启用开发者模式，请在设备上确认并重试挂载")
		return fmt.Errorf("开发者模式未启用")
	}

	mounter, err := imagemounter.NewImageMounter(device)
	if err != nil {
		Log.Error("ImageService", fmt.Sprintf("连接镜像服务失败: %v", err))
		return fmt.Errorf("连接镜像服务失败: %w", err)
	}
	defer mounter.Close()

	sigs, err := mounter.ListImages()
	if err != nil {
		Log.Error("ImageService", fmt.Sprintf("获取镜像列表失败: %v", err))
		return fmt.Errorf("获取镜像列表失败: %w", err)
	}

	if len(sigs) > 0 {
		Log.Info("ImageService", "开发者镜像已挂载，跳过")
		return nil
	}

	basedir := filepath.Join("devimages")
	Log.Info("ImageService", fmt.Sprintf("下载开发者镜像目录: %s", basedir))
	imagePath, err := i.downloadDeveloperImage(device, basedir)
	if err != nil {
		Log.Error("ImageService", fmt.Sprintf("下载开发者镜像失败: %v", err))
		return fmt.Errorf("下载开发者镜像失败: %w", err)
	}

	Log.Info("ImageService", fmt.Sprintf("下载完成，镜像路径: %s", imagePath))

	Log.Info("ImageService", "正在挂载开发者镜像...")
	if err := mounter.MountImage(imagePath); err != nil {
		Log.Error("ImageService", fmt.Sprintf("挂载开发者镜像失败: %v", err))
		return fmt.Errorf("挂载开发者镜像失败: %w", err)
	}

	Log.Info("ImageService", "开发者镜像挂载成功")
	return nil
}

// UnmountDeveloperImage 卸载镜像
func (i *ImageService) UnmountDeveloperImage(udid string) error {
	device, err := ios.GetDevice(udid)
	if err != nil {
		return fmt.Errorf("获取设备失败: %w", err)
	}

	mounter, err := imagemounter.NewImageMounter(device)
	if err != nil {
		Log.Error("ImageService", fmt.Sprintf("连接镜像服务失败: %v", err))
		return fmt.Errorf("连接镜像服务失败: %w", err)
	}
	defer mounter.Close()

	if err := mounter.UnmountImage(); err != nil {
		Log.Error("ImageService", fmt.Sprintf("卸载开发者镜像失败: %v", err))
		return fmt.Errorf("卸载开发者镜像失败: %w", err)
	}

	Log.Info("ImageService", "开发者镜像卸载成功")
	return nil
}

// CheckDeveloperImage 检查镜像是否已挂载
func (i *ImageService) CheckDeveloperImage(udid string) (bool, error) {
	device, err := ios.GetDevice(udid)
	if err != nil {
		return false, fmt.Errorf("获取设备失败: %w", err)
	}

	mounter, err := imagemounter.NewImageMounter(device)
	if err != nil {
		Log.Error("ImageService", fmt.Sprintf("连接镜像服务失败: %v", err))
		return false, fmt.Errorf("连接镜像服务失败: %w", err)
	}
	defer mounter.Close()

	sigs, err := mounter.ListImages()
	if err != nil {
		Log.Error("ImageService", fmt.Sprintf("获取镜像列表失败: %v", err))
		return false, fmt.Errorf("获取镜像列表失败: %w", err)
	}

	mounted := len(sigs) > 0
	return mounted, nil
}

// downloadDeveloperImage 下载镜像
func (i *ImageService) downloadDeveloperImage(device ios.DeviceEntry, basedir string) (string, error) {
	allValues, err := ios.GetValues(device)
	if err != nil {
		return "", fmt.Errorf("获取设备信息失败: %w", err)
	}

	pv := allValues.Value.ProductVersion
	parsed, err := semver.NewVersion(pv)
	if err != nil {
		return "", fmt.Errorf("解析版本失败: %w", err)
	}

	if parsed.GreaterThan(semver.MustParse("17.0.0")) || parsed.Equal(semver.MustParse("17.0.0")) {
		return i.download17Plus(basedir)
	}

	return i.downloadLegacyImage(basedir, pv)
}

// download17Plus 下载 iOS 17+ 镜像
func (i *ImageService) download17Plus(basedir string) (string, error) {
	extractedPath := filepath.Join(basedir, xcode15_4_ddi)
	restorePath := filepath.Join(extractedPath, "Restore")
	if _, err := os.Stat(restorePath); err == nil {
		Log.Info("ImageService", fmt.Sprintf("使用已下载的镜像: %s", restorePath))
		return restorePath, nil
	}

	downloadURL := deviceboxURL + xcode15_4_ddi + ".zip"
	zipPath := filepath.Join(basedir, xcode15_4_ddi+".zip")

	Log.Info("ImageService", fmt.Sprintf("下载 iOS 17+ 镜像: %s", downloadURL))
	if err := i.downloadFile(zipPath, downloadURL); err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}

	Log.Info("ImageService", "解压镜像...")
	if _, _, err := ios.Unzip(zipPath, extractedPath); err != nil {
		return "", fmt.Errorf("解压失败: %w", err)
	}

	return restorePath, nil
}

// downloadLegacyImage 下载 iOS 16 及以下镜像
func (i *ImageService) downloadLegacyImage(basedir string, productVersion string) (string, error) {
	version := imagemounter.MatchAvailable(productVersion)
	versionDir := strings.Split(version, " (")[0]
	imageDir := filepath.Join(basedir, versionDir)
	imagePath := filepath.Join(imageDir, imageFile)
	signaturePath := filepath.Join(imageDir, signatureFile)

	if _, err := os.Stat(imagePath); err == nil {
		if _, err := os.Stat(signaturePath); err == nil {
			Log.Info("ImageService", fmt.Sprintf("使用已下载的镜像: %s", imagePath))
			return imagePath, nil
		}
	}

	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	baseURL := fmt.Sprintf("https://github.com/mspvirajpatel/Xcode_Developer_Disk_Images/raw/master/Developer%%20Disk%%20Image/%s", version)
	imageURL := baseURL + "/" + imageFile
	signatureURL := baseURL + "/" + signatureFile

	var lastErr error
	for _, mirror := range githubMirrors {
		mirrorImageURL := mirror + imageURL
		mirrorSignatureURL := mirror + signatureURL

		if mirror == "" {
			Log.Info("ImageService", "尝试从 GitHub 原始地址下载...")
		} else {
			Log.Info("ImageService", fmt.Sprintf("尝试从镜像站下载: %s", mirror))
		}

		if err := i.downloadFile(imagePath, mirrorImageURL); err != nil {
			lastErr = err
			Log.Warn("ImageService", fmt.Sprintf("下载镜像失败: %v, 尝试下一个镜像...", err))
			continue
		}

		if err := i.downloadFile(signaturePath, mirrorSignatureURL); err != nil {
			lastErr = err
			os.Remove(imagePath)
			Log.Warn("ImageService", fmt.Sprintf("下载签名失败: %v, 尝试下一个镜像...", err))
			continue
		}

		Log.Info("ImageService", "下载成功")
		return imagePath, nil
	}

	return "", fmt.Errorf("所有镜像下载失败: %w", lastErr)
}

// downloadFile 下载文件到本地
func (i *ImageService) downloadFile(filepathLocal string, url string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(filepathLocal)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
