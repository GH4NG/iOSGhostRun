package device

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/danielpaulus/go-ios/ios"

	"iOSGhostRun/internal/logger"
	"iOSGhostRun/internal/storage"
)

// DeviceInfo 设备信息
type DeviceInfo struct {
	UDID           string `json:"udid"`
	Name           string `json:"name"`
	ProductType    string `json:"productType"`
	ProductVersion string `json:"productVersion"`
	Connected      bool   `json:"connected"`
	SupportsRsd    bool   `json:"supportsRsd"`
}

// Manager 设备管理器
type Manager struct {
	ctx              context.Context
	tunnelCancel     context.CancelFunc
	mu               sync.RWMutex
	devices          map[string]*DeviceInfo
	selectedDevice   string
	deviceEntry      *ios.DeviceEntry
	imageBasedir     string
	log              *logger.Service
	locationService  interface{}
	locationDeviceID string
}

func (m *Manager) Startup(ctx context.Context) {
	m.ctx = ctx
	m.log.Info("设备管理", "设备管理器已启动")

	// 刷新设备列表
	devices, err := m.RefreshDevices()
	if err != nil || len(devices) == 0 {
		return
	}

	for _, dev := range devices {
		if isVersionIOS17OrAbove(dev.ProductVersion) {
			m.log.Info("设备管理", "检测到 iOS 17+ 设备 (%s, iOS %s)，启动 Tunnel 服务...", dev.Name, dev.ProductVersion)
			m.startTunnelAsync()
			break
		}
	}
}

// isVersionIOS17OrAbove 检查版本号是否是 iOS 17+
func isVersionIOS17OrAbove(version string) bool {
	if version == "" {
		return false
	}
	// 取主版本号
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return false
	}
	var major int
	fmt.Sscanf(parts[0], "%d", &major)
	return major >= 17
}

// startTunnelAsync 异步启动 tunnel 服务
func (m *Manager) startTunnelAsync() {
	if m.tunnelCancel != nil {
		return // 已经在运行
	}

	tunnelCtx, cancel := context.WithCancel(m.ctx)
	m.tunnelCancel = cancel

	go func() {
		if err := StartTunnelService(tunnelCtx); err != nil {
			m.log.Error("设备管理", "Tunnel 服务异常: %v", err)
		}
	}()

	m.log.Info("设备管理", "Tunnel 服务已在后台启动，等待 tunnel 就绪...")

	go func() {
		select {
		case <-m.ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}

		m.log.Info("设备管理", "重新刷新设备列表以获取 tunnel 连接...")
		if _, err := m.RefreshDevices(); err != nil {
			m.log.Warn("设备管理", "刷新设备失败: %v", err)
		}
	}()
}

// StopTunnel 停止 tunnel 服务
func (m *Manager) StopTunnel() {
	if m.tunnelCancel != nil {
		m.tunnelCancel()
		m.tunnelCancel = nil
		m.log.Info("设备管理", "Tunnel 服务已停止")
	}
}

// NewManager 创建新的设备管理器
func NewManager(log *logger.Service) *Manager {
	imageDir := storage.ResolveAppDir("devimages")
	log.Info("设备管理", "设备管理器初始化完成，镜像目录: %s", imageDir)

	return &Manager{
		devices:      make(map[string]*DeviceInfo),
		imageBasedir: imageDir,
		log:          log,
	}
}

// RefreshDevices 刷新设备列表
func (m *Manager) RefreshDevices() ([]DeviceInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.Info("设备管理", "开始扫描连接的设备...")

	deviceList, err := ios.ListDevices()
	if err != nil {
		m.log.Error("设备管理", "获取设备列表失败: %v", err)
		return nil, fmt.Errorf("获取设备列表失败: %w", err)
	}

	// 清空旧设备列表
	m.devices = make(map[string]*DeviceInfo)

	var result []DeviceInfo
	for _, entry := range deviceList.DeviceList {
		udid := entry.Properties.SerialNumber

		// 检查是否有对应的 tunnel 连接
		supportsRsd := entry.SupportsRsd()
		if !supportsRsd {
			// 尝试从 tunnel 服务获取信息
			if _, err := GetTunnelForDevice(udid); err == nil {
				supportsRsd = true
			}
		}

		info := &DeviceInfo{
			UDID:        udid,
			Connected:   true,
			SupportsRsd: supportsRsd,
		}

		// 获取设备详细信息
		if values, err := ios.GetValues(entry); err == nil {
			info.Name = values.Value.DeviceName
			info.ProductType = values.Value.ProductType
			info.ProductVersion = values.Value.ProductVersion
			m.log.Debug("设备管理", "设备: %s (%s) iOS %s, RSD: %v",
				info.Name, info.ProductType, info.ProductVersion, info.SupportsRsd)
		}

		if info.Name == "" {
			info.Name = "iOS Device"
		}

		m.devices[info.UDID] = info
		result = append(result, *info)
	}

	m.log.Info("设备管理", "扫描完成，发现 %d 台设备", len(result))
	return result, nil
}

// GetDevices 获取设备列表
func (m *Manager) GetDevices() []DeviceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]DeviceInfo, 0, len(m.devices))
	for _, info := range m.devices {
		result = append(result, *info)
	}
	return result
}

// SelectDevice 选择设备
func (m *Manager) SelectDevice(udid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.devices[udid]; !exists {
		m.log.Error("设备管理", "设备不存在: %s", udid)
		return fmt.Errorf("设备不存在: %s", udid)
	}

	entry, err := ios.GetDevice(udid)
	if err != nil {
		m.log.Error("设备管理", "获取设备失败: %v", err)
		return fmt.Errorf("获取设备失败: %w", err)
	}

	m.selectedDevice = udid
	m.deviceEntry = &entry

	m.log.Info("设备管理", "已选择设备: %s (RSD: %v)", udid, entry.SupportsRsd())
	return nil
}

// GetSelectedDevice 获取已选择的设备 UDID
func (m *Manager) GetSelectedDevice() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.selectedDevice
}

// GetDeviceEntry 获取设备入口
func (m *Manager) GetDeviceEntry() *ios.DeviceEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.deviceEntry
}

// HasSelectedDevice 是否已选择设备
func (m *Manager) HasSelectedDevice() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.deviceEntry != nil
}

// ClearSelectedDevice 清除已选择的设备
func (m *Manager) ClearSelectedDevice() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.selectedDevice = ""
	m.deviceEntry = nil
}
