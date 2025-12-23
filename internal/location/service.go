package location

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Coordinate 坐标点
type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Route 路线
type Route struct {
	Name       string       `json:"name"`
	Points     []Coordinate `json:"points"`
	TotalKM    float64      `json:"totalKm"`
	CreateTime string       `json:"createTime"`
}

// Service 位置服务
type Service struct {
	mu         sync.RWMutex
	routes     []Route
	currentPos *Coordinate
}

// NewService 创建位置服务
func NewService() *Service {
	return &Service{
		routes: make([]Route, 0),
	}
}

// GetCurrentPosition 获取当前位置
func (s *Service) GetCurrentPosition() *Coordinate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentPos
}

// SetCurrentPosition 设置当前位置
func (s *Service) SetCurrentPosition(lat, lon float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentPos = &Coordinate{Latitude: lat, Longitude: lon}
}

// SaveRoute 保存路线
func (s *Service) SaveRoute(name string, points []Coordinate) (*Route, error) {
	if len(points) < 2 {
		return nil, fmt.Errorf("路线至少需要2个点")
	}

	route := Route{
		Name:       name,
		Points:     points,
		TotalKM:    calculateRouteDistance(points),
		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	s.mu.Lock()
	s.routes = append(s.routes, route)
	s.mu.Unlock()

	return &route, nil
}

// GetRoutes 获取所有路线
func (s *Service) GetRoutes() []Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.routes
}

// DeleteRoute 删除路线
func (s *Service) DeleteRoute(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= len(s.routes) {
		return fmt.Errorf("无效的路线索引")
	}

	s.routes = append(s.routes[:index], s.routes[index+1:]...)
	return nil
}

// InterpolateRoute 插值路线
func (s *Service) InterpolateRoute(points []Coordinate, targetDistance float64) []Coordinate {
	if len(points) < 2 {
		return points
	}

	var result []Coordinate
	result = append(result, points[0])

	for i := 1; i < len(points); i++ {
		start := points[i-1]
		end := points[i]
		distance := haversineDistance(start, end)

		if distance > targetDistance {
			// 需要插值
			numSegments := int(math.Ceil(distance / targetDistance))
			for j := 1; j <= numSegments; j++ {
				ratio := float64(j) / float64(numSegments)
				interpolated := Coordinate{
					Latitude:  start.Latitude + (end.Latitude-start.Latitude)*ratio,
					Longitude: start.Longitude + (end.Longitude-start.Longitude)*ratio,
				}
				result = append(result, interpolated)
			}
		} else {
			result = append(result, end)
		}
	}

	return result
}

// AddRandomOffset 给坐标添加随机偏移
func (s *Service) AddRandomOffset(coord Coordinate, maxOffsetMeters float64) Coordinate {
	// 随机偏移量
	latOffset := (rand.Float64()*2 - 1) * maxOffsetMeters / 111000.0
	lonOffset := (rand.Float64()*2 - 1) * maxOffsetMeters / (111000.0 * math.Cos(coord.Latitude*math.Pi/180))

	return Coordinate{
		Latitude:  coord.Latitude + latOffset,
		Longitude: coord.Longitude + lonOffset,
	}
}

// CalculateDistance 计算两点之间的距离
func (s *Service) CalculateDistance(p1, p2 Coordinate) float64 {
	return haversineDistance(p1, p2) * 1000 // 转换为米
}

// haversineDistance 使用Haversine公式计算两点之间的距离
func haversineDistance(p1, p2 Coordinate) float64 {
	const earthRadius = 6371.0 // 地球半径

	lat1Rad := p1.Latitude * math.Pi / 180
	lat2Rad := p2.Latitude * math.Pi / 180
	deltaLat := (p2.Latitude - p1.Latitude) * math.Pi / 180
	deltaLon := (p2.Longitude - p1.Longitude) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// calculateRouteDistance 计算路线总距离
func calculateRouteDistance(points []Coordinate) float64 {
	if len(points) < 2 {
		return 0
	}

	var total float64
	for i := 1; i < len(points); i++ {
		total += haversineDistance(points[i-1], points[i])
	}

	return total
}

// ConvertBD09ToWGS84 将百度坐标转换为WGS84
func (s *Service) ConvertBD09ToWGS84(coords []Coordinate) []Coordinate {
	return TransformCoordinates(coords, CoordBD09, CoordWGS84)
}

// ConvertGCJ02ToWGS84 将高德坐标转换为WGS84
func (s *Service) ConvertGCJ02ToWGS84(coords []Coordinate) []Coordinate {
	return TransformCoordinates(coords, CoordGCJ02, CoordWGS84)
}

// ConvertToWGS84 通用转换到WGS84
func (s *Service) ConvertToWGS84(coords []Coordinate, fromSystem string) []Coordinate {
	var from CoordSystem
	switch fromSystem {
	case "BD09", "bd09", "baidu":
		from = CoordBD09
	case "GCJ02", "gcj02", "gaode", "amap":
		from = CoordGCJ02
	default:
		return coords
	}
	return TransformCoordinates(coords, from, CoordWGS84)
}
