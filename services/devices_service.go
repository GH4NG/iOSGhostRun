package services

import (
	"fmt"
	"sync"

	"github.com/danielpaulus/go-ios/ios"
)

// DevicesService 设备管理服务
type DevicesService struct {
	mu           sync.RWMutex
	selectedUDID string
	deviceInfo   map[string]DeviceInfo
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
	for _, entry := range list.DeviceList {
		udid := entry.Properties.SerialNumber
		info, err := GetDeviceInfo(udid)
		if err != nil {
			Log.Debug("DevicesService", "跳过设备: "+udid+" -> "+err.Error())
			continue
		}

		devices = append(devices, info)
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

	d.mu.Lock()
	d.selectedUDID = udid
	d.mu.Unlock()

	imgSvc := &ImageService{}
	if err := imgSvc.MountImage(udid); err != nil {
		return err
	}

	if IsVersionAbove17(udid) {
		tunSvc := &TunnelService{}
		if err := tunSvc.startTunnel(); err != nil {
			return err
		}
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
