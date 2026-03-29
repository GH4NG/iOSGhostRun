package services

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/amfi"
	"github.com/danielpaulus/go-ios/ios/imagemounter"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// MountImage 挂载镜像
func MountImage(udid string) error {
	device, _, err := GetDeviceAndVersion(udid)
	if err != nil {
		return err
	}

	// 检查并尝试启用开发者模式
	if enabled, err := imagemounter.IsDevModeEnabled(device); err == nil && !enabled {
		Log.Warn("ImageService", "设备开发者模式未启用，尝试请求启用")
		err := amfi.EnableDeveloperMode(device, true)
		if err != nil {
			if strings.Contains(err.Error(), "Developer Mode menu has been revealed in Settings") {
				msg := "开发者模式菜单已显示，请在 设置 → 隐私与安全性 中启用开发者模式，然后重试"
				Log.Info("ImageService", msg)
				application.Get().Event.Emit("developer-mode-menu-revealed", msg)
				return fmt.Errorf("%s", msg)
			}
			Log.Error("ImageService", fmt.Sprintf("启用开发者模式失败: %v", err))
			return err
		}
		Log.Info("ImageService", "已请求启用开发者模式，请在设备上确认并重试挂载")
		return nil
	}

	if IsVersionAbove17(udid) {
		tunnelDevice, err := getTunnelDevice(udid)
		if err != nil {
			return fmt.Errorf("获取隧道设备失败: %w", err)
		}
		return mountPersonalizedImage(*tunnelDevice)
	}
	return mountDeveloperImage(device)
}

// UnmountImage 卸载镜像
func UnmountImage(udid string) error {
	return unmountDeveloperImage(udid)
}

// mountPersonalizedImage 挂载私人镜像 (iOS >= 17)
func mountPersonalizedImage(device ios.DeviceEntry) error {
	vals, err := ios.GetValues(device)
	if err != nil {
		return fmt.Errorf("获取设备信息失败: %w", err)
	}

	ver, err := semver.NewVersion(vals.Value.ProductVersion)
	if err != nil {
		return fmt.Errorf("解析系统版本失败: %w", err)
	}

	imagePath, err := downloadPersonalizedImage(ResolveAppDir("devimages"))
	if err != nil {
		return fmt.Errorf("下载开发者镜像失败: %w", err)
	}

	pm, err := imagemounter.NewPersonalizedDeveloperDiskImageMounter(device, ver)
	if err != nil {
		return fmt.Errorf("创建 personalized mounter 失败: %w", err)
	}
	defer pm.Close()

	if sigs, err := pm.ListImages(); err == nil && len(sigs) > 0 {
		Log.Info("ImageService", "开发者镜像已挂载 (personalized)，跳过")
		return nil
	}

	if err := pm.MountImage(imagePath); err != nil {
		return fmt.Errorf("挂载 personalized 镜像失败: %w", err)
	}

	Log.Info("ImageService", "personalized 开发者镜像挂载成功")
	return nil
}

// mountDeveloperImage 挂载开发者镜像
func mountDeveloperImage(device ios.DeviceEntry) error {
	vals, err := ios.GetValues(device)
	if err != nil {
		return fmt.Errorf("获取设备信息失败: %w", err)
	}

	imagePath, err := downloadDeveloperImage(ResolveAppDir("devimages"), vals.Value.ProductVersion)
	if err != nil {
		return fmt.Errorf("准备开发者镜像失败: %w", err)
	}

	if err := imagemounter.MountImage(device, imagePath); err != nil {
		return fmt.Errorf("挂载开发者镜像失败: %w", err)
	}

	Log.Info("ImageService", "开发者镜像挂载成功")
	return nil
}

// unmountDeveloperImage 卸载开发者镜像
func unmountDeveloperImage(udid string) error {
	device, ver, err := GetDeviceAndVersion(udid)
	if err != nil {
		return err
	}

	mounter, err := imagemounter.NewDeveloperDiskImageMounter(device, ver)
	if err != nil {
		return fmt.Errorf("连接镜像服务失败: %w", err)
	}
	defer mounter.Close()

	if err := mounter.UnmountImage(); err != nil {
		return fmt.Errorf("卸载开发者镜像失败: %w", err)
	}

	Log.Info("ImageService", "开发者镜像卸载成功")
	return nil
}
