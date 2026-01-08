package device

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/instruments"
	"github.com/danielpaulus/go-ios/ios/simlocation"
)

// isIOS17OrAbove 检查设备是否是 iOS 17 或更高版本
func (m *Manager) isIOS17OrAbove(device ios.DeviceEntry) bool {
	// 先检查 SupportsRsd
	if device.SupportsRsd() {
		return true
	}

	// 再检查版本号
	values, err := ios.GetValues(device)
	if err != nil {
		return false
	}

	version, err := semver.NewVersion(values.Value.ProductVersion)
	if err != nil {
		return false
	}

	return version.Major() >= 17
}

// SetLocation 设置虚拟定位
func (m *Manager) SetLocation(lat, lon float64) error {
	deviceEntry, _, err := m.getDeviceContext()
	if err != nil {
		return err
	}

	// m.log.Info("设备管理", "设置虚拟定位: %.6f, %.6f", lat, lon)

	// iOS 17+ 使用 instruments 服务
	if m.isIOS17OrAbove(*deviceEntry) {
		service, err := m.getOrCreateLocationService(deviceEntry)
		if err != nil {
			return err
		}

		if err := service.StartSimulateLocation(lat, lon); err != nil {
			m.log.Error("设备管理", "设置位置失败: %v", err)
			// 服务可能已断开，清除缓存并重试
			m.closeLocationService()
			return fmt.Errorf("设置位置失败: %w", err)
		}

		// m.log.Info("设备管理", "虚拟定位设置成功 (instruments)")
		return nil
	}

	// iOS 16 及以下使用 simlocation
	latStr := fmt.Sprintf("%f", lat)
	lonStr := fmt.Sprintf("%f", lon)

	if err := simlocation.SetLocation(*deviceEntry, latStr, lonStr); err != nil {
		m.log.Error("设备管理", "设置位置失败: %v", err)
		return fmt.Errorf("设置位置失败: %w", err)
	}

	// m.log.Info("设备管理", "虚拟定位设置成功 (simlocation)")
	return nil
}

// getOrCreateLocationService 获取或创建 instruments 位置服务
func (m *Manager) getOrCreateLocationService(deviceEntry *ios.DeviceEntry) (*instruments.LocationSimulationService, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	udid := deviceEntry.Properties.SerialNumber

	// 如果已有服务且设备相同，直接返回
	if m.locationService != nil && m.locationDeviceID == udid {
		if service, ok := m.locationService.(*instruments.LocationSimulationService); ok {
			return service, nil
		}
	}

	// 关闭旧服务
	if m.locationService != nil {
		if service, ok := m.locationService.(*instruments.LocationSimulationService); ok {
			service.Close()
		}
		m.locationService = nil
		m.locationDeviceID = ""
	}

	// 创建新服务
	tunnelDevice, err := m.getTunnelDevice(udid)
	if err != nil {
		m.log.Warn("设备管理", "获取 tunnel 设备失败: %v,尝试直接连接...", err)
		tunnelDevice = deviceEntry
	}

	service, err := instruments.NewLocationSimulationService(*tunnelDevice)
	if err != nil {
		m.log.Error("设备管理", "创建定位服务失败: %v", err)
		return nil, fmt.Errorf("创建定位服务失败: %w", err)
	}

	m.locationService = service
	m.locationDeviceID = udid
	m.log.Debug("设备管理", "创建了新的 instruments 位置服务")

	return service, nil
}

// closeLocationService 关闭 instruments 位置服务
func (m *Manager) closeLocationService() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.locationService != nil {
		if service, ok := m.locationService.(*instruments.LocationSimulationService); ok {
			service.Close()
		}
		m.locationService = nil
		m.locationDeviceID = ""
		m.log.Debug("设备管理", "已关闭 instruments 位置服务")
	}
}

// getTunnelDevice 获取通过 tunnel 连接的设备 entry
func (m *Manager) getTunnelDevice(udid string) (*ios.DeviceEntry, error) {
	// 从 tunnel 获取设备信息
	tunnelInfo, err := GetTunnelForDevice(udid)
	if err != nil {
		return nil, fmt.Errorf("获取 tunnel 信息失败: %w", err)
	}

	// m.log.Debug("设备管理", "找到 tunnel: %s:%d, userspaceTUN=%v, port=%d",
	// 	tunnelInfo.Address, tunnelInfo.RsdPort, tunnelInfo.UserspaceTUN, tunnelInfo.UserspaceTUNPort)

	// 先获取基础设备信息
	baseDevice, err := ios.GetDevice(udid)
	if err != nil {
		return nil, fmt.Errorf("获取设备信息失败: %w", err)
	}

	// 设置 userspace TUN 信息
	baseDevice.UserspaceTUN = tunnelInfo.UserspaceTUN
	baseDevice.UserspaceTUNPort = tunnelInfo.UserspaceTUNPort
	baseDevice.UserspaceTUNHost = "localhost"

	// 连接到 RSD 服务获取服务端口映射
	rsdService, err := ios.NewWithAddrPortDevice(tunnelInfo.Address, tunnelInfo.RsdPort, baseDevice)
	if err != nil {
		return nil, fmt.Errorf("连接 RSD 服务失败: %w", err)
	}
	defer rsdService.Close()

	// 执行握手获取 RsdPortProvider
	rsdProvider, err := rsdService.Handshake()
	if err != nil {
		return nil, fmt.Errorf("RSD 握手失败: %w", err)
	}

	// 使用 RsdPortProvider 获取带 tunnel 信息的设备
	device, err := ios.GetDeviceWithAddress(udid, tunnelInfo.Address, rsdProvider)
	if err != nil {
		return nil, fmt.Errorf("通过 tunnel 获取设备失败: %w", err)
	}

	// 保留 userspace TUN 设置
	device.UserspaceTUN = tunnelInfo.UserspaceTUN
	device.UserspaceTUNPort = tunnelInfo.UserspaceTUNPort
	device.UserspaceTUNHost = "localhost"

	return &device, nil
}

// ResetLocation 重置虚拟定位
func (m *Manager) ResetLocation() error {
	deviceEntry, _, err := m.getDeviceContext()
	if err != nil {
		return err
	}

	m.log.Info("设备管理", "重置虚拟定位...")

	// iOS 17+ 使用 instruments 服务
	if m.isIOS17OrAbove(*deviceEntry) {
		m.mu.Lock()
		service := m.locationService
		m.mu.Unlock()

		// 如果有缓存的服务，尝试使用它重置
		if service != nil {
			if locService, ok := service.(*instruments.LocationSimulationService); ok {
				if err := locService.StopSimulateLocation(); err != nil {
					m.log.Warn("设备管理", "通过缓存服务重置位置失败: %v", err)
				} else {
					// m.log.Info("设备管理", "虚拟定位已重置 (instruments)")
				}
			}
		}

		m.closeLocationService()
		m.log.Info("设备管理", "已关闭位置服务连接")
		return nil
	}

	// iOS 16 及以下使用 simlocation
	if err := simlocation.ResetLocation(*deviceEntry); err != nil {
		m.log.Error("设备管理", "重置虚拟定位失败: %v", err)
		return fmt.Errorf("重置虚拟定位失败: %w", err)
	}

	m.log.Info("设备管理", "虚拟定位已重置 (simlocation)")
	return nil
}
