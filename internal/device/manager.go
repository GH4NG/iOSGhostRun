package device

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/imagemounter"
	"github.com/danielpaulus/go-ios/ios/instruments"
	"github.com/danielpaulus/go-ios/ios/simlocation"
	"github.com/danielpaulus/go-ios/ios/tunnel"

	"iOSGhostRun/internal/logger"
)

// DeviceInfo 设备信息
type DeviceInfo struct {
	UDID           string `json:"udid"`
	Name           string `json:"name"`
	ProductType    string `json:"productType"`
	ProductVersion string `json:"productVersion"`
	Connected      bool   `json:"connected"`
}

// Manager 设备管理器
type Manager struct {
	mu              sync.RWMutex
	devices         map[string]*DeviceInfo
	selectedDevice  string
	deviceEntry     *ios.DeviceEntry
	imageBasedir    string
	log             *logger.Service
	locationService *instruments.LocationSimulationService
}

// NewManager 创建新的设备管理器
func NewManager(log *logger.Service) *Manager {
	currentDir, _ := os.Getwd()
	imageDir := filepath.Join(currentDir, "devimages")
	os.MkdirAll(imageDir, 0755)

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

	// 获取所有连接的设备
	deviceList, err := ios.ListDevices()
	if err != nil {
		m.log.Error("设备管理", "获取设备列表失败: %v", err)
		return nil, fmt.Errorf("获取设备列表失败: %w", err)
	}

	// 清空旧设备列表
	m.devices = make(map[string]*DeviceInfo)

	var result []DeviceInfo
	for _, entry := range deviceList.DeviceList {
		info := &DeviceInfo{
			UDID:      entry.Properties.SerialNumber,
			Connected: true,
		}

		// 获取设备详细信息
		if values, err := ios.GetValues(entry); err == nil {
			info.Name = values.Value.DeviceName
			info.ProductType = values.Value.ProductType
			info.ProductVersion = values.Value.ProductVersion
			m.log.Debug("设备管理", "获取设备信息: %s (%s) iOS %s", info.Name, info.ProductType, info.ProductVersion)
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

	var result []DeviceInfo
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

	// 获取设备入口
	entry, err := ios.GetDevice(udid)
	if err != nil {
		m.log.Error("设备管理", "获取设备失败: %v", err)
		return fmt.Errorf("获取设备失败: %w", err)
	}

	m.selectedDevice = udid
	m.deviceEntry = &entry

	m.log.Info("设备管理", "已选择设备: %s", udid)
	return nil
}

// GetSelectedDevice 获取已选择的设备
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

// CheckDeveloperImage 检查开发者镜像是否已挂载
func (m *Manager) CheckDeveloperImage() (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.deviceEntry == nil {
		return false, fmt.Errorf("未选择设备")
	}

	m.log.Debug("设备管理", "检查开发者镜像挂载状态...")

	conn, err := imagemounter.NewImageMounter(*m.deviceEntry)
	if err != nil {
		m.log.Error("设备管理", "连接镜像服务失败: %v", err)
		return false, fmt.Errorf("连接镜像服务失败: %w", err)
	}
	defer conn.Close()

	signatures, err := conn.ListImages()
	if err != nil {
		m.log.Error("设备管理", "获取镜像列表失败: %v", err)
		return false, fmt.Errorf("获取镜像列表失败: %w", err)
	}

	mounted := len(signatures) > 0
	if mounted {
		m.log.Info("设备管理", "开发者镜像已挂载")
	} else {
		m.log.Info("设备管理", "开发者镜像未挂载")
	}

	return mounted, nil
}

// MountDeveloperImage 挂载开发者镜像
func (m *Manager) MountDeveloperImage() error {
	m.mu.RLock()
	deviceEntry := m.deviceEntry
	basedir := m.imageBasedir
	m.mu.RUnlock()

	if deviceEntry == nil {
		return fmt.Errorf("未选择设备")
	}

	m.log.Info("设备管理", "开始挂载开发者镜像...")

	// 检查是否已挂载
	conn, err := imagemounter.NewImageMounter(*deviceEntry)
	if err != nil {
		m.log.Error("设备管理", "连接镜像服务失败: %v", err)
		return fmt.Errorf("连接镜像服务失败: %w", err)
	}

	signatures, err := conn.ListImages()
	conn.Close()
	if err != nil {
		m.log.Error("设备管理", "获取镜像列表失败: %v", err)
		return fmt.Errorf("获取镜像列表失败: %w", err)
	}

	if len(signatures) > 0 {
		m.log.Info("设备管理", "开发者镜像已挂载，跳过")
		return nil
	}

	// 下载并挂载镜像
	m.log.Info("设备管理", "正在下载开发者镜像，目录: %s", basedir)
	imagePath, err := imagemounter.DownloadImageFor(*deviceEntry, basedir)
	if err != nil {
		m.log.Error("设备管理", "下载开发者镜像失败: %v", err)
		return fmt.Errorf("下载开发者镜像失败: %w", err)
	}

	m.log.Info("设备管理", "下载完成，镜像路径: %s", imagePath)
	m.log.Info("设备管理", "正在挂载开发者镜像...")

	err = imagemounter.MountImage(*deviceEntry, imagePath)
	if err != nil {
		m.log.Error("设备管理", "挂载开发者镜像失败: %v", err)
		return fmt.Errorf("挂载开发者镜像失败: %w", err)
	}

	m.log.Info("设备管理", "开发者镜像挂载成功！")
	return nil
}

// GetDeveloperImageStatus 获取开发者镜像状态
func (m *Manager) GetDeveloperImageStatus() (map[string]interface{}, error) {
	mounted, err := m.CheckDeveloperImage()

	status := map[string]interface{}{
		"mounted": mounted,
		"basedir": m.imageBasedir,
	}

	if err != nil {
		status["error"] = err.Error()
	}

	return status, nil
}

// ResetSimLocation 重置虚拟定位
func (m *Manager) ResetSimLocation() error {
	m.mu.Lock()
	deviceEntry := m.deviceEntry
	locationService := m.locationService
	m.mu.Unlock()

	if deviceEntry == nil {
		return fmt.Errorf("未选择设备")
	}

	m.log.Info("设备管理", "重置虚拟定位...")

	// 如果有 instruments 定位服务，先停止它
	if locationService != nil {
		err := locationService.StopSimulateLocation()
		if err != nil {
			m.log.Warn("设备管理", "停止 instruments 定位服务失败: %v", err)
		}
		m.mu.Lock()
		m.locationService = nil
		m.mu.Unlock()
	}

	err := simlocation.ResetLocation(*deviceEntry)
	if err != nil {
		m.log.Error("设备管理", "重置虚拟定位失败: %v", err)
		return fmt.Errorf("重置虚拟定位失败: %v", err)
	}

	m.log.Info("设备管理", "虚拟定位已重置")
	return nil
}

// GetDeviceStatus 获取设备详细状态
func (m *Manager) GetDeviceStatus() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := map[string]interface{}{
		"hasDevice":      m.deviceEntry != nil,
		"selectedDevice": m.selectedDevice,
		"imageBasedir":   m.imageBasedir,
	}

	if m.deviceEntry != nil {
		// 检查是否支持 RSD
		status["supportsRsd"] = m.deviceEntry.SupportsRsd()

		// 获取设备详细信息
		if values, err := ios.GetValues(*m.deviceEntry); err == nil {
			status["deviceName"] = values.Value.DeviceName
			status["productType"] = values.Value.ProductType
			status["productVersion"] = values.Value.ProductVersion
			status["buildVersion"] = values.Value.BuildVersion
			status["deviceClass"] = values.Value.DeviceClass

			m.log.Debug("设备管理", "设备状态: %s (%s) iOS %s (%s), RSD支持: %v",
				values.Value.DeviceName, values.Value.ProductType,
				values.Value.ProductVersion, values.Value.BuildVersion,
				m.deviceEntry.SupportsRsd())
		}

		// 检查开发者镜像
		conn, err := imagemounter.NewImageMounter(*m.deviceEntry)
		if err == nil {
			signatures, err := conn.ListImages()
			conn.Close()
			if err == nil {
				status["imageMounted"] = len(signatures) > 0
				status["imageCount"] = len(signatures)
			}
		}
	}

	return status, nil
}

// SetSimLocation 设置虚拟定位
func (m *Manager) SetSimLocation(lat, lon float64) error {
	m.mu.Lock()
	deviceEntry := m.deviceEntry
	locationService := m.locationService
	m.mu.Unlock()

	if deviceEntry == nil {
		return fmt.Errorf("未选择设备")
	}

	// 检查是否是支持 RSD 的设备
	if deviceEntry.SupportsRsd() {
		// 如果还没有创建服务，先创建
		if locationService == nil {
			m.log.Debug("设备管理", "创建 instruments 定位服务...")
			service, err := instruments.NewLocationSimulationService(*deviceEntry)
			if err != nil {
				m.log.Error("设备管理", "创建定位服务失败: %v", err)
				return fmt.Errorf("创建定位服务失败: %v", err)
			}
			m.mu.Lock()
			m.locationService = service
			locationService = service
			m.mu.Unlock()
		}

		err := locationService.StartSimulateLocation(lat, lon)
		if err != nil {
			m.log.Error("设备管理", "设置位置失败: %v", err)
			return fmt.Errorf("设置位置失败: %v", err)
		}
		return nil
	}

	latStr := fmt.Sprintf("%f", lat)
	lonStr := fmt.Sprintf("%f", lon)

	err := simlocation.SetLocation(*deviceEntry, latStr, lonStr)
	if err != nil {
		m.log.Error("设备管理", "设置位置失败: %v", err)
		return fmt.Errorf("设置位置失败: %v", err)
	}

	return nil
}

// CheckTunnelStatus 检查 tunnel 服务状态
func (m *Manager) CheckTunnelStatus() map[string]interface{} {
	isRunning := tunnel.IsAgentRunning()

	status := map[string]interface{}{
		"running": isRunning,
	}

	if isRunning {
		m.log.Info("设备管理", "Tunnel 服务正在运行")
		// 尝试获取 tunnel 列表
		tunnels, err := tunnel.ListRunningTunnels(ios.HttpApiHost(), ios.HttpApiPort())
		if err == nil {
			status["tunnelCount"] = len(tunnels)
		}
	} else {
		m.log.Warn("设备管理", "Tunnel 服务未运行，iOS 17+ 设备需要先启动 tunnel")
	}

	return status
}

// IsIOS17OrAbove 检查设备是否是 iOS 17 或更高版本
func (m *Manager) IsIOS17OrAbove() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.deviceEntry == nil {
		return false
	}

	return m.deviceEntry.SupportsRsd()
}

