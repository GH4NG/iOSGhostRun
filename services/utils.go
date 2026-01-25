package services

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Masterminds/semver"
	"github.com/danielpaulus/go-ios/ios"
)

// ResolveAppDir 返回应用的可写目录，用于存放指定子目录的数据。
//
// 目录选择优先级：
//  1. 可执行文件同级目录
//  2. 系统推荐的用户配置目录
//  3. 用户主目录
//  4. 当前工作目录
//
// 在任意一级成功创建目录后立即返回。
func ResolveAppDir(subdir string) string {
	const appName = "iosghostrun"

	// 可执行文件同级目录
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		dir := filepath.Join(exeDir, subdir)
		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir
		}
	}

	// 用户配置目录
	if configDir, err := os.UserConfigDir(); err == nil {
		var dir string

		switch runtime.GOOS {
		case "windows":
			// %AppData%\iosghostrun\subdir
			dir = filepath.Join(configDir, appName, subdir)
		case "darwin":
			// ~/Library/Application Support/iosghostrun/subdir
			dir = filepath.Join(configDir, appName, subdir)
		default:
			// Linux / Unix: ~/.config/iosghostrun/subdir
			dir = filepath.Join(configDir, appName, subdir)
		}

		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir
		}
	}

	// 用户主目录兜底
	if homeDir, err := os.UserHomeDir(); err == nil {
		dir := filepath.Join(homeDir, "."+appName, subdir)
		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir
		}
	}

	// 相对路径兜底
	dir := filepath.Join(subdir)
	_ = os.MkdirAll(dir, 0755)
	return dir
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

// IsVersionAbove17 检查设备版本是否在 iOS 17 以上
func IsVersionAbove17(udid string) bool {
	device, err := ios.GetDevice(udid)
	if err != nil {
		return false
	}
	version, err := ios.GetProductVersion(device)
	if err != nil {
		return false
	}
	return version.Major() >= 17
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

// DeviceInfo 设备信息
type DeviceInfo struct {
	UDID           string
	DeviceName     string
	ProductType    string
	ProductVersion string
}
