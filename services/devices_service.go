package services

import (
	"fmt"
	"strings"
	"sync"

	"github.com/danielpaulus/go-ios/ios"
)

// DevicesService 设备管理服务
type DevicesService struct {
	mu           sync.RWMutex
	selectedUDID string
	deviceInfo   map[string]DeviceInfo
}

func NewDevicesService() *DevicesService {
	return &DevicesService{
		deviceInfo: make(map[string]DeviceInfo),
	}
}

// ListDevices 获取可连接设备列表
func (d *DevicesService) ListDevices() ([]DeviceInfo, error) {
	Log.Info("DevicesService", "列出已连接的设备...")
	list, err := ios.ListDevices()
	if err != nil {
		Log.Error("DevicesService", "列出设备失败: "+err.Error())
		return nil, err
	}

	devices := make([]DeviceInfo, 0)
	trustHintShown := false
	for _, entry := range list.DeviceList {
		// 仅接受 USB 连接的设备，过滤掉 Network 或其他连接类型
		connectionType := entry.Properties.ConnectionType
		if connectionType == "" {
			connectionType = "unknown"
		}
		if strings.ToLower(connectionType) != "usb" {
			Log.Debug("DevicesService", fmt.Sprintf("跳过非 USB 连接设备: %s (连接类型: %s)", entry.Properties.SerialNumber, connectionType))
			continue
		}

		udid := entry.Properties.SerialNumber
		info, err := GetDeviceInfo(udid)
		if err != nil {
			Log.Debug("DevicesService", "跳过设备: "+udid+" -> "+err.Error())
			if !trustHintShown {
				errMsg := strings.ToLower(err.Error())
				if strings.Contains(errMsg, "invalidhostid") || strings.Contains(errMsg, "invalid host id") {
					Log.Warn("DevicesService", "设备连接被拒绝：请解锁手机并点击“信任此电脑”，然后重新插拔 USB 再刷新设备列表")
					trustHintShown = true
				}
			}
			continue
		}

		devices = append(devices, info)
	}

	if trustHintShown && len(devices) == 0 {
		return nil, fmt.Errorf("设备连接被拒绝：请解锁手机并点击“信任此电脑”")
	}

	Log.Info("DevicesService", fmt.Sprintf("找到 %d 个可连接设备", len(devices)))
	return devices, nil
}

// SelectDevice 选择设备
func (d *DevicesService) SelectDevice(udid string) error {
	Log.Info("DevicesService", "选择设备: "+udid)
	if udid == "" {
		return fmt.Errorf("udid required")
	}

	if _, err := ios.GetDevice(udid); err != nil {
		Log.Error("DevicesService", "选择设备 "+udid+" 失败: "+err.Error())
		return err
	}

	// 先检测 iOS 版本
	if IsVersionAbove17(udid) {
		// 再检测 Wintun
		if !CheckWintunInstalled() {
			err := fmt.Errorf("iOS 17+ 设备需要安装 Wintun")
			Log.Error("DevicesService", err.Error())
			return err
		}

		Log.Info("DevicesService", "检测到 iOS17+ 且 Wintun 已安装，启用隧道服务")
		if err := ensureTunnelReady(udid); err != nil {
			return err
		}
	}

	d.mu.Lock()
	d.selectedUDID = udid
	d.mu.Unlock()

	if err := MountImage(udid); err != nil {
		return err
	}

	Log.Info("DevicesService", fmt.Sprintf("挂载开发者镜像完成: %s", udid))
	return nil
}

// GetSelectedDevice 获取已选设备信息
func (d *DevicesService) GetSelectedDevice() (*DeviceInfo, error) {
	d.mu.RLock()
	udid := d.selectedUDID
	if udid == "" {
		d.mu.RUnlock()
		return nil, fmt.Errorf("未选择设备")
	}

	if info, ok := d.deviceInfo[udid]; ok {
		d.mu.RUnlock()
		return &info, nil
	}
	d.mu.RUnlock()

	info, err := GetDeviceInfo(udid)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
