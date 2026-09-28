package services

import (
	"fmt"
	"sync"

	"github.com/danielpaulus/go-ios/ios"

	"github.com/danielpaulus/go-ios/ios/instruments"
	"github.com/danielpaulus/go-ios/ios/simlocation"
)

type locationSimulator interface {
	StartSimulateLocation(lat, lon float64) error
	StopSimulateLocation() error
	Close()
}

// LocationService 位置模拟服务
type LocationService struct {
	mu              sync.Mutex
	locationServers map[string]locationSimulator
	legacyDevices   map[string]ios.DeviceEntry
}

func NewLocationService() *LocationService {
	return &LocationService{
		locationServers: make(map[string]locationSimulator),
		legacyDevices:   make(map[string]ios.DeviceEntry),
	}
}

func (l *LocationService) SetLocation(udid string, lat, lon float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.locationServers == nil {
		l.locationServers = make(map[string]locationSimulator)
	}
	if l.legacyDevices == nil {
		l.legacyDevices = make(map[string]ios.DeviceEntry)
	}

	// 复用已建立的 DTX 服务，避免每个定位点重新查询版本和握手 RSD。
	if server, exists := l.locationServers[udid]; exists {
		return l.setSimulatedLocation(udid, server, lat, lon)
	}
	if device, exists := l.legacyDevices[udid]; exists {
		return l.setLegacyLocation(udid, device, lat, lon)
	}

	device, version, err := GetDeviceAndVersion(udid)
	if err != nil {
		return err
	}

	// iOS 17+ 需要通过 tunnel 设备对象连接 dtservicehub
	if version.Major() >= 17 {
		tunnelDevice, err := getTunnelDevice(udid)
		if err != nil {
			return fmt.Errorf("获取隧道设备失败: %w", err)
		}

		server, err := instruments.NewLocationSimulationService(*tunnelDevice)
		if err != nil {
			return fmt.Errorf("创建位置模拟服务失败: %w", err)
		}
		l.locationServers[udid] = server
		return l.setSimulatedLocation(udid, server, lat, lon)
	}

	l.legacyDevices[udid] = device
	return l.setLegacyLocation(udid, device, lat, lon)
}

func (l *LocationService) setSimulatedLocation(udid string, server locationSimulator, lat, lon float64) error {
	if err := server.StartSimulateLocation(lat, lon); err != nil {
		server.Close()
		delete(l.locationServers, udid)
		return fmt.Errorf("启动位置模拟失败: %w", err)
	}
	return nil
}

func (l *LocationService) setLegacyLocation(udid string, device ios.DeviceEntry, lat, lon float64) error {
	if err := simlocation.SetLocation(device, fmt.Sprintf("%f", lat), fmt.Sprintf("%f", lon)); err != nil {
		delete(l.legacyDevices, udid)
		Log.Error("LocationService", fmt.Sprintf("设置位置失败 for %s: %v", udid, err))
		return fmt.Errorf("设置位置失败: %w", err)
	}

	return nil
}

// ResetLocation 重置设备位置
func (l *LocationService) ResetLocation(udid string) error {
	Log.Info("LocationService", fmt.Sprintf("重置设备 %s 位置...", udid))
	l.mu.Lock()
	defer l.mu.Unlock()

	if server, exists := l.locationServers[udid]; exists {
		delete(l.locationServers, udid)
		if err := server.StopSimulateLocation(); err != nil {
			// go-ios 仅在停止成功时自动关闭连接；失败时也必须释放。
			server.Close()
			return fmt.Errorf("停止位置模拟失败: %w", err)
		}
		return nil
	}

	device, exists := l.legacyDevices[udid]
	delete(l.legacyDevices, udid)
	if !exists {
		versionDevice, version, err := GetDeviceAndVersion(udid)
		if err != nil {
			return err
		}
		if version.Major() >= 17 {
			return nil
		}
		device = versionDevice
	}
	if err := simlocation.ResetLocation(device); err != nil {
		return fmt.Errorf("重置位置失败: %w", err)
	}

	Log.Info("LocationService", fmt.Sprintf("设备 %s 位置已重置", udid))
	return nil
}
