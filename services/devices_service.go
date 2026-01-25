package services

import (
	"fmt"
	"sync"

	"github.com/danielpaulus/go-ios/ios"
)

// DeviceInfo 设备信息
type DeviceInfo struct {
	UDID           string
	DeviceName     string
	ProductType    string
	ProductVersion string
}

// DevicesService 设备管理服务
type DevicesService struct {
	mu           sync.RWMutex
	selectedUDID string
	DeviceEntry  map[string]ios.DeviceEntry
	DeviceInfo   map[string]DeviceInfo
}

// ListDevices 获取可连接设备列表
func (d *DevicesService) ListDevices() ([]DeviceInfo, error) {
	Log.Info("DevicesService", "Listing connected devices...")
	list, err := ios.ListDevices()
	if err != nil {
		Log.Error("DevicesService", "Failed to list devices: "+err.Error())
		return nil, err
	}

	devices := make([]DeviceInfo, 0, len(list.DeviceList))
	for _, entry := range list.DeviceList {
		udid := entry.Properties.SerialNumber

		dev, err := ios.GetDevice(udid)
		if err != nil {
			Log.Debug("DevicesService", "Skip device (cannot connect): "+udid+" -> "+err.Error())
			continue
		}

		info, err := d.GetDeviceInfo(udid, dev)
		if err != nil {
			Log.Debug("DevicesService", "Skip device (no values): "+udid+" -> "+err.Error())
			continue
		}

		devices = append(devices, DeviceInfo{
			UDID:           udid,
			DeviceName:     info.DeviceName,
			ProductType:    info.ProductType,
			ProductVersion: info.ProductVersion,
		})
	}

	Log.Info("DevicesService", fmt.Sprintf("Found %d connectable devices", len(devices)))
	return devices, nil
}

// GetDeviceInfo 获取设备信息
func (d *DevicesService) GetDeviceInfo(udid string, dev ios.DeviceEntry) (DeviceInfo, error) {
	d.mu.RLock()
	if d.DeviceInfo != nil {
		if v, ok := d.DeviceInfo[udid]; ok {
			d.mu.RUnlock()
			return v, nil
		}
	}
	d.mu.RUnlock()

	info, err := ios.GetValues(dev)
	if err != nil {
		return DeviceInfo{}, err
	}

	v := DeviceInfo{
		DeviceName:     info.Value.DeviceName,
		ProductType:    info.Value.ProductType,
		ProductVersion: info.Value.ProductVersion,
	}

	d.mu.Lock()
	if d.DeviceInfo == nil {
		d.DeviceInfo = make(map[string]DeviceInfo)
	}
	d.DeviceInfo[udid] = v
	d.mu.Unlock()
	return v, nil
}

// SelectDevice 选择设备
func (d *DevicesService) SelectDevice(udid string) error {
	Log.Info("DevicesService", "Selecting device: "+udid)
	if udid == "" {
		return fmt.Errorf("udid required")
	}

	if _, err := ios.GetDevice(udid); err != nil {
		Log.Error("DevicesService", "Failed to select device "+udid+": "+err.Error())
		return err
	}

	d.mu.Lock()
	d.selectedUDID = udid
	d.mu.Unlock()

	imgSvc := &ImageService{}
	if err := imgSvc.MountImage(udid); err != nil {
		return err
	}

	isAbove, _ := d.IsVersionAbove17(udid)

	if isAbove {
		tunSvc := &TunnelService{}
		if err := tunSvc.startTunnel(); err != nil {
			return err
		}
	}

	Log.Info("DevicesService", fmt.Sprintf("挂载开发者镜像完成: %s", udid))
	return nil
}

// IsVersionAbove17
func (d *DevicesService) IsVersionAbove17(udid string) (bool, error) {
	device, _ := ios.GetDevice(udid)
	version, _ := ios.GetProductVersion(device)
	if version.Major() < 17 {
		return false, nil
	}
	return true, nil
}
