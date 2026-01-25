// 坐标系转换常量
const xPi = (Math.PI * 3000.0) / 180.0
const pi = Math.PI
const a = 6378245.0 // 长半轴
const ee = 0.00669342162296594323 // 扁率

// 坐标系类型
export enum CoordSystem {
    WGS84 = 'WGS84', // GPS/iOS
    GCJ02 = 'GCJ02', // 国测局
    BD09 = 'BD09' // 百度
}

// 坐标点
export interface Coordinate {
    Latitude: number
    Longitude: number
}

// BD09 -> GCJ02
export function BD09ToGCJ02(bdLat: number, bdLon: number): [number, number] {
    const x = bdLon - 0.0065
    const y = bdLat - 0.006
    const z = Math.sqrt(x * x + y * y) - 0.00002 * Math.sin(y * xPi)
    const theta = Math.atan2(y, x) - 0.000003 * Math.cos(x * xPi)
    const gcjLon = z * Math.cos(theta)
    const gcjLat = z * Math.sin(theta)
    return [gcjLat, gcjLon]
}

// GCJ02 -> BD09
export function GCJ02ToBD09(gcjLat: number, gcjLon: number): [number, number] {
    const z = Math.sqrt(gcjLon * gcjLon + gcjLat * gcjLat) + 0.00002 * Math.sin(gcjLat * xPi)
    const theta = Math.atan2(gcjLat, gcjLon) + 0.000003 * Math.cos(gcjLon * xPi)
    const bdLon = z * Math.cos(theta) + 0.0065
    const bdLat = z * Math.sin(theta) + 0.006
    return [bdLat, bdLon]
}

// GCJ02 -> WGS84
export function GCJ02ToWGS84(gcjLat: number, gcjLon: number): [number, number] {
    if (outOfChina(gcjLat, gcjLon)) {
        return [gcjLat, gcjLon]
    }
    let dLat = transformLat(gcjLon - 105.0, gcjLat - 35.0)
    let dLon = transformLon(gcjLon - 105.0, gcjLat - 35.0)
    const radLat = (gcjLat / 180.0) * pi
    let magic = Math.sin(radLat)
    magic = 1 - ee * magic * magic
    const sqrtMagic = Math.sqrt(magic)
    dLat = (dLat * 180.0) / (((a * (1 - ee)) / (magic * sqrtMagic)) * pi)
    dLon = (dLon * 180.0) / ((a / sqrtMagic) * Math.cos(radLat) * pi)
    const mgLat = gcjLat + dLat
    const mgLon = gcjLon + dLon
    return [gcjLat * 2 - mgLat, gcjLon * 2 - mgLon]
}

// WGS84 -> GCJ02
export function WGS84ToGCJ02(wgsLat: number, wgsLon: number): [number, number] {
    if (outOfChina(wgsLat, wgsLon)) {
        return [wgsLat, wgsLon]
    }
    let dLat = transformLat(wgsLon - 105.0, wgsLat - 35.0)
    let dLon = transformLon(wgsLon - 105.0, wgsLat - 35.0)
    const radLat = (wgsLat / 180.0) * pi
    let magic = Math.sin(radLat)
    magic = 1 - ee * magic * magic
    const sqrtMagic = Math.sqrt(magic)
    dLat = (dLat * 180.0) / (((a * (1 - ee)) / (magic * sqrtMagic)) * pi)
    dLon = (dLon * 180.0) / ((a / sqrtMagic) * Math.cos(radLat) * pi)
    const mgLat = wgsLat + dLat
    const mgLon = wgsLon + dLon
    return [mgLat, mgLon]
}

// BD09 -> WGS84
export function BD09ToWGS84(bdLat: number, bdLon: number): [number, number] {
    const [gcjLat, gcjLon] = BD09ToGCJ02(bdLat, bdLon)
    return GCJ02ToWGS84(gcjLat, gcjLon)
}

// WGS84 -> BD09
export function WGS84ToBD09(wgsLat: number, wgsLon: number): [number, number] {
    const [gcjLat, gcjLon] = WGS84ToGCJ02(wgsLat, wgsLon)
    return GCJ02ToBD09(gcjLat, gcjLon)
}

// 通用坐标转换
export function TransformCoordinate(lat: number, lon: number, from: CoordSystem, to: CoordSystem): [number, number] {
    if (from === to) {
        return [lat, lon]
    }

    let gcjLat: number
    let gcjLon: number

    switch (from) {
        case CoordSystem.WGS84:
            ;[gcjLat, gcjLon] = WGS84ToGCJ02(lat, lon)
            break
        case CoordSystem.GCJ02:
            gcjLat = lat
            gcjLon = lon
            break
        case CoordSystem.BD09:
            ;[gcjLat, gcjLon] = BD09ToGCJ02(lat, lon)
            break
        default:
            return [lat, lon]
    }

    switch (to) {
        case CoordSystem.WGS84:
            return GCJ02ToWGS84(gcjLat, gcjLon)
        case CoordSystem.GCJ02:
            return [gcjLat, gcjLon]
        case CoordSystem.BD09:
            return GCJ02ToBD09(gcjLat, gcjLon)
        default:
            return [gcjLat, gcjLon]
    }
}

// 单点转换
export function TransformCoordinatePoint(coord: Coordinate, from: CoordSystem, to: CoordSystem): Coordinate {
    const [lat, lon] = TransformCoordinate(coord.Latitude, coord.Longitude, from, to)
    return { Latitude: lat, Longitude: lon }
}

// 批量转换
export function TransformCoordinates(coords: Coordinate[], from: CoordSystem, to: CoordSystem): Coordinate[] {
    return coords.map(c => TransformCoordinatePoint(c, from, to))
}

// 是否在中国境内
function outOfChina(lat: number, lon: number): boolean {
    return lon < 72.004 || lon > 137.8347 || lat < 0.8293 || lat > 55.8271
}

// 纬度转换
function transformLat(x: number, y: number): number {
    let ret = -100.0 + 2.0 * x + 3.0 * y + 0.2 * y * y + 0.1 * x * y + 0.2 * Math.sqrt(Math.abs(x))
    ret += ((20.0 * Math.sin(6.0 * x * pi) + 20.0 * Math.sin(2.0 * x * pi)) * 2.0) / 3.0
    ret += ((20.0 * Math.sin(y * pi) + 40.0 * Math.sin((y / 3.0) * pi)) * 2.0) / 3.0
    ret += ((160.0 * Math.sin((y / 12.0) * pi) + 320 * Math.sin((y * pi) / 30.0)) * 2.0) / 3.0
    return ret
}

// 经度转换
function transformLon(x: number, y: number): number {
    let ret = 300.0 + x + 2.0 * y + 0.1 * x * x + 0.1 * x * y + 0.1 * Math.sqrt(Math.abs(x))
    ret += ((20.0 * Math.sin(6.0 * x * pi) + 20.0 * Math.sin(2.0 * x * pi)) * 2.0) / 3.0
    ret += ((20.0 * Math.sin(x * pi) + 40.0 * Math.sin((x / 3.0) * pi)) * 2.0) / 3.0
    ret += ((150.0 * Math.sin((x / 12.0) * pi) + 300.0 * Math.sin((x / 30.0) * pi)) * 2.0) / 3.0
    return ret
}
