package services

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Point 路线点
type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// RunningState 跑步状态
type RunningState string

const (
	StateIdle    RunningState = "idle"
	StateRunning RunningState = "running"
	StatePaused  RunningState = "paused"
)

// RunningStatus 跑步状态信息
type RunningStatus struct {
	State         RunningState `json:"state"`
	CurrentIndex  int          `json:"currentIndex"`
	TotalPoints   int          `json:"totalPoints"`
	CurrentLat    float64      `json:"currentLat"`
	CurrentLon    float64      `json:"currentLon"`
	Speed         float64      `json:"speed"`
	Distance      float64      `json:"distance"`
	ElapsedTimeMs int64        `json:"elapsedTimeMs"`
}

// RunningService 跑步模拟服务
type RunningService struct {
	mu              sync.Mutex
	state           RunningState
	route           []Point
	currentIndex    int
	speed           float64 // km/h
	speedVariance   float64 // 速度随机波动百分比 0-1
	routeOffset     float64 // 路线偏移距离（米）
	udid            string
	cancel          context.CancelFunc
	pauseChan       chan struct{}
	resumeChan      chan struct{}
	locationService *LocationService
	distance        float64
	startTime       time.Time
	pausedDuration  time.Duration
	lastPauseTime   time.Time
}

// NewRunningService 创建跑步服务
func NewRunningService() *RunningService {
	return &RunningService{
		state:           StateIdle,
		speed:           8.0,
		speedVariance:   0.1,
		routeOffset:     2.0,
		locationService: &LocationService{},
		pauseChan:       make(chan struct{}),
		resumeChan:      make(chan struct{}),
	}
}

// StartRun 开始跑步
func (r *RunningService) StartRun(udid string, route []Point, speed float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == StateRunning {
		return nil
	}

	Log.Info("RunningService", fmt.Sprintf("Starting run for device %s with %d points at %.2f km/h", udid, len(route), speed))
	r.udid = udid
	r.route = route
	r.speed = speed
	r.currentIndex = 0
	r.state = StateRunning
	r.distance = 0
	r.startTime = time.Now()
	r.pausedDuration = 0

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	go r.runLoop(ctx)

	return nil
}

// PauseRun 暂停跑步
func (r *RunningService) PauseRun() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == StateRunning {
		Log.Info("RunningService", "Pausing run")
		r.state = StatePaused
		r.lastPauseTime = time.Now()
		r.pauseChan <- struct{}{}
	}
}

// ResumeRun 恢复跑步
func (r *RunningService) ResumeRun() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == StatePaused {
		Log.Info("RunningService", "Resuming run")
		r.state = StateRunning
		r.pausedDuration += time.Since(r.lastPauseTime)
		r.resumeChan <- struct{}{}
	}
}

// StopRun 停止跑步
func (r *RunningService) StopRun() {
	r.mu.Lock()
	defer r.mu.Unlock()

	Log.Info("RunningService", "Stopping run")
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.state = StateIdle
	r.currentIndex = 0

	// 重置设备位置
	if r.udid != "" {
		Log.Info("RunningService", "Resetting location for device: "+r.udid)
		_ = r.locationService.ResetLocation(r.udid)
	}
}

// SetSpeed 设置速度
func (r *RunningService) SetSpeed(speed float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.speed = speed
}

// SetRandomization 设置随机化参数
func (r *RunningService) SetRandomization(speedVariance, routeOffset float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.speedVariance = speedVariance
	r.routeOffset = routeOffset
}

// GetStatus 获取当前状态
func (r *RunningService) GetStatus() RunningStatus {
	r.mu.Lock()
	defer r.mu.Unlock()

	var currentLat, currentLon float64
	if r.currentIndex < len(r.route) {
		currentLat = r.route[r.currentIndex].Lat
		currentLon = r.route[r.currentIndex].Lon
	}

	var elapsed time.Duration
	if r.state != StateIdle {
		elapsed = time.Since(r.startTime) - r.pausedDuration
		if r.state == StatePaused {
			elapsed -= time.Since(r.lastPauseTime)
		}
	}

	return RunningStatus{
		State:         r.state,
		CurrentIndex:  r.currentIndex,
		TotalPoints:   len(r.route),
		CurrentLat:    currentLat,
		CurrentLon:    currentLon,
		Speed:         r.speed,
		Distance:      r.distance,
		ElapsedTimeMs: elapsed.Milliseconds(),
	}
}

// runLoop 跑步循环
func (r *RunningService) runLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.pauseChan:
			// 等待恢复
			select {
			case <-ctx.Done():
				return
			case <-r.resumeChan:
				// 继续
			}
		default:
			r.mu.Lock()
			if r.currentIndex >= len(r.route) {
				r.state = StateIdle
				r.mu.Unlock()
				// 发送完成事件
				application.Get().Event.Emit("running:completed", r.GetStatus())
				return
			}

			point := r.route[r.currentIndex]
			speed := r.speed

			// 应用速度随机波动
			if r.speedVariance > 0 {
				variance := (rand.Float64()*2 - 1) * r.speedVariance
				speed = speed * (1 + variance)
			}

			// 应用路线偏移
			lat, lon := point.Lat, point.Lon
			if r.routeOffset > 0 {
				// 随机偏移（米转换为经纬度）
				latOffset := (rand.Float64()*2 - 1) * r.routeOffset / 111000
				lonOffset := (rand.Float64()*2 - 1) * r.routeOffset / (111000 * math.Cos(lat*math.Pi/180))
				lat += latOffset
				lon += lonOffset
			}

			// 计算到下一个点的距离
			if r.currentIndex > 0 {
				prevPoint := r.route[r.currentIndex-1]
				r.distance += haversine(prevPoint.Lat, prevPoint.Lon, point.Lat, point.Lon)
			}

			r.currentIndex++
			r.mu.Unlock()

			// 设置设备位置
			err := r.locationService.SetLocation(r.udid, lat, lon)
			if err != nil {
				Log.Error("RunningService", fmt.Sprintf("Failed to set location: %v", err))
			} else {
				// 发送位置更新事件
				application.Get().Event.Emit("running:position", map[string]interface{}{
					"lat":   lat,
					"lon":   lon,
					"index": r.currentIndex,
				})

				if r.currentIndex%10 == 0 {
					// 每10个点记录一次Debug日志
					Log.Debug("RunningService", fmt.Sprintf("Position update: %d/%d (%.6f, %.6f)", r.currentIndex, len(r.route), lat, lon))
				}
			}

			// 根据速度计算等待时间（假设每个点间隔约10米）
			interval := 10.0 / (speed * 1000 / 3600) * 1000 // 毫秒
			time.Sleep(time.Duration(interval) * time.Millisecond)
		}
	}
}

// haversine 计算两点间距离（公里）
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // 地球半径（公里）
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
