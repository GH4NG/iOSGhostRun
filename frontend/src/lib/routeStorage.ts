export interface RoutePoint {
    lat: number
    lon: number
}

export interface SavedRoute {
    name: string
    points: RoutePoint[]
    createdAt: number
}

const ROUTES_KEY = 'ios-ghost-run-routes'

/**
 * 获取所有保存的路线
 */
export function listRoutes(): SavedRoute[] {
    try {
        const data = localStorage.getItem(ROUTES_KEY)
        if (!data) return []
        const parsed = JSON.parse(data)
        if (Array.isArray(parsed)) return parsed
        if (parsed && typeof parsed === 'object' && 'name' in parsed && 'points' in parsed) {
            return [parsed as SavedRoute]
        }
        return []
    } catch {
        return []
    }
}

/**
 * 保存路线
 */
export function saveRoute(name: string, points: RoutePoint[]): void {
    const routes = listRoutes()
    const existing = routes.findIndex(r => r.name === name)

    const route: SavedRoute = {
        name,
        points,
        createdAt: Date.now()
    }

    if (existing >= 0) {
        routes[existing] = route
    } else {
        routes.push(route)
    }

    localStorage.setItem(ROUTES_KEY, JSON.stringify(routes))
}

/**
 * 加载路线
 */
export function loadRoute(name: string): RoutePoint[] | null {
    const routes = listRoutes()
    const route = routes.find(r => r.name === name)
    return route?.points ?? null
}

/**
 * 删除路线
 */
export function deleteRoute(name: string): void {
    const routes = listRoutes().filter(r => r.name !== name)
    localStorage.setItem(ROUTES_KEY, JSON.stringify(routes))
}

/**
 * 保存上次路线
 */
export function saveLastRoute(points: RoutePoint[]): void {
    const routesRaw = listRoutes()
    const routes = Array.isArray(routesRaw) ? routesRaw : []

    const route: SavedRoute = {
        name: 'last_route',
        points,
        createdAt: Date.now()
    }

    const existing = routes.findIndex(r => r.name === route.name)
    if (existing >= 0) {
        routes[existing] = route
    } else {
        routes.push(route)
    }

    localStorage.setItem(ROUTES_KEY, JSON.stringify(routes))
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
