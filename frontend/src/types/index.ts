// 设备信息
export interface Device {
    supportsRsd: any
    udid: string
    name: string
    productType: string
    productVersion: string
}

// 运行配置
export interface RunConfig {
    speed: number
    speedVariation: number
    routeVariation: number
    loopCount: number
    updateInterval: number
}

// 运行状态
export type RunStatus = 'idle' | 'running' | 'paused'

// 运行统计
export interface RunStats {
    status: RunStatus
    totalDistance: number
    currentSpeed: number
    elapsedTime: number
    currentLoop: number
    pointIndex: number
    totalPoints: number
    currentLat: number
    currentLon: number
}

// 坐标点
export interface Coordinate {
    latitude: number
    longitude: number
}

// 路线中心
export interface RouteCenter {
    lat: number
    lon: number
}

// 日志条目
export interface LogEntry {
    time: string
    level: string
    module: string
    message: string
}

// Toast 提示
export interface Toast {
    show: boolean
    message: string
    type: 'info' | 'success' | 'error' | 'warning'
}

// Tunnel 状态
export interface TunnelStatus {
    running: boolean
    tunnelCount?: number
}

// Tunnel 操作结果
export interface TunnelResult {
    success: boolean
    message: string
    alreadyRunning?: boolean
}

// 开发者镜像状态
export interface DeveloperImageStatus {
    mounted: boolean
}

// 路线统计
export interface RouteStats {
    distance: number
    points: number
}
