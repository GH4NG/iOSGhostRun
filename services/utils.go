package services

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Masterminds/semver"
	"github.com/danielpaulus/go-ios/ios"
)

// ResolveAppDir 返回启动时选定的数据根目录下的子目录。
// InitializeDataDirectory 必须在启动 Wails 服务前调用。
func ResolveAppDir(subdir string) string {
	return filepath.Join(dataDirectory, subdir)
}

// GetDeviceAndVersion 获取设备和版本信息
func GetDeviceAndVersion(udid string) (ios.DeviceEntry, *semver.Version, error) {
	device, err := ios.GetDevice(udid)
	if err != nil {
		return ios.DeviceEntry{}, nil, fmt.Errorf("获取设备失败: %w", err)
	}

	vals, err := ios.GetValues(device)
	if err != nil {
		return ios.DeviceEntry{}, nil, fmt.Errorf("获取设备信息失败: %w", err)
	}

	ver, err := semver.NewVersion(vals.Value.ProductVersion)
	if err != nil {
		return ios.DeviceEntry{}, nil, fmt.Errorf("解析系统版本失败: %w", err)
	}

	return device, ver, nil
}

// GetDeviceInfo 获取设备信息
func GetDeviceInfo(udid string) (DeviceInfo, error) {
	device, err := ios.GetDevice(udid)
	if err != nil {
		return DeviceInfo{}, fmt.Errorf("获取设备失败: %w", err)
	}

	info, err := ios.GetValues(device)
	if err != nil {
		return DeviceInfo{}, fmt.Errorf("获取设备值失败: %w", err)
	}

	return DeviceInfo{
		UDID:           udid,
		DeviceName:     info.Value.DeviceName,
		ProductType:    info.Value.ProductType,
		ProductVersion: info.Value.ProductVersion,
	}, nil
}

func CheckWintunInstalled() bool {
	if runtime.GOOS != "windows" {
		return true
	}

	systemDir := os.Getenv("SystemRoot")
	wintunPath := filepath.Join(systemDir, "System32", "wintun.dll")
	if _, err := os.Stat(wintunPath); err == nil {
		return true
	}
	return false
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	UDID           string
	DeviceName     string
	ProductType    string
	ProductVersion string
}
