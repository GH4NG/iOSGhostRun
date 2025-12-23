package runner

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"iOSGhostRun/internal/device"
	"iOSGhostRun/internal/location"
	"iOSGhostRun/internal/logger"
)

// RunStatus 运行状态
type RunStatus string

const (
	StatusIdle    RunStatus = "idle"
	StatusRunning RunStatus = "running"
	StatusPaused  RunStatus = "paused"
)

// RunConfig 运行配置
type RunConfig struct {
	Speed          float64 `json:"speed"`          // 速度 (km/h)
	SpeedVariation float64 `json:"speedVariation"` // 速度变化范围 (km/h)
	RouteVariation float64 `json:"routeVariation"` // 路线偏移范围 (米)
	LoopCount      int     `json:"loopCount"`      // 循环次数 (0=无限)
	UpdateInterval int     `json:"updateInterval"` // 位置更新间隔 (毫秒)
}

// RunStats 运行统计
type RunStats struct {
	Status        RunStatus `json:"status"`
	TotalDistance float64   `json:"totalDistance"` // 总距离 (米)
	CurrentSpeed  float64   `json:"currentSpeed"`  // 当前速度 (km/h)
	ElapsedTime   int64     `json:"elapsedTime"`   // 已用时间 (秒)
	CurrentLoop   int       `json:"currentLoop"`   // 当前循环
	PointIndex    int       `json:"pointIndex"`    // 当前点索引
	TotalPoints   int       `json:"totalPoints"`   // 总点数
	CurrentLat    float64   `json:"currentLat"`    // 当前纬度
	CurrentLon    float64   `json:"currentLon"`    // 当前经度
}

// Service 跑步服务
type Service struct {
	deviceManager   *device.Manager
	locationService *location.Service
	log             *logger.Service

	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wailsCtx context.Context

	status     RunStatus
	config     RunConfig
	stats      RunStats
	route      []location.Coordinate
	startTime  time.Time
	pausedTime time.Duration
}

// NewService 创建跑步服务
func NewService(dm *device.Manager, ls *location.Service, log *logger.Service) *Service {
	return &Service{
		deviceManager:   dm,
		locationService: ls,
		log:             log,
		status:          StatusIdle,
		config: RunConfig{
			Speed:          8.0,  // 默认8km/h
			SpeedVariation: 1.0,  // 默认±1km/h变化
			RouteVariation: 3.0,  // 默认±3米路线偏移
			LoopCount:      1,    // 默认1圈
			UpdateInterval: 1000, // 默认1秒更新一次
		},
	}
}

// Startup Wails启动回调
func (s *Service) Startup(ctx context.Context) {
	s.wailsCtx = ctx
	s.log.Info("跑步服务", "跑步服务已启动")
}

// Shutdown Wails关闭回调
func (s *Service) Shutdown(ctx context.Context) {
	s.log.Info("跑步服务", "跑步服务关闭中...")
	s.Stop()
}

// GetConfig 获取配置
func (s *Service) GetConfig() RunConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// SetConfig 设置配置
func (s *Service) SetConfig(config RunConfig) error {
	if config.Speed <= 0 || config.Speed > 30 {
		return fmt.Errorf("速度必须在 0-30 km/h 之间")
	}
	if config.UpdateInterval < 100 {
		return fmt.Errorf("更新间隔不能小于 100ms")
	}

	s.mu.Lock()
	s.config = config
	s.mu.Unlock()

	s.log.Info("跑步服务", "配置已更新: 速度=%.1fkm/h, 变化=±%.1fkm/h, 偏移=±%.0fm, 圈数=%d",
		config.Speed, config.SpeedVariation, config.RouteVariation, config.LoopCount)
	return nil
}

// GetStats 获取运行统计
func (s *Service) GetStats() RunStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := s.stats
	stats.Status = s.status

	if s.status == StatusRunning {
		stats.ElapsedTime = int64(time.Since(s.startTime).Seconds()) - int64(s.pausedTime.Seconds())
	}

	return stats
}

// SetRoute 设置路线
func (s *Service) SetRoute(points []location.Coordinate) error {
	if len(points) < 2 {
		return fmt.Errorf("路线至少需要2个点")
	}

	s.mu.Lock()
	// 插值路线，使点更密集
	s.route = s.locationService.InterpolateRoute(points, 0.005) // 每5米一个点
	s.mu.Unlock()

	s.log.Info("跑步服务", "路线已设置: 原始%d点, 插值后%d点", len(points), len(s.route))
	return nil
}

