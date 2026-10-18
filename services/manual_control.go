package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// StartManualControl 从指定 WGS84 坐标开始手动定位，与路线任务共用生命周期和清理流程。
func (r *RunningService) StartManualControl(udid string, origin Point) error {
	return r.withIdleDeviceOperation(func() error {
		if strings.TrimSpace(udid) == "" {
			return fmt.Errorf("未选择设备")
		}
		if !validManualNumber(origin.Lat) || !validManualNumber(origin.Lon) ||
			math.Abs(origin.Lat) > 90 || math.Abs(origin.Lon) > 180 {
			return fmt.Errorf("起始经纬度无效")
		}
		ctx, cancel := context.WithCancel(context.Background())
		r.mu.Lock()
		r.state = StateManual
		r.udid = udid
		r.route = []Point{origin}
		r.currentIndex, r.currentLoop = 0, 0
		r.progress, r.distance, r.currentSpeed = 0, 0, 0
		r.startTime, r.endTime = time.Now(), time.Time{}
		r.pausedDuration = 0
		r.manualEast, r.manualNorth = 0, 0
		r.manualSpeed = 5
		r.manualInputAt = time.Now()
		r.cancel = cancel
		r.mu.Unlock()
		r.startLocationTask(udid, cancel, func() { r.manualLoop(ctx, udid, origin) }, func() {
			r.mu.Lock()
			r.state = StateIdle
			r.currentSpeed = 0
			r.endTime = time.Now()
			r.mu.Unlock()
			emitAppEvent("running:position", r.GetStatus())
		})
		Log.Info("RunningService", "已开启手动定位")
		return nil
	})
}

// SetManualDirection 更新摇杆方向（东、北，范围 -1..1）及速度（km/h），同时续租控制会话。
func (r *RunningService) SetManualDirection(east, north, speed float64) error {
	if !validManualNumber(east) || !validManualNumber(north) || !validManualNumber(speed) ||
		math.Abs(east) > 1 || math.Abs(north) > 1 || speed < 1 || speed > 20 {
		return fmt.Errorf("手动方向或速度无效，速度范围为 1–20 km/h")
	}
	if length := math.Hypot(east, north); length > 1 {
		east, north = east/length, north/length
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state != StateManual || appShuttingDown.Load() {
		return fmt.Errorf("手动定位未开启")
	}
	r.manualEast, r.manualNorth, r.manualSpeed = east, north, speed
	r.manualInputAt = time.Now()
	return nil
}

func validManualNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// moveManualPoint 使用球面正算，跨越日期变更线和极点时仍保持合法坐标。
func moveManualPoint(point Point, east, north, meters float64) Point {
	if meters == 0 || (east == 0 && north == 0) {
		return point
	}
	lat, lon := point.Lat*math.Pi/180, point.Lon*math.Pi/180
	bearing := math.Atan2(east, north)
	angle := meters / 6371000
	nextLat := math.Asin(math.Max(-1, math.Min(1,
		math.Sin(lat)*math.Cos(angle)+math.Cos(lat)*math.Sin(angle)*math.Cos(bearing))))
	nextLon := lon + math.Atan2(math.Sin(bearing)*math.Sin(angle)*math.Cos(lat),
		math.Cos(angle)-math.Sin(lat)*math.Sin(nextLat))
	return Point{Lat: nextLat * 180 / math.Pi, Lon: math.Mod(nextLon*180/math.Pi+540, 360) - 180}
}

func (r *RunningService) manualLoop(ctx context.Context, udid string, point Point) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	last := time.Now()
	first := true
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			// 设备写入阻塞后不补偿大步跳跃。
			seconds := math.Min(now.Sub(last).Seconds(), 0.2)
			last = now
			r.mu.Lock()
			east, north, speed, inputAt := r.manualEast, r.manualNorth, r.manualSpeed, r.manualInputAt
			active := r.state == StateManual
			r.mu.Unlock()
			if !active || ctx.Err() != nil {
				return
			}
			if now.Sub(inputAt) > 3*time.Second {
				Log.Warn("RunningService", "手动定位控制超时，已自动停止")
				return
			}
			// 输入续租中断后，先立即停步，再由会话超时释放资源。
			if now.Sub(inputAt) > 750*time.Millisecond {
				east, north = 0, 0
			}
			currentSpeed := speed * math.Hypot(east, north)
			meters := currentSpeed / 3.6 * seconds
			next := moveManualPoint(point, east, north, meters)
			if first || meters > 0 {
				if err := r.locationService.SetLocation(udid, next.Lat, next.Lon); err != nil {
					Log.Error("RunningService", fmt.Sprintf("手动定位失败: %v", err))
					emitAppEvent("running:error", fmt.Sprintf("手动定位失败: %v", err))
					return
				}
				first = false
			}
			if ctx.Err() != nil {
				return
			}
			point = next
			r.mu.Lock()
			if r.state != StateManual {
				r.mu.Unlock()
				return
			}
			r.route[0] = point
			r.distance += meters / 1000
			r.currentSpeed = currentSpeed
			r.mu.Unlock()
			emitAppEvent("running:position", r.GetStatus())
		}
	}
}
