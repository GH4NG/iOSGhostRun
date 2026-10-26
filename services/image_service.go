package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/amfi"
	"github.com/danielpaulus/go-ios/ios/imagemounter"
)

var imageOperationMu sync.Mutex

// MountImage 挂载镜像
func MountImage(udid string) error {
	udid = strings.TrimSpace(udid)
	if udid == "" {
		return fmt.Errorf("未选择设备")
	}
	imageOperationMu.Lock()
	defer imageOperationMu.Unlock()
	device, version, err := GetDeviceAndVersion(udid)
	if err != nil {
		return err
	}

	// 检查并尝试启用开发者模式
	if version.Major() >= 16 {
		enabled, err := imagemounter.IsDevModeEnabled(device)
		if err != nil {
			return fmt.Errorf("检查开发者模式失败: %w", err)
		}
		if !enabled {
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
	}

	if version.Major() >= 17 {
		tunnelDevice, err := getTunnelDevice(udid)
		if err != nil {
			return fmt.Errorf("获取隧道设备失败: %w", err)
		}
		return mountDeveloperImage(*tunnelDevice, version)
	}
	return mountDeveloperImage(device, version)
}

// UnmountImage 卸载镜像。重复清理时，没有已挂载镜像视为成功。
func UnmountImage(udid string) error {
	udid = strings.TrimSpace(udid)
	if udid == "" {
		return fmt.Errorf("未选择设备")
	}
	imageOperationMu.Lock()
	defer imageOperationMu.Unlock()
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
	defer closeImageMounter(mounter)

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

type developerImageKind uint8

const (
	imageLegacy developerImageKind = iota
	imagePersonalized
	imageCryptex
)

func selectDeveloperImageKind(version *semver.Version, supportsCryptex bool) developerImageKind {
	if version.Major() < 17 {
		return imageLegacy
	}
	if version.Major() >= 27 && supportsCryptex {
		return imageCryptex
	}
	return imagePersonalized
}

func supportsCryptexInstall(device ios.DeviceEntry) bool {
	return imagemounter.SupportsCryptexDDI(device)
}

func newDeveloperImageMounter(device ios.DeviceEntry, version *semver.Version, kind developerImageKind) (imagemounter.ImageMounter, error) {
	switch kind {
	case imageCryptex:
		return imagemounter.NewCryptexDeveloperDiskImageMounter(device)
	case imagePersonalized:
		return imagemounter.NewPersonalizedDeveloperDiskImageMounter(device, version)
	default:
		return imagemounter.NewDeveloperDiskImageMounter(device, version)
	}
}

func closeImageMounter(mounter imagemounter.ImageMounter) {
	if err := mounter.Close(); err != nil {
		Log.Warn("ImageService", fmt.Sprintf("关闭镜像服务连接失败: %v", err))
	}
}

func mountDeveloperImage(device ios.DeviceEntry, version *semver.Version) error {
	kind := selectDeveloperImageKind(version, supportsCryptexInstall(device))
	mounter, err := newDeveloperImageMounter(device, version, kind)
	if err != nil {
		return fmt.Errorf("连接镜像服务失败: %w", err)
	}
	defer closeImageMounter(mounter)

	images, err := mounter.ListImages()
	if err != nil {
		return fmt.Errorf("查询已挂载镜像失败: %w", err)
	}
	if len(images) > 0 {
		Log.Info("ImageService", "开发者镜像已挂载，复用现有镜像")
		return nil
	}
	if kind == imageCryptex {
		personalized, err := imagemounter.NewPersonalizedDeveloperDiskImageMounter(device, version)
		if err != nil {
			return fmt.Errorf("连接 Personalized 镜像服务失败: %w", err)
		}
		defer closeImageMounter(personalized)
		images, err := personalized.ListImages()
		if err != nil {
			return fmt.Errorf("查询 Personalized 镜像失败: %w", err)
		}
		if len(images) > 0 {
			Log.Info("ImageService", "Personalized 开发者镜像已挂载，复用现有镜像")
			return nil
		}
	}

	var imagePath string
	if kind != imageCryptex {
		imagePath, err = imagemounter.DownloadImageFor(device, ResolveAppDir("devimages"))
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		client := newDownloadHTTPClient(2 * time.Minute)
		defer client.CloseIdleConnections()
		repository := newCryptexRepository(client)
		imagePath, err = repository.ensure(ctx, ResolveAppDir("devimages"))
	}
	if err != nil {
		return fmt.Errorf("获取开发者镜像失败: %w", err)
	}
	return mountImageWithMounter(mounter, imagePath)
}

func mountImageWithMounter(mounter imagemounter.ImageMounter, imagePath string) error {
	images, err := mounter.ListImages()
	if err != nil {
		return fmt.Errorf("查询已挂载镜像失败: %w", err)
	}
	if len(images) > 0 {
		Log.Info("ImageService", "开发者镜像已挂载，复用现有镜像")
		return nil
	}
	if err := mounter.MountImage(imagePath); err != nil {
		return fmt.Errorf("挂载开发者镜像失败: %w", err)
	}
	Log.Info("ImageService", "开发者镜像挂载成功")
	return nil
}
