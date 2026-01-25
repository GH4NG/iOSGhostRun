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
	Progress      float64      `json:"progress"`    // 当前段内的进度 0-1
	LoopCount     int          `json:"loopCount"`   // 循环次数
	CurrentLoop   int          `json:"currentLoop"` // 当前圈数
}

// RunningService 跑步模拟服务
type RunningService struct {
	mu              sync.Mutex
	state           RunningState
	route           []Point
	currentIndex    int
	speed           float64       // km/h
	speedVariance   float64       // 速度变化范围 km/h
	routeOffset     float64       // 路线偏移距离
	loopCount       int           // 循环次数 0=无限
	updateInterval  time.Duration // 位置更新间隔
	udid            string
	cancel          context.CancelFunc
	locationService *LocationService
	distance        float64
	startTime       time.Time
	pausedDuration  time.Duration
	lastPauseTime   time.Time
	progress        float64 // 当前段内的进度 0-1
	currentLoop     int     // 当前圈数
}

// NewRunningService 创建跑步服务
func NewRunningService() *RunningService {
	return &RunningService{
		state:           StateIdle,
		speed:           8.0,
		speedVariance:   1.0,
		routeOffset:     3.0,
		updateInterval:  100 * time.Millisecond,
		loopCount:       1,
		locationService: &LocationService{},
	}
}

// StartRun 开始跑步
func (r *RunningService) StartRun(udid string, route []Point, speed float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == StateRunning {
		return nil
	}

	Log.Info("RunningService", fmt.Sprintf("为设备 %s 开始跑步，%d 个路线点，速度 %.2f km/h", udid, len(route), speed))
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
		Log.Info("RunningService", "暂停跑步")
		r.state = StatePaused
		r.lastPauseTime = time.Now()
	}
}

// ResumeRun 恢复跑步
func (r *RunningService) ResumeRun() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == StatePaused {
		Log.Info("RunningService", "恢复跑步")
		r.state = StateRunning
		r.pausedDuration += time.Since(r.lastPauseTime)
	}
}

