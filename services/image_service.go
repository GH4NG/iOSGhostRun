package services

import (
	"fmt"
	"strings"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/amfi"
	"github.com/danielpaulus/go-ios/ios/imagemounter"
)

// MountImage 挂载镜像
func MountImage(udid string) error {
	device, version, err := GetDeviceAndVersion(udid)
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
				emitAppEvent("developer-mode-menu-revealed", msg)
				return fmt.Errorf("%s", msg)
			}
			Log.Error("ImageService", fmt.Sprintf("启用开发者模式失败: %v", err))
			return err
		}
		msg := "已请求启用开发者模式，请在设备上确认并重试挂载"
		Log.Info("ImageService", msg)
		emitAppEvent("developer-mode-menu-revealed", msg)
		return fmt.Errorf("%s", msg)
	}

	if version.Major() >= 17 {
		tunnelDevice, err := getTunnelDevice(udid)
		if err != nil {
			return fmt.Errorf("获取隧道设备失败: %w", err)
		}
		return mountDeveloperImage(*tunnelDevice)
	}
	return mountDeveloperImage(device)
}

// UnmountImage 卸载镜像。重复清理时，没有已挂载镜像视为成功。
func UnmountImage(udid string) error {
	device, version, err := GetDeviceAndVersion(udid)
	if err != nil {
		return err
	}

	if version.Major() >= 17 {
		tunnelDevice, err := getTunnelDevice(udid)
		if err != nil {
			return fmt.Errorf("获取隧道设备失败: %w", err)
		}
		device = *tunnelDevice
	}

	// 统一入口会同时处理 Cryptex 和 Personalized 镜像。
	mounter, err := imagemounter.NewImageMounter(device)
	if err != nil {
		return fmt.Errorf("连接镜像服务失败: %w", err)
	}
	defer mounter.Close()

	images, err := mounter.ListImages()
	if err != nil {
		return fmt.Errorf("查询已挂载镜像失败: %w", err)
	}
	if len(images) == 0 {
		return nil
	}
	if err := mounter.UnmountImage(); err != nil {
		return fmt.Errorf("卸载开发者镜像失败: %w", err)
	}

	Log.Info("ImageService", "开发者镜像卸载成功")
	return nil
}

// mountDeveloperImage 挂载开发者镜像
func mountDeveloperImage(device ios.DeviceEntry) error {
	imagePath, err := imagemounter.DownloadImageFor(device, ResolveAppDir("devimages"))
	if err != nil {
		return fmt.Errorf("获取开发者镜像失败: %w", err)
	}

	// 由上游检查镜像是否一致，仅替换不同镜像，并自动选择 Cryptex 或 Personalized。
	if err := imagemounter.MountImage(device, imagePath); err != nil {
		return fmt.Errorf("挂载开发者镜像失败: %w", err)
	}

	Log.Info("ImageService", "开发者镜像挂载成功")
	return nil
}
