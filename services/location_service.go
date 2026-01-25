package services

import (
	"fmt"
	"sync"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/simlocation"
)

// LocationService 位置模拟服务
type LocationService struct {
	mu sync.Mutex
}

// SetLocation 设置设备虚拟位置
func (l *LocationService) SetLocation(udid string, lat, lon float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	device, err := ios.GetDevice(udid)
	if err != nil {
		return fmt.Errorf("获取设备失败: %w", err)
	}

	err = simlocation.SetLocation(device, fmt.Sprintf("%f", lat), fmt.Sprintf("%f", lon))
	if err != nil {
		Log.Error("LocationService", fmt.Sprintf("Failed to set location for %s: %v", udid, err))
		return fmt.Errorf("设置位置失败: %w", err)
	}

	return nil
}

// ResetLocation 重置设备位置
func (l *LocationService) ResetLocation(udid string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	device, err := ios.GetDevice(udid)
	if err != nil {
		return fmt.Errorf("获取设备失败: %w", err)
	}

	err = simlocation.ResetLocation(device)
	if err != nil {
		Log.Error("LocationService", fmt.Sprintf("Failed to reset location for %s: %v", udid, err))
		return fmt.Errorf("重置位置失败: %w", err)
	}

	return nil
}