// StopRun 停止跑步
func (r *RunningService) StopRun() {
	r.mu.Lock()
	defer r.mu.Unlock()

	Log.Info("RunningService", "停止跑步")
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.state = StateIdle
	r.currentIndex = 0

	// 重置设备位置
	if r.udid != "" {
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

// SetLoopCount 设置循环圈数
func (r *RunningService) SetLoopCount(count int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if count > 0 && count <= 999 {
		r.loopCount = count
	}
}

// GetStatus 获取当前状态
func (r *RunningService) GetStatus() RunningStatus {
	r.mu.Lock()
	defer r.mu.Unlock()

	var currentLat, currentLon float64
	if r.currentIndex < len(r.route) {
		// 线性插值计算当前位置
		if r.currentIndex < len(r.route)-1 {
			startPoint := r.route[r.currentIndex]
			endPoint := r.route[r.currentIndex+1]
			currentLat = startPoint.Lat + (endPoint.Lat-startPoint.Lat)*r.progress
			currentLon = startPoint.Lon + (endPoint.Lon-startPoint.Lon)*r.progress
		} else {
			currentLat = r.route[r.currentIndex].Lat
			currentLon = r.route[r.currentIndex].Lon
		}
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
		Progress:      r.progress,
		LoopCount:     r.loopCount,
		CurrentLoop:   r.currentLoop,
	}
}

// runLoop 跑步循环 - 更精确的速度控制
func (r *RunningService) runLoop(ctx context.Context) {
	ticker := time.NewTicker(r.updateInterval)
	defer ticker.Stop()

	pointIndex := 0
	currentLoop := 1
	var totalDistance float64
	var lastPos Point
	lastLogTime := time.Now()
	progress := 0.0

	var offsetLat, offsetLon float64
	lastOffsetUpdateTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.mu.Lock()
			state := r.state
			config := r
			route := r.route
			r.mu.Unlock()

			if state == StatePaused {
				continue
			}

			if state != StateRunning {
				return
			}

			// 检查是否完成
			if pointIndex >= len(route)-1 {
				if config.loopCount > 0 && currentLoop >= config.loopCount {
					r.mu.Lock()
					r.state = StateIdle
					r.mu.Unlock()
					Log.Info("RunningService", fmt.Sprintf("跑步完成！总距离: %.0fm, 圈数: %d", totalDistance, currentLoop))
					application.Get().Event.Emit("running:completed", r.GetStatus())
					return
				}
				// 开始新的循环
				pointIndex = 0
				progress = 0
				currentLoop++
				Log.Info("RunningService", fmt.Sprintf("开始第 %d 圈", currentLoop))
			}

			// 获取当前段的起点和终点
			startPoint := route[pointIndex]
			endPoint := route[pointIndex+1]

			currentSpeed := config.speed
			if config.speedVariance > 0 {
				// 使用正弦函数实现速度的平滑波动
				elapsed := time.Since(r.startTime).Seconds()
				variation := math.Sin(elapsed*0.5) * config.speedVariance * 0.5
				currentSpeed += variation
				if currentSpeed < 0.5 {
					currentSpeed = 0.5
				}
			}

			// 计算两点之间的距离
			segmentDistance := haversine(startPoint.Lat, startPoint.Lon, endPoint.Lat, endPoint.Lon)
			if segmentDistance < 0.0001 {
				// 两点太近，直接跳到下一段
				pointIndex++
				progress = 0
				continue
			}

			// 速度 km/h 转换为 km/ms，然后计算这个时间间隔内移动的距离
			speedKMMS := currentSpeed / (3600 * 1000)
			moveDistance := speedKMMS * float64(r.updateInterval.Milliseconds())

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
					segmentDistance = haversine(startPoint.Lat, startPoint.Lon, endPoint.Lat, endPoint.Lon)
					if segmentDistance < 0.0001 {
						progress = 1 // 继续跳到下一段
					}
				}
			}

			// 检查是否到达终点
			if pointIndex >= len(route)-1 {
				continue
			}

			// 在两点之间进行线性插值计算当前位置
			currentLat := startPoint.Lat + (endPoint.Lat-startPoint.Lat)*progress
			currentLon := startPoint.Lon + (endPoint.Lon-startPoint.Lon)*progress

			// 路线偏移：使用缓慢变化的偏移量，而不是每次随机
			if config.routeOffset > 0 {
				// 每3秒缓慢更新一次目标偏移量
				if time.Since(lastOffsetUpdateTime) > 3*time.Second {
					targetOffsetLat := (rand.Float64()*2 - 1) * config.routeOffset * 0.00001
					targetOffsetLon := (rand.Float64()*2 - 1) * config.routeOffset * 0.00001
					// 平滑过渡到新偏移
					offsetLat = offsetLat*0.7 + targetOffsetLat*0.3
					offsetLon = offsetLon*0.7 + targetOffsetLon*0.3
					lastOffsetUpdateTime = time.Now()
				}
				currentLat += offsetLat
				currentLon += offsetLon
			}

			currentPoint := Point{
				Lat: currentLat,
				Lon: currentLon,
			}

			if lastPos.Lat != 0 || lastPos.Lon != 0 {
				dist := haversine(lastPos.Lat, lastPos.Lon, currentPoint.Lat, currentPoint.Lon)
				if dist < 0.05 { // 避免异常大的距离跳跃
					totalDistance += dist
				}
			}
			lastPos = currentPoint

			// 设置位置
			err := r.locationService.SetLocation(config.udid, currentPoint.Lat, currentPoint.Lon)
			if err != nil {
				Log.Error("RunningService", fmt.Sprintf("设置位置失败: %v", err))
				application.Get().Event.Emit("running:error", err.Error())
			}

			// 更新统计信息
			r.mu.Lock()
			r.currentIndex = pointIndex
			r.currentLoop = currentLoop
			r.distance = totalDistance
			r.speed = currentSpeed
			r.progress = progress
			r.mu.Unlock()

			// 计算经过的时间
			var elapsed time.Duration
			if state != StateIdle {
				elapsed = time.Since(r.startTime) - r.pausedDuration
				if state == StatePaused {
					elapsed -= time.Since(r.lastPauseTime)
				}
			}

			application.Get().Event.Emit("running:position", RunningStatus{
				State:         state,
				CurrentLat:    currentPoint.Lat,
				CurrentLon:    currentPoint.Lon,
				Speed:         currentSpeed,
				Distance:      totalDistance,
				CurrentLoop:   currentLoop,
				CurrentIndex:  pointIndex,
				TotalPoints:   len(route),
				ElapsedTimeMs: elapsed.Milliseconds(),
				Progress:      progress,
			})

			// 每10秒输出一次状态日志
			if time.Since(lastLogTime) >= 10*time.Second {
				Log.Debug("RunningService", fmt.Sprintf("跑步中：距离=%.0fm，速度=%.1fkm/h，位置=(%.5f, %.5f)，圈数=%d/%d",
					totalDistance, currentSpeed, currentPoint.Lat, currentPoint.Lon, currentLoop, config.loopCount))
				lastLogTime = time.Now()
			}
		}
	}
}

// haversine 计算两点间距离
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // 地球半径
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
