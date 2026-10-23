package services

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
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
		return mountPersonalizedImage(*tunnelDevice)
	}
	return mountDeveloperImage(device)
}

// UnmountImage 卸载镜像
func UnmountImage(udid string) error {
	device, ver, err := GetDeviceAndVersion(udid)
	if err != nil {
		return err
	}

	if ver.Major() >= 17 {
		if err := unmountPersonalizedImage(device, ver); err != nil {
			return fmt.Errorf("卸载 personalized 开发者镜像失败: %w", err)
		}
		return nil
	}

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

	imagePath, err := imagemounter.DownloadImageFor(device, ResolveAppDir("devimages"))
	if err != nil {
		return fmt.Errorf("获取开发者镜像失败: %w", err)
	}

	// 同一挂载点只能存在一份镜像。Xcode 的 CoreDevice 会在设备接入时自动挂载
	// 个性化镜像并占用 /System/Developer，此时直接挂载会被设备拒绝
	// （ImageMountFailed: ... is already mounted at /System/Developer）。
	// 先尝试卸载旧镜像；个性化镜像在多数设备上无法卸载，卸载失败时退回复用现有镜像。
	if err := unmountPersonalizedImage(device, ver); err != nil {
		Log.Warn("ImageService", "卸载原有 personalized 镜像失败: "+err.Error())
	}

	pm, err := imagemounter.NewPersonalizedDeveloperDiskImageMounter(device, ver)
	if err != nil {
		return fmt.Errorf("创建 personalized mounter 失败: %w", err)
	}
	defer pm.Close()

	if sigs, err := pm.ListImages(); err == nil && len(sigs) > 0 {
		Log.Warn("ImageService", "镜像仍处于挂载状态，跳过重复挂载")
		return nil
	}

	if err := pm.MountImage(imagePath); err != nil {
		if isImageAlreadyMounted(err) {
			Log.Warn("ImageService", "设备上已存在挂载的 personalized 镜像，直接复用")
			return nil
		}
		return fmt.Errorf("挂载 personalized 镜像失败: %w", err)
	}

	Log.Info("ImageService", "personalized 开发者镜像挂载成功")
	return nil
}

// isImageAlreadyMounted 判断设备是否已经挂载了同类型镜像。
// 个性化开发者镜像一旦挂上，卸载命令通常无效（只能重启设备清除），
// 此时再次挂载会收到：
// A disk image of type Personalized/DeveloperDiskImage is already mounted at /System/Developer.
// 这种情况下现有镜像已可满足调试/定位需求，按挂载成功处理。
func isImageAlreadyMounted(err error) bool {
	return strings.Contains(err.Error(), "is already mounted")
}

// unmountPersonalizedImage 卸载个性化开发者镜像 (iOS >= 17)
func unmountPersonalizedImage(device ios.DeviceEntry, ver *semver.Version) error {
	pm, err := imagemounter.NewPersonalizedDeveloperDiskImageMounter(device, ver)
	if err != nil {
		return fmt.Errorf("创建 personalized mounter 失败: %w", err)
	}
	defer pm.Close()

	if err := pm.UnmountImage(); err != nil {
		return err
	}

	Log.Info("ImageService", "已卸载原有的 personalized 开发者镜像")
	return nil
}

// mountDeveloperImage 挂载开发者镜像
func mountDeveloperImage(device ios.DeviceEntry) error {

	imagePath, err := imagemounter.DownloadImageFor(device, ResolveAppDir("devimages"))
	if err != nil {
		return fmt.Errorf("获取开发者镜像失败: %w", err)
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
