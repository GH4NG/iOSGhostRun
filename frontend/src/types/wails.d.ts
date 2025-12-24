import type {
    Device,
    RunConfig,
    RunStats,
    Coordinate,
    LogEntry,
    TunnelStatus,
    TunnelResult,
    DeveloperImageStatus,
    RouteStats
} from './index'

// Wails Go 绑定类型声明
declare global {
    interface Window {
        go: {
            device: {
                Manager: {
                    RefreshDevices(): Promise<Device[]>
                    SelectDevice(udid: string): Promise<void>
                    IsIOS17OrAbove(): Promise<boolean>
                    CheckTunnelStatus(): Promise<TunnelStatus>
                    StartTunnel(): Promise<TunnelResult>
                    GetDeveloperImageStatus(): Promise<DeveloperImageStatus>
                    MountDeveloperImage(): Promise<void>
                    SetSimLocation(lat: number, lon: number): Promise<void>
                    ResetSimLocation(): Promise<void>
                }
            }
            runner: {
                Service: {
                    SetConfig(config: RunConfig): Promise<void>
                    SetRoute(points: Coordinate[]): Promise<void>
                    Start(): Promise<void>
                    Pause(): Promise<void>
                    Resume(): Promise<void>
                    Stop(): Promise<void>
                    CalculateRouteStats(points: Coordinate[]): Promise<RouteStats>
                }
            }
            location: {
                Service: {
                    ConvertToWGS84(points: Coordinate[], coordSystem: string): Promise<Coordinate[]>
                }
            }
            logger: {
                Service: {
                    GetLogs(): Promise<LogEntry[]>
                    ClearLogs(): Promise<void>
                }
            }
        }
        runtime: {
            EventsOn(eventName: string, callback: (data: any) => void): void
            EventsOff(eventName: string): void
        }
    }
}

export {}
