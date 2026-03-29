/**
 * 路线相关工具函数
 */

export interface RoutePoint {
    lat: number
    lon: number
}

/**
 * 计算路线总距离
 */
export function calculateRouteDistance(points: RoutePoint[]): number {
    if (points.length < 2) return 0

    let distance = 0
    for (let i = 1; i < points.length; i++) {
        distance += haversine(points[i - 1].lat, points[i - 1].lon, points[i].lat, points[i].lon)
    }
    return distance
}

/**
 * Haversine 公式计算两点距离
 */
function haversine(lat1: number, lon1: number, lat2: number, lon2: number): number {
    const R = 6371
    const dLat = ((lat2 - lat1) * Math.PI) / 180
    const dLon = ((lon2 - lon1) * Math.PI) / 180
    const a =
        Math.sin(dLat / 2) * Math.sin(dLat / 2) +
        Math.cos((lat1 * Math.PI) / 180) * Math.cos((lat2 * Math.PI) / 180) * Math.sin(dLon / 2) * Math.sin(dLon / 2)
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
    return R * c
}

/**
 * 格式化距离显示
 */
export function formatDistance(km: number): string {
    if (km < 1) {
        return `${Math.round(km * 1000)} m`
    }
    return `${km.toFixed(2)} km`
}

/**
 * 格式化时间显示
 */
export function formatTime(ms: number): string {
    const seconds = Math.floor(ms / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)

    if (hours > 0) {
        return `${hours}:${String(minutes % 60).padStart(2, '0')}:${String(seconds % 60).padStart(2, '0')}`
    }
    return `${minutes}:${String(seconds % 60).padStart(2, '0')}`
}
