package services

import (
	"fmt"
	"sync"

	"github.com/danielpaulus/go-ios/ios"
)

// DeviceInfo 设备信息
type DeviceInfo struct {
	UDID           string `json:"udid"`
	DeviceName     string `json:"deviceName"`
	ProductType    string `json:"productType"`
	ProductVersion string `json:"productVersion"`
}

// DevicesService 设备管理服务
type DevicesService struct {
	mu           sync.Mutex
	selectedUDID string
}

// ListDevices 获取已连接设备列表
func (d *DevicesService) ListDevices() ([]DeviceInfo, error) {
	Log.Info("DevicesService", "Listing connected devices...")
	deviceList, err := ios.ListDevices()
	if err != nil {
		Log.Error("DevicesService", "Failed to list devices: "+err.Error())
		return nil, err
	}

	var devices []DeviceInfo
	for _, entry := range deviceList.DeviceList {
		info := DeviceInfo{
			UDID: entry.Properties.SerialNumber,
		}

		device, err := ios.GetDevice(entry.Properties.SerialNumber)
		if err == nil {
			values, err := ios.GetValues(device)
			if err == nil {
				info.DeviceName = values.Value.DeviceName
				info.ProductType = values.Value.ProductType
				info.ProductVersion = values.Value.ProductVersion
			}
		}

		devices = append(devices, info)
	}

	Log.Info("DevicesService", fmt.Sprintf("Found %d devices", len(devices)))
	return devices, nil
}

// GetDeviceInfo 获取单个设备详细信息
func (d *DevicesService) GetDeviceInfo(udid string) (*DeviceInfo, error) {
	Log.Info("DevicesService", "Fetching info for device: "+udid)
	device, err := ios.GetDevice(udid)
	if err != nil {
		Log.Error("DevicesService", "Failed to get device "+udid+": "+err.Error())
		return nil, err
	}

	values, err := ios.GetValues(device)
	if err != nil {
		Log.Error("DevicesService", "Failed to get values for device "+udid+": "+err.Error())
		return nil, err
	}

	info := &DeviceInfo{
		UDID:           udid,
		DeviceName:     values.Value.DeviceName,
		ProductType:    values.Value.ProductType,
		ProductVersion: values.Value.ProductVersion,
	}

	Log.Info("DevicesService", fmt.Sprintf("Devices info: %+v", info))
	return info, nil
}

// SelectDevice 选择当前操作的设备
func (d *DevicesService) SelectDevice(udid string) error {
	Log.Info("DevicesService", "Selecting device: "+udid)
	_, err := ios.GetDevice(udid)
	if err != nil {
		Log.Error("DevicesService", "Failed to select device "+udid+": "+err.Error())
		return err
	}

	d.mu.Lock()
	d.selectedUDID = udid
	d.mu.Unlock()

	imgSvc := &ImageService{}
	if err := imgSvc.MountDeveloperImage(udid); err != nil {
		Log.Error("DevicesService", fmt.Sprintf("挂载开发者镜像失败: %s: %v", udid, err))
		return err
	}
	Log.Info("DevicesService", fmt.Sprintf("挂载开发者镜像完成: %s", udid))

	return nil
}

// GetSelectedDevice 返回当前已选择设备的详细信息
func (d *DevicesService) GetSelectedDevice() (*DeviceInfo, error) {
	d.mu.Lock()
	udid := d.selectedUDID
	d.mu.Unlock()
	if udid == "" {
		return nil, fmt.Errorf("no device selected")
	}
	return d.GetDeviceInfo(udid)
}

// IsImageMounted 检查是否已挂载镜像
func (d *DevicesService) IsImageMounted() (bool, error) {
	d.mu.Lock()
	udid := d.selectedUDID
	d.mu.Unlock()
	if udid == "" {
		return false, fmt.Errorf("no device selected")
	}

	imgSvc := &ImageService{}
	mounted, err := imgSvc.CheckDeveloperImage(udid)
	if err != nil {
		return false, err
	}
	return mounted, nil
}

// DisconnectDevice 断开设备连接
func (d *DevicesService) DisconnectDevice(udid string) error {
	Log.Info("DevicesService", "Disconnecting device: "+udid)
	device, err := ios.GetDevice(udid)
	if err != nil {
		Log.Error("DevicesService", "Failed to get device "+udid+": "+err.Error())
		return err
	}

	if closer, ok := interface{}(device).(interface{ Close() error }); ok {
		if cerr := closer.Close(); cerr != nil {
			Log.Error("DevicesService", "Failed to close device "+udid+": "+cerr.Error())
			return cerr
		}

		d.mu.Lock()
		if d.selectedUDID == udid {
			d.selectedUDID = ""
		}
		d.mu.Unlock()
		return nil
	}

	return fmt.Errorf("device does not support close operation")
}
