package services

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"
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

type locationProvider interface {
	SetLocation(udid string, lat, lon float64) error
	ResetLocation(udid string) error
}

// RunningService 跑步模拟服务
type RunningService struct {
	lifecycleMu     sync.Mutex
	mu              sync.Mutex
	runWG           sync.WaitGroup
	state           RunningState
	route           []Point
	currentIndex    int
	speed           float64       // 目标速度 km/h
	currentSpeed    float64       // 当前实时速度 km/h
	speedVariance   float64       // 速度波动比例，0.1 表示目标速度的 ±10%
	routeOffset     float64       // 路线横向偏移上限，单位米
	loopCount       int           // 循环次数 0=无限
	updateInterval  time.Duration // 位置更新间隔
	udid            string
	cancel          context.CancelFunc
	locationService locationProvider
	distance        float64
	startTime       time.Time
	endTime         time.Time
	pausedDuration  time.Duration
	lastPauseTime   time.Time
	progress        float64 // 当前段内的进度 0-1
	currentLoop     int     // 当前圈数
}

// NewRunningService 创建跑步服务
func NewRunningService(locationService *LocationService) *RunningService {
	if locationService == nil {
		locationService = NewLocationService()
	}

	return &RunningService{
		state:           StateIdle,
		speed:           8.0,
		currentSpeed:    8.0,
		speedVariance:   0.1,
		routeOffset:     2.0,
		updateInterval:  100 * time.Millisecond,
		loopCount:       1,
		locationService: locationService,
	}
}

// StartRun 开始跑步
func (r *RunningService) StartRun(udid string, route []Point, speed float64) error {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()

	if strings.TrimSpace(udid) == "" {
		return fmt.Errorf("未选择设备")
	}

	if len(route) < 2 {
		return fmt.Errorf("路线点数量不足，至少需要 2 个点")
	}

	if speed <= 0 || math.IsNaN(speed) || math.IsInf(speed, 0) {
		return fmt.Errorf("速度必须大于 0")
	}
	for i, point := range route {
		if math.IsNaN(point.Lat) || math.IsInf(point.Lat, 0) || math.IsNaN(point.Lon) || math.IsInf(point.Lon, 0) ||
			point.Lat < -90 || point.Lat > 90 || point.Lon < -180 || point.Lon > 180 {
			return fmt.Errorf("第 %d 个路线点的经纬度无效", i+1)
		}
	}

	r.mu.Lock()
	if r.state != StateIdle {
		r.mu.Unlock()
		return fmt.Errorf("已有跑步任务，请先停止当前任务")
	}
	r.mu.Unlock()
	// 完成事件发出后，仍需等上一轮的定位连接释放完毕才能重新启动。
	r.runWG.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()

	routeCopy := append([]Point(nil), route...)

	Log.Info("RunningService", fmt.Sprintf("为设备 %s 开始跑步，%d 个路线点，速度 %.2f km/h", udid, len(route), speed))
	r.udid = udid
	r.route = routeCopy
	r.speed = speed
	r.currentSpeed = speed
	r.currentIndex = 0
	r.state = StateRunning
	r.distance = 0
	r.currentLoop = 1
	r.progress = 0
	r.startTime = time.Now()
	r.endTime = time.Time{}
	r.pausedDuration = 0

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	r.runWG.Add(1)
	go func() {
		defer r.runWG.Done()
		defer cancel()
		defer func() {
			if r.locationService != nil {
				if err := r.locationService.ResetLocation(udid); err != nil {
					Log.Warn("RunningService", fmt.Sprintf("重置设备位置失败: %v", err))
				}
			}
		}()
		r.runLoop(ctx)
	}()

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
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	r.mu.Lock()
	Log.Info("RunningService", "停止跑步")
	cancel := r.cancel
	r.cancel = nil
	if r.state != StateIdle {
		r.endTime = time.Now()
		if r.state == StatePaused {
			r.pausedDuration += r.endTime.Sub(r.lastPauseTime)
		}
	}
	r.state = StateIdle
	r.currentIndex = 0
	r.progress = 0
	r.currentLoop = 0
	r.currentSpeed = r.speed
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	// 等待 runLoop 完全退出
	r.runWG.Wait()
}

