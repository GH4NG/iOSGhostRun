package services

import (
	"fmt"
	"sync"

	"github.com/danielpaulus/go-ios/ios"

	"github.com/danielpaulus/go-ios/ios/instruments"
	"github.com/danielpaulus/go-ios/ios/simlocation"
)

// LocationService 位置模拟服务
type LocationService struct {
	mu              sync.Mutex
	locationServers map[string]*instruments.LocationSimulationService
}

func (l *LocationService) SetLocation(udid string, lat, lon float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.locationServers == nil {
		l.locationServers = make(map[string]*instruments.LocationSimulationService)
	}

	device, err := ios.GetDevice(udid)
	if err != nil {
		return fmt.Errorf("获取设备失败: %w", err)
	}

	// iOS 17+ 使用 instruments 服务
	if IsVersionAbove17(udid) {
		// 检查是否已经有服务实例，没有才创建
		server, exists := l.locationServers[udid]
		if !exists {
			var err error
			server, err = instruments.NewLocationSimulationService(device)
			if err != nil {
				return fmt.Errorf("创建位置模拟服务失败: %w", err)
			}
			// 保存服务实例供后续复用
			l.locationServers[udid] = server
		}

		// 使用已存在的服务多次定位
		err = server.StartSimulateLocation(lat, lon)
		if err != nil {
			return fmt.Errorf("启动位置模拟失败: %w", err)
		}
	}

	err = simlocation.SetLocation(device, fmt.Sprintf("%f", lat), fmt.Sprintf("%f", lon))
	if err != nil {
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
		err := server.StopSimulateLocation()
		if err != nil {
			Log.Error("LocationService", fmt.Sprintf("停止位置模拟失败 for %s: %v", udid, err))
		}
		delete(l.locationServers, udid)
	}

	device, err := ios.GetDevice(udid)
	if err != nil {
		return fmt.Errorf("获取设备失败: %w", err)
	}

	err = simlocation.ResetLocation(device)
	if err != nil {
		Log.Error("LocationService", fmt.Sprintf("重置位置失败 for %s: %v", udid, err))
		return fmt.Errorf("重置位置失败: %w", err)
	}

	Log.Info("LocationService", fmt.Sprintf("设备 %s 位置已重置", udid))
	return nil
}