// StartTunnel 启动 tunnel 服务（iOS 17+ 设备）
func (m *Manager) StartTunnel() map[string]interface{} {
	result := map[string]interface{}{
		"success": false,
		"message": "",
	}

	// 检查是否已运行
	if tunnel.IsAgentRunning() {
		m.log.Info("设备管理", "Tunnel 服务已在运行")
		result["success"] = true
		result["message"] = "Tunnel 服务已在运行"
		result["alreadyRunning"] = true
		return result
	}

	m.mu.RLock()
	deviceEntry := m.deviceEntry
	m.mu.RUnlock()

	if deviceEntry == nil {
		result["message"] = "未选择设备"
		return result
	}

	m.log.Info("设备管理", "正在启动 Tunnel 服务...")

	// 创建配对记录管理器
	pm, err := tunnel.NewPairRecordManager(".")
	if err != nil {
		m.log.Error("设备管理", "创建配对管理器失败: %v", err)
		result["message"] = fmt.Sprintf("创建配对管理器失败: %v", err)
		return result
	}

	// 为当前设备启动 tunnel
	// 注意：这是高级 API，会自动处理配对和连接
	tunnelInfo, err := tunnel.ManualPairAndConnectToTunnel(m.getContext(), *deviceEntry, pm)
	if err != nil {
		m.log.Error("设备管理", "启动 Tunnel 失败: %v", err)
		result["message"] = fmt.Sprintf("启动 Tunnel 失败: %v", err)
		return result
	}

	m.log.Info("设备管理", "Tunnel 服务启动成功! 地址: %s, RSD 端口: %d", tunnelInfo.Address, tunnelInfo.RsdPort)
	result["success"] = true
	result["message"] = fmt.Sprintf("Tunnel 服务已启动 (%s:%d)", tunnelInfo.Address, tunnelInfo.RsdPort)
	return result
}

// getContext 获取上下文
func (m *Manager) getContext() context.Context {
	// 返回一个不会被取消的背景上下文
	return context.Background()
}
