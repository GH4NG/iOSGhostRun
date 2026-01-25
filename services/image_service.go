package services

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/amfi"
	"github.com/danielpaulus/go-ios/ios/imagemounter"
)

type ImageService struct{}

// MountImage 挂载镜像
func (i *ImageService) MountImage(udid string) error {
	device, err := ios.GetDevice(udid)
	if err != nil {
		return fmt.Errorf("获取设备失败: %w", err)
	}

	// 检查并尝试启用开发者模式
	if enabled, err := imagemounter.IsDevModeEnabled(device); err == nil && !enabled {
		Log.Warn("ImageService", "设备开发者模式未启用，尝试请求启用")
		if err := amfi.EnableDeveloperMode(device, true); err != nil {
			Log.Error("ImageService", fmt.Sprintf("启用开发者模式失败: %v", err))
			return fmt.Errorf("启用开发者模式失败: %w", err)
		}
		Log.Info("ImageService", "已请求启用开发者模式，请在设备上确认并重试挂载")
		return nil
	}

	ds := &DevicesService{}
	isAbove, _ := ds.IsVersionAbove17(udid)

	if isAbove {
		// 挂载私人镜像 PersonalizedImages
	}

	//  挂载开发者镜像 DeveloperDiskImage
	return i.mountDeveloperImage(device)
}

// UnmountImage 卸载镜像
func (i *ImageService) UnmountImage(udid string) error {
	ds := &DevicesService{}
	isAbove, _ := ds.IsVersionAbove17(udid)

	if isAbove {
		// 卸载私人镜像 PersonalizedImages
	}
	return i.unmountDeveloperImage(udid)
}

// mountPersonalizedImage 挂载私人镜像
func (i *ImageService) mountPersonalizedImage(device ios.DeviceEntry) error {
	imageDir := ResolveAppDir("devimages")
	Log.Info("ImageService", fmt.Sprintf("下载开发者镜像目录: %s", imageDir))
	dl := &ImageDownloadService{}
	imagePath, err := dl.downloadPersonalizedImage(imageDir)
	if err != nil {
		Log.Error("ImageService", fmt.Sprintf("下载开发者镜像失败: %v", err))
		return fmt.Errorf("下载开发者镜像失败: %w", err)
	}
	Log.Info("ImageService", fmt.Sprintf("下载完成，镜像路径: %s", imagePath))
	Log.Info("ImageService", "使用 PersonalizedDeveloperDiskImageMounter 进行挂载 (iOS >= 17)")

	// 获取设备版本
	vals, err := ios.GetValues(device)
	if err != nil {
		return fmt.Errorf("获取设备信息失败: %w", err)
	}
	pv := vals.Value.ProductVersion
	ver, err := semver.NewVersion(pv)
	if err != nil {
		return fmt.Errorf("解析系统版本失败: %w", err)
	}

	pm, err := imagemounter.NewPersonalizedDeveloperDiskImageMounter(device, ver)
	if err != nil {
		Log.Error("ImageService", fmt.Sprintf("创建 personalized mounter 失败: %v", err))
		return fmt.Errorf("创建 personalized mounter 失败: %w", err)
	}
	defer pm.Close()

	if sigs, err := pm.ListImages(); err == nil && len(sigs) > 0 {
		Log.Info("ImageService", "开发者镜像已挂载 (personalized)，跳过")
		return nil
	}

	Log.Info("ImageService", "正在挂载 personalized 开发者镜像...")
	if err := pm.MountImage(imagePath); err != nil {
		Log.Error("ImageService", fmt.Sprintf("挂载 personalized 镜像失败: %v", err))
		return fmt.Errorf("挂载 personalized 镜像失败: %w", err)
	}

	Log.Info("ImageService", "personalized 开发者镜像挂载成功")
	return nil
}

// mountDeveloperImage 挂载开发者镜像
func (i *ImageService) mountDeveloperImage(device ios.DeviceEntry) error {
	imageDir := ResolveAppDir("devimages")

	// 获取设备版本以选择合适的镜像
	vals, err := ios.GetValues(device)
	if err != nil {
		return fmt.Errorf("获取设备信息失败: %w", err)
	}
	productVersion := vals.Value.ProductVersion

	// 确保使用具体的镜像文件路径，而不是目录
	dl := &ImageDownloadService{}
	imagePath, err := dl.downloadDeveloperImage(imageDir, productVersion)
	if err != nil {
		Log.Error("ImageService", fmt.Sprintf("准备开发者镜像失败: %v", err))
		return fmt.Errorf("准备开发者镜像失败: %w", err)
	}

	if err := imagemounter.MountImage(device, imagePath); err != nil {
		Log.Error("ImageService", fmt.Sprintf("挂载开发者镜像失败: %v", err))
		return fmt.Errorf("挂载开发者镜像失败: %w", err)
	}

	Log.Info("ImageService", "开发者镜像挂载成功")
	return nil
}

// unmountDeveloperImage 卸载开发者镜像
func (i *ImageService) unmountDeveloperImage(udid string) error {
	device, err := ios.GetDevice(udid)
	if err != nil {
		return fmt.Errorf("获取设备失败: %w", err)
	}

	vals, err := ios.GetValues(device)
	if err != nil {
		return fmt.Errorf("获取设备信息失败: %w", err)
	}
	pv := vals.Value.ProductVersion
	ver, err := semver.NewVersion(pv)
	if err != nil {
		return fmt.Errorf("解析系统版本失败: %w", err)
	}
	mounter, err := imagemounter.NewDeveloperDiskImageMounter(device, ver)
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

	vals, err := ios.GetValues(device)
	if err != nil {
		return false, fmt.Errorf("获取设备信息失败: %w", err)
	}
	pv := vals.Value.ProductVersion
	ver, err := semver.NewVersion(pv)
	if err != nil {
		return false, fmt.Errorf("解析系统版本失败: %w", err)
	}
	mounter, err := imagemounter.NewDeveloperDiskImageMounter(device, ver)
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
