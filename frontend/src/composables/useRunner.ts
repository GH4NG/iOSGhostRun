import { ref, reactive, watch } from 'vue'
import type { RunConfig, RunStats, Coordinate } from '@/types'

const go = window.go

// 单例状态
const config = reactive<RunConfig>({
  speed: 8.0,
  speedVariation: 1.0,
  routeVariation: 3.0,
  loopCount: 1,
  updateInterval: 1000,
})

const stats = reactive<RunStats>({
  status: 'idle',
  totalDistance: 0,
  currentSpeed: 0,
  elapsedTime: 0,
  currentLoop: 0,
  pointIndex: 0,
  totalPoints: 0,
  currentLat: 0,
  currentLon: 0,
})

const routePoints = ref<Coordinate[]>([])
const routeDistance = ref(0)

// 标记是否已初始化 watcher
let watcherInitialized = false

async function calculateRouteDistance(): Promise<void> {
  if (routePoints.value.length < 2) {
    routeDistance.value = 0
    return
  }

  try {
    if (go?.runner?.Service) {
      const result = await go.runner.Service.CalculateRouteStats(routePoints.value)
      routeDistance.value = result.distance || 0
    }
  } catch (error) {
    console.error('计算距离失败:', error)
  }
}

function saveRoute(): void {
  try {
    const data = {
      routePoints: routePoints.value,
      savedAt: new Date().toISOString(),
    }
    localStorage.setItem('iOSGhostRun_route', JSON.stringify(data))
  } catch (error) {
    console.error('保存路线失败:', error)
  }
}

export function useRunner() {
  // 确保 watcher 只注册一次
  if (!watcherInitialized) {
    watcherInitialized = true
    // 监听路线变化，自动计算距离并保存
    watch(routePoints, async () => {
      await calculateRouteDistance()
      saveRoute()
    }, { deep: true })
  }

  function loadSavedRoute(): void {
    try {
      const savedData = localStorage.getItem('iOSGhostRun_route')
      if (savedData) {
        const data = JSON.parse(savedData)
        if (data.routePoints && data.routePoints.length > 0) {
          routePoints.value = data.routePoints
        }
      }
    } catch (error) {
      console.error('加载保存的路线失败:', error)
    }
  }

  function addRoutePoint(lat: number, lon: number): void {
    routePoints.value.push({ latitude: lat, longitude: lon })
  }

  function clearRoute(): void {
    routePoints.value = []
    routeDistance.value = 0
    localStorage.removeItem('iOSGhostRun_route')
  }

  async function start(): Promise<void> {
    if (go?.runner?.Service) {
      await go.runner.Service.SetConfig(config)

      // 转换坐标系
      let routeToUse = routePoints.value
      if (go?.location?.Service) {
        routeToUse = await go.location.Service.ConvertToWGS84(routePoints.value, 'GCJ02')
      }

      await go.runner.Service.SetRoute(routeToUse)
      await go.runner.Service.Start()
      stats.status = 'running'
    }
  }

  async function pause(): Promise<void> {
    if (go?.runner?.Service) {
      await go.runner.Service.Pause()
      stats.status = 'paused'
    }
  }

  async function resume(): Promise<void> {
    if (go?.runner?.Service) {
      await go.runner.Service.Resume()
      stats.status = 'running'
    }
  }

  async function stop(): Promise<void> {
    if (go?.runner?.Service) {
      await go.runner.Service.Stop()
      stats.status = 'idle'
    }
  }

  function updateStats(data: RunStats): void {
    Object.assign(stats, data)
  }

  function setStatusIdle(): void {
    stats.status = 'idle'
  }

  return {
    config,
    stats,
    routePoints,
    routeDistance,
    loadSavedRoute,
    addRoutePoint,
    clearRoute,
    start,
    pause,
    resume,
    stop,
    updateStats,
    setStatusIdle,
  }
}