// SetSpeed 设置速度
func (r *RunningService) SetSpeed(speed float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if speed <= 0 || math.IsNaN(speed) || math.IsInf(speed, 0) {
		return
	}
	r.speed = speed
	r.currentSpeed = speed
}

// SetRandomization 设置随机化参数
func (r *RunningService) SetRandomization(speedVariance, routeOffset float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if speedVariance < 0 || math.IsNaN(speedVariance) || math.IsInf(speedVariance, 0) {
		speedVariance = 0
	}
	if speedVariance > 0.3 {
		speedVariance = 0.3
	}
	if routeOffset < 0 || math.IsNaN(routeOffset) || math.IsInf(routeOffset, 0) {
		routeOffset = 0
	}
	if routeOffset > 10 {
		routeOffset = 10
	}
	r.speedVariance = speedVariance
	r.routeOffset = routeOffset
}

// SetLoopCount 设置循环圈数
func (r *RunningService) SetLoopCount(count int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if count >= 0 && count <= 999 {
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
	if !r.startTime.IsZero() {
		end := time.Now()
		if !r.endTime.IsZero() {
			end = r.endTime
		}
		elapsed = end.Sub(r.startTime) - r.pausedDuration
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
		Speed:         r.currentSpeed,
		Distance:      r.distance,
		ElapsedTimeMs: elapsed.Milliseconds(),
		Progress:      r.progress,
		LoopCount:     r.loopCount,
		CurrentLoop:   r.currentLoop,
	}
}

// runLoop 跑步循环 - 更精确的速度控制
func (r *RunningService) runLoop(ctx context.Context) {
	r.mu.Lock()
	interval := r.updateInterval
	if interval <= 0 {
		interval = 100 * time.Millisecond
		r.updateInterval = interval
		Log.Warn("RunningService", fmt.Sprintf("检测到无效更新间隔，使用默认值: %s", interval))
	}
	r.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	pointIndex := 0
	currentLoop := 1
	var totalDistanceKM float64
	lastLogTime := time.Now()
	lastStepTime := time.Now()
	progress := 0.0
	r.mu.Lock()
	smoothedSpeed := r.speed
	r.mu.Unlock()

	// 每次跑步选择不同的缓慢配速周期，叠加较小的次级波动。
	speedPhase := rand.Float64() * 2 * math.Pi
	speedFreq := 0.15 + rand.Float64()*0.15
	speedPhase2 := rand.Float64() * 2 * math.Pi
	speedFreq2 := speedFreq * (2.0 + rand.Float64())

	var targetOffsetMeters, lateralOffsetMeters float64
	var offsetEastMeters, offsetNorthMeters float64
	nextOffsetChange := time.Now().Add(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			stepDuration := now.Sub(lastStepTime)
			lastStepTime = now

			r.mu.Lock()
			state := r.state
			route := r.route
			baseSpeed := r.speed
			speedVariance := r.speedVariance
			routeOffset := r.routeOffset
			loopCount := r.loopCount
			udid := r.udid
			startTime := r.startTime
			pausedDuration := r.pausedDuration
			lastPauseTime := r.lastPauseTime
			locationSvc := r.locationService
			r.mu.Unlock()

			if state == StatePaused {
				lastStepTime = time.Now()
				continue
			}

			if state != StateRunning {
				return
			}

			if len(route) < 2 {
				Log.Error("RunningService", "路线点不足，终止跑步")
				r.mu.Lock()
				r.state = StateIdle
				r.endTime = time.Now()
				r.mu.Unlock()
				emitAppEvent("running:error", "路线点不足，至少需要 2 个点")
				return
			}

			// 检查是否完成
			if pointIndex >= len(route)-1 {
				if loopCount > 0 && currentLoop >= loopCount {
					r.mu.Lock()
					r.state = StateIdle
					r.endTime = time.Now()
					r.distance = totalDistanceKM
					r.currentLoop = currentLoop
					r.currentIndex = len(route) - 1
					r.progress = 1
					r.mu.Unlock()
					Log.Info("RunningService", fmt.Sprintf("跑步完成！总距离: %.0fm, 圈数: %d", totalDistanceKM*1000, currentLoop))
					emitAppEvent("running:completed", r.GetStatus())
					return
				}
				// 开始新的循环
				pointIndex = 0
				progress = 0
				currentLoop++
				Log.Info("RunningService", fmt.Sprintf("开始第 %d 圈", currentLoop))
			}

			stepSeconds := stepDuration.Seconds()
			desiredSpeed := baseSpeed
			if speedVariance > 0 {
				elapsed := now.Sub(startTime).Seconds() - pausedDuration.Seconds()
				paceWave := 0.8*math.Sin(elapsed*speedFreq+speedPhase) + 0.2*math.Sin(elapsed*speedFreq2+speedPhase2)
				desiredSpeed *= 1 + speedVariance*paceWave
			}
			desiredSpeed = math.Max(0.5, desiredSpeed)
			// 限制每秒速度变化，避免定位点突然加速或减速。
			maxSpeedChange := 1.8 * stepSeconds
			smoothedSpeed += math.Max(-maxSpeedChange, math.Min(maxSpeedChange, desiredSpeed-smoothedSpeed))
			currentSpeed := smoothedSpeed

			// 按真实经过时间推进，避免设备位置注入耗时导致实际速度偏慢。
			remainingMoveKM := currentSpeed * stepSeconds / 3600

			// 沿路线推进本步距离
			previousLoop := currentLoop
			var movedKM float64
			pointIndex, progress, currentLoop, movedKM = advanceAlongRoute(route, pointIndex, progress, currentLoop, loopCount, remainingMoveKM)
			totalDistanceKM += movedKM
			if currentLoop > previousLoop {
				Log.Info("RunningService", fmt.Sprintf("开始第 %d 圈", currentLoop))
			}

			// 检查是否到达终点
			if pointIndex >= len(route)-1 {
				continue
			}

			startPoint := route[pointIndex]
			endPoint := route[pointIndex+1]

			// 在两点之间进行线性插值计算当前位置
			currentLat := startPoint.Lat + (endPoint.Lat-startPoint.Lat)*progress
			currentLon := startPoint.Lon + (endPoint.Lon-startPoint.Lon)*progress

			// 以米计算路线的横向偏移，并在目标变化和转弯时连续过渡。
			if routeOffset > 0 && !now.Before(nextOffsetChange) {
				targetOffsetMeters = (rand.Float64()*2 - 1) * routeOffset
				nextOffsetChange = now.Add(time.Duration(5+rand.Intn(4)) * time.Second)
			} else if routeOffset == 0 {
				targetOffsetMeters = 0
			}
			targetOffsetMeters = math.Max(-routeOffset, math.Min(routeOffset, targetOffsetMeters))
			lateralBlend := 1 - math.Exp(-stepSeconds/2.5)
			lateralOffsetMeters += (targetOffsetMeters - lateralOffsetMeters) * lateralBlend
			latMetersPerDegree := 111320.0
			lonMetersPerDegree := math.Max(latMetersPerDegree*math.Abs(math.Cos(currentLat*math.Pi/180)), 1)
			segmentEast := (endPoint.Lon - startPoint.Lon) * lonMetersPerDegree
			segmentNorth := (endPoint.Lat - startPoint.Lat) * latMetersPerDegree
			segmentLength := math.Hypot(segmentEast, segmentNorth)
			var desiredOffsetEast, desiredOffsetNorth float64
			if segmentLength > 0 {
				desiredOffsetEast = -segmentNorth / segmentLength * lateralOffsetMeters
				desiredOffsetNorth = segmentEast / segmentLength * lateralOffsetMeters
			}
			turnBlend := 1 - math.Exp(-stepSeconds/0.8)
			offsetEastMeters += (desiredOffsetEast - offsetEastMeters) * turnBlend
			offsetNorthMeters += (desiredOffsetNorth - offsetNorthMeters) * turnBlend
			currentLat += offsetNorthMeters / latMetersPerDegree
			currentLon += offsetEastMeters / lonMetersPerDegree

			currentPoint := Point{
				Lat: currentLat,
				Lon: currentLon,
			}

			// 设置位置
			if locationSvc != nil {
				err := locationSvc.SetLocation(udid, currentPoint.Lat, currentPoint.Lon)
				if err != nil {
					Log.Error("RunningService", fmt.Sprintf("设置位置失败: %v", err))
					r.mu.Lock()
					r.state = StateIdle
					r.endTime = time.Now()
					r.mu.Unlock()
					emitAppEvent("running:error", err.Error())
					return
				}
			}

			// 更新统计信息
			r.mu.Lock()
			if r.state != StateRunning || ctx.Err() != nil {
				r.mu.Unlock()
				continue
			}
			r.currentIndex = pointIndex
			r.currentLoop = currentLoop
			r.distance = totalDistanceKM
			r.currentSpeed = currentSpeed
			r.progress = progress
			r.mu.Unlock()

			// 计算经过的时间
			var elapsed time.Duration
			if state != StateIdle {
				elapsed = time.Since(startTime) - pausedDuration
				if state == StatePaused {
					elapsed -= time.Since(lastPauseTime)
				}
			}

			emitAppEvent("running:position", RunningStatus{
				State:         state,
				CurrentLat:    currentPoint.Lat,
				CurrentLon:    currentPoint.Lon,
				Speed:         currentSpeed,
				Distance:      totalDistanceKM,
				CurrentLoop:   currentLoop,
				CurrentIndex:  pointIndex,
				TotalPoints:   len(route),
				ElapsedTimeMs: elapsed.Milliseconds(),
				Progress:      progress,
			})

			// 每10秒输出一次状态日志
			if time.Since(lastLogTime) >= 10*time.Second {
				Log.Debug("RunningService", fmt.Sprintf("跑步中：距离=%.0fm，速度=%.1fkm/h，位置=(%.5f, %.5f)，路线点=%d/%d，圈数=%d/%d",
					totalDistanceKM*1000, currentSpeed, currentPoint.Lat, currentPoint.Lon, pointIndex+1, len(route), currentLoop, loopCount))
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

func advanceAlongRoute(route []Point, pointIndex int, progress float64, currentLoop int, loopCount int, moveKM float64) (int, float64, int, float64) {
	moved := 0.0
	for moveKM > 0 {
		if pointIndex >= len(route)-1 {
			break
		}

		startPoint := route[pointIndex]
		endPoint := route[pointIndex+1]
		segmentDistanceKM := haversine(startPoint.Lat, startPoint.Lon, endPoint.Lat, endPoint.Lon)

		if segmentDistanceKM < 0.0001 {
			pointIndex++
			progress = 0
			continue
		}

		remainingInSegmentKM := segmentDistanceKM * (1 - progress)
		if moveKM < remainingInSegmentKM {
			progress += moveKM / segmentDistanceKM
			moved += moveKM
			moveKM = 0
			continue
		}

		moved += remainingInSegmentKM
		moveKM -= remainingInSegmentKM
		pointIndex++
		progress = 0

		if pointIndex >= len(route)-1 {
			if loopCount > 0 && currentLoop >= loopCount {
				break
			}
			pointIndex = 0
			currentLoop++
		}
	}
	return pointIndex, progress, currentLoop, moved
}
