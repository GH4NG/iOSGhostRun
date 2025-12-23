package location

import (
	"math"
)

// 坐标系转换常量
const (
	xPi = math.Pi * 3000.0 / 180.0
	pi  = math.Pi
	a   = 6378245.0              // 长半轴
	ee  = 0.00669342162296594323 // 扁率
)

// CoordSystem 坐标系类型
type CoordSystem string

const (
	CoordWGS84 CoordSystem = "WGS84" // GPS/iOS 坐标系
	CoordGCJ02 CoordSystem = "GCJ02" // 国测局坐标系
	CoordBD09  CoordSystem = "BD09"  // 百度坐标系
)

// BD09ToGCJ02 百度坐标系(BD-09) -> 国测局坐标系(GCJ-02)
func BD09ToGCJ02(bdLat, bdLon float64) (float64, float64) {
	x := bdLon - 0.0065
	y := bdLat - 0.006
	z := math.Sqrt(x*x+y*y) - 0.00002*math.Sin(y*xPi)
	theta := math.Atan2(y, x) - 0.000003*math.Cos(x*xPi)
	gcjLon := z * math.Cos(theta)
	gcjLat := z * math.Sin(theta)
	return gcjLat, gcjLon
}

// GCJ02ToBD09 国测局坐标系(GCJ-02) -> 百度坐标系(BD-09)
func GCJ02ToBD09(gcjLat, gcjLon float64) (float64, float64) {
	z := math.Sqrt(gcjLon*gcjLon+gcjLat*gcjLat) + 0.00002*math.Sin(gcjLat*xPi)
	theta := math.Atan2(gcjLat, gcjLon) + 0.000003*math.Cos(gcjLon*xPi)
	bdLon := z*math.Cos(theta) + 0.0065
	bdLat := z*math.Sin(theta) + 0.006
	return bdLat, bdLon
}

// GCJ02ToWGS84 国测局坐标系(GCJ-02) -> GPS坐标系(WGS-84)
func GCJ02ToWGS84(gcjLat, gcjLon float64) (float64, float64) {
	if outOfChina(gcjLat, gcjLon) {
		return gcjLat, gcjLon
	}
	dLat := transformLat(gcjLon-105.0, gcjLat-35.0)
	dLon := transformLon(gcjLon-105.0, gcjLat-35.0)
	radLat := gcjLat / 180.0 * pi
	magic := math.Sin(radLat)
	magic = 1 - ee*magic*magic
	sqrtMagic := math.Sqrt(magic)
	dLat = (dLat * 180.0) / ((a * (1 - ee)) / (magic * sqrtMagic) * pi)
	dLon = (dLon * 180.0) / (a / sqrtMagic * math.Cos(radLat) * pi)
	mgLat := gcjLat + dLat
	mgLon := gcjLon + dLon
	return gcjLat*2 - mgLat, gcjLon*2 - mgLon
}

// WGS84ToGCJ02 GPS坐标系(WGS-84) -> 国测局坐标系(GCJ-02)
func WGS84ToGCJ02(wgsLat, wgsLon float64) (float64, float64) {
	if outOfChina(wgsLat, wgsLon) {
		return wgsLat, wgsLon
	}
	dLat := transformLat(wgsLon-105.0, wgsLat-35.0)
	dLon := transformLon(wgsLon-105.0, wgsLat-35.0)
	radLat := wgsLat / 180.0 * pi
	magic := math.Sin(radLat)
	magic = 1 - ee*magic*magic
	sqrtMagic := math.Sqrt(magic)
	dLat = (dLat * 180.0) / ((a * (1 - ee)) / (magic * sqrtMagic) * pi)
	dLon = (dLon * 180.0) / (a / sqrtMagic * math.Cos(radLat) * pi)
	mgLat := wgsLat + dLat
	mgLon := wgsLon + dLon
	return mgLat, mgLon
}

// BD09ToWGS84 百度坐标系(BD-09) -> GPS坐标系(WGS-84)
func BD09ToWGS84(bdLat, bdLon float64) (float64, float64) {
	gcjLat, gcjLon := BD09ToGCJ02(bdLat, bdLon)
	return GCJ02ToWGS84(gcjLat, gcjLon)
}

// WGS84ToBD09 GPS坐标系(WGS-84) -> 百度坐标系(BD-09)
func WGS84ToBD09(wgsLat, wgsLon float64) (float64, float64) {
	gcjLat, gcjLon := WGS84ToGCJ02(wgsLat, wgsLon)
	return GCJ02ToBD09(gcjLat, gcjLon)
}

// TransformCoordinate 通用坐标转换
func TransformCoordinate(lat, lon float64, from, to CoordSystem) (float64, float64) {
	if from == to {
		return lat, lon
	}

	// 先转换到 GCJ02
	var gcjLat, gcjLon float64
	switch from {
	case CoordWGS84:
		gcjLat, gcjLon = WGS84ToGCJ02(lat, lon)
	case CoordGCJ02:
		gcjLat, gcjLon = lat, lon
	case CoordBD09:
		gcjLat, gcjLon = BD09ToGCJ02(lat, lon)
	default:
		return lat, lon
	}

	// 再从 GCJ02 转换到目标坐标系
	switch to {
	case CoordWGS84:
		return GCJ02ToWGS84(gcjLat, gcjLon)
	case CoordGCJ02:
		return gcjLat, gcjLon
	case CoordBD09:
		return GCJ02ToBD09(gcjLat, gcjLon)
	default:
		return gcjLat, gcjLon
	}
}

// TransformCoordinatePoint 转换坐标点
func TransformCoordinatePoint(coord Coordinate, from, to CoordSystem) Coordinate {
	lat, lon := TransformCoordinate(coord.Latitude, coord.Longitude, from, to)
	return Coordinate{Latitude: lat, Longitude: lon}
}

// TransformCoordinates 批量转换坐标点
func TransformCoordinates(coords []Coordinate, from, to CoordSystem) []Coordinate {
	result := make([]Coordinate, len(coords))
	for i, coord := range coords {
		result[i] = TransformCoordinatePoint(coord, from, to)
	}
	return result
}

// 判断是否在中国境内
func outOfChina(lat, lon float64) bool {
	return lon < 72.004 || lon > 137.8347 || lat < 0.8293 || lat > 55.8271
}

// 转换纬度
func transformLat(x, y float64) float64 {
	ret := -100.0 + 2.0*x + 3.0*y + 0.2*y*y + 0.1*x*y + 0.2*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*pi) + 20.0*math.Sin(2.0*x*pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(y*pi) + 40.0*math.Sin(y/3.0*pi)) * 2.0 / 3.0
	ret += (160.0*math.Sin(y/12.0*pi) + 320*math.Sin(y*pi/30.0)) * 2.0 / 3.0
	return ret
}

// 转换经度
func transformLon(x, y float64) float64 {
	ret := 300.0 + x + 2.0*y + 0.1*x*x + 0.1*x*y + 0.1*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*pi) + 20.0*math.Sin(2.0*x*pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(x*pi) + 40.0*math.Sin(x/3.0*pi)) * 2.0 / 3.0
	ret += (150.0*math.Sin(x/12.0*pi) + 300.0*math.Sin(x/30.0*pi)) * 2.0 / 3.0
	return ret
}