// Start 开始跑步
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.deviceManager.HasSelectedDevice() {
		s.log.Error("跑步服务", "请先选择设备")
		return fmt.Errorf("请先选择设备")
	}

	if len(s.route) < 2 {
		s.log.Error("跑步服务", "请先设置路线")
		return fmt.Errorf("请先设置路线")
	}

	if s.status == StatusRunning {
		return fmt.Errorf("已经在运行中")
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.status = StatusRunning
	s.startTime = time.Now()
	s.pausedTime = 0
	s.stats = RunStats{
		Status:      StatusRunning,
		TotalPoints: len(s.route),
	}

	s.log.Info("跑步服务", "开始跑步！速度: %.1fkm/h, 路线点数: %d", s.config.Speed, len(s.route))

	go s.runLoop()
	return nil
}

// Pause 暂停
func (s *Service) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != StatusRunning {
		return fmt.Errorf("当前未在运行")
	}

	s.status = StatusPaused
	s.log.Info("跑步服务", "跑步已暂停")
	return nil
}

// Resume 恢复
func (s *Service) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != StatusPaused {
		return fmt.Errorf("当前未暂停")
	}

	s.status = StatusRunning
	s.log.Info("跑步服务", "跑步已恢复")
	return nil
}

// Stop 停止
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}
	s.status = StatusIdle

	// 重置位置
	if s.deviceManager.HasSelectedDevice() {
		s.deviceManager.ResetSimLocation()
		s.log.Info("跑步服务", "设备位置已重置")
	}

	s.log.Info("跑步服务", "跑步已停止，总距离: %.0fm", s.stats.TotalDistance)
	return nil
}

// ResetLocation 重置设备位置
func (s *Service) ResetLocation() error {
	if !s.deviceManager.HasSelectedDevice() {
		return fmt.Errorf("请先选择设备")
	}

	err := s.deviceManager.ResetSimLocation()
	if err != nil {
		s.log.Error("跑步服务", "重置位置失败: %v", err)
		return err
	}

	s.log.Info("跑步服务", "设备位置已重置为真实位置")
	return nil
}

// SetSingleLocation 设置单个位置点
func (s *Service) SetSingleLocation(lat, lon float64) error {
	if !s.deviceManager.HasSelectedDevice() {
		return fmt.Errorf("请先选择设备")
	}

	return s.deviceManager.SetSimLocation(lat, lon)
}

