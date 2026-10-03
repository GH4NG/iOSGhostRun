/**
 * 路线相关工具函数
 */

export interface RoutePoint {
    lat: number
    lon: number
}

/**
 * 从 GPX 文件解析路线点，优先读取轨迹点，没有轨迹时读取路线点。
 */
export function parseGPXRoute(xml: string): RoutePoint[] {
    const document = new DOMParser().parseFromString(xml, 'application/xml')
    const elements = Array.from(document.getElementsByTagName('*'))
    const root = document.documentElement

    if (elements.some(element => element.localName === 'parsererror') || root.localName !== 'gpx') {
        throw new Error('文件不是有效的 GPX XML。')
    }

    const trackPoints = elements.filter(element => element.localName === 'trkpt')
    const routePoints = elements.filter(element => element.localName === 'rtept')
    const sourcePoints = trackPoints.length > 0 ? trackPoints : routePoints

    if (sourcePoints.length === 0) {
        throw new Error('GPX 文件中没有找到轨迹点或路线点。')
    }

    return sourcePoints.map((element, index) => {
        const latText = element.getAttribute('lat')?.trim()
        const lonText = element.getAttribute('lon')?.trim()
        const lat = latText ? Number(latText) : Number.NaN
        const lon = lonText ? Number(lonText) : Number.NaN

        if (!Number.isFinite(lat) || !Number.isFinite(lon) || lat < -90 || lat > 90 || lon < -180 || lon > 180) {
            throw new Error(`GPX 第 ${index + 1} 个位置点的经纬度无效。`)
        }

        return { lat, lon }
    })
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