// runLoop 运行循环
func (s *Service) runLoop() {

	updateInterval := 100 * time.Millisecond
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	pointIndex := 0
	currentLoop := 1
	var totalDistance float64
	var lastPos location.Coordinate
	lastLogTime := time.Now()

	var progress float64 = 0

	var offsetLat, offsetLon float64
	var lastOffsetUpdateTime = time.Now()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			status := s.status
			config := s.config
			route := s.route
			s.mu.RUnlock()

			if status == StatusPaused {
				continue
			}

			if status != StatusRunning {
				return
			}

			// 检查是否完成
			if pointIndex >= len(route)-1 {
				if config.LoopCount > 0 && currentLoop >= config.LoopCount {
					s.mu.Lock()
					s.status = StatusIdle
					s.mu.Unlock()
					s.log.Info("跑步服务", "跑步完成！总距离: %.0fm, 总圈数: %d", totalDistance, currentLoop)
					s.emitEvent("run:completed", nil)
					return
				}
				// 开始新的循环
				pointIndex = 0
				progress = 0
				currentLoop++
				s.log.Info("跑步服务", "开始第 %d 圈", currentLoop)
			}

			// 获取当前段的起点和终点
			startPoint := route[pointIndex]
			endPoint := route[pointIndex+1]

			currentSpeed := config.Speed
			if config.SpeedVariation > 0 {
				// 使用正弦函数实现速度的平滑波动
				elapsed := time.Since(s.startTime).Seconds()
				variation := math.Sin(elapsed*0.5) * config.SpeedVariation * 0.5
				currentSpeed += variation
				if currentSpeed < 0.5 {
					currentSpeed = 0.5
				}
			}

			// 计算两点之间的距离
			segmentDistance := s.locationService.CalculateDistance(startPoint, endPoint)
			if segmentDistance < 0.1 {
				// 两点太近，直接跳到下一段
				pointIndex++
				progress = 0
				continue
			}

			// 速度 km/h 转换为 m/s，然后计算这个时间间隔内移动的距离
			speedMS := currentSpeed * 1000 / 3600
			moveDistance := speedMS * updateInterval.Seconds()

			// 计算进度增量
			progressIncrement := moveDistance / segmentDistance
			progress += progressIncrement

			// 如果进度超过1，移动到下一段
			for progress >= 1 && pointIndex < len(route)-1 {
				// 累加已完成段的距离
				totalDistance += segmentDistance
				lastPos = endPoint

				progress -= 1
				pointIndex++

				if pointIndex < len(route)-1 {
					startPoint = route[pointIndex]
					endPoint = route[pointIndex+1]
					segmentDistance = s.locationService.CalculateDistance(startPoint, endPoint)
					if segmentDistance < 0.1 {
						progress = 1 // 继续跳到下一段
					}
				}
			}

			// 检查是否到达终点
			if pointIndex >= len(route)-1 {
				continue
			}

			// 在两点之间进行线性插值计算当前位置
			currentLat := startPoint.Latitude + (endPoint.Latitude-startPoint.Latitude)*progress
			currentLon := startPoint.Longitude + (endPoint.Longitude-startPoint.Longitude)*progress

			// 路线偏移：使用缓慢变化的偏移量，而不是每次随机
			if config.RouteVariation > 0 {
				// 每3秒缓慢更新一次目标偏移量
				if time.Since(lastOffsetUpdateTime) > 3*time.Second {

					targetOffsetLat := (rand.Float64()*2 - 1) * config.RouteVariation * 0.00001
					targetOffsetLon := (rand.Float64()*2 - 1) * config.RouteVariation * 0.00001
					// 平滑过渡到新偏移
					offsetLat = offsetLat*0.7 + targetOffsetLat*0.3
					offsetLon = offsetLon*0.7 + targetOffsetLon*0.3
					lastOffsetUpdateTime = time.Now()
				}
				currentLat += offsetLat
				currentLon += offsetLon
			}

			currentPoint := location.Coordinate{
				Latitude:  currentLat,
				Longitude: currentLon,
			}

			if lastPos.Latitude != 0 || lastPos.Longitude != 0 {
				dist := s.locationService.CalculateDistance(lastPos, currentPoint)
				if dist < 50 { // 避免异常大的距离跳跃
					totalDistance += dist
				}
			}
			lastPos = currentPoint

			// 设置位置
			err := s.deviceManager.SetSimLocation(currentPoint.Latitude, currentPoint.Longitude)
			if err != nil {
				s.log.Error("跑步服务", "设置位置失败: %v", err)
				s.emitEvent("run:error", err.Error())
			}

			// 更新统计信息
			s.mu.Lock()
			s.stats.PointIndex = pointIndex
			s.stats.CurrentLoop = currentLoop
			s.stats.TotalDistance = totalDistance
			s.stats.CurrentSpeed = currentSpeed
			s.stats.CurrentLat = currentPoint.Latitude
			s.stats.CurrentLon = currentPoint.Longitude
			s.mu.Unlock()

			// 发送位置更新事件
			s.emitEvent("run:update", s.GetStats())

			// 每10秒输出一次状态日志
			if time.Since(lastLogTime) >= 10*time.Second {
				s.log.Debug("跑步服务", "运行中: 距离=%.0fm, 速度=%.1fkm/h, 位置=(%.5f, %.5f), 圈数=%d/%d",
					totalDistance, currentSpeed, currentPoint.Latitude, currentPoint.Longitude, currentLoop, config.LoopCount)
				lastLogTime = time.Now()
			}
		}
	}
}

// emitEvent 发送事件到前端
func (s *Service) emitEvent(eventName string, data interface{}) {
	if s.wailsCtx != nil {
		runtime.EventsEmit(s.wailsCtx, eventName, data)
	}
}

// CalculateRouteStats 计算路线统计
func (s *Service) CalculateRouteStats(points []location.Coordinate) map[string]interface{} {
	if len(points) < 2 {
		return map[string]interface{}{
			"distance": 0,
			"points":   0,
		}
	}

	var totalDistance float64
	for i := 1; i < len(points); i++ {
		totalDistance += s.locationService.CalculateDistance(points[i-1], points[i])
	}

	return map[string]interface{}{
		"distance": totalDistance,
		"points":   len(points),
	}
}
