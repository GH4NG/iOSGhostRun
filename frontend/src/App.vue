<template>
    <div id="app">
        <div class="app-container">
            <!-- 侧边栏 -->
            <aside class="sidebar">
                <div class="sidebar-header">
                    <h1>iOSGhostRun</h1>
                </div>

                <div class="sidebar-content">
                    <!-- 设备选择 -->
                    <div class="card">
                        <div class="card-title">
                            <span>设备连接</span>
                            <button class="btn btn-sm btn-secondary" @click="refreshDevices" :disabled="loading">
                                {{ loading ? '刷新中...' : '刷新' }}
                            </button>
                        </div>

                        <div v-if="devices.length === 0" class="empty-state">
                            <p>未检测到设备</p>
                            <p style="font-size: 12px; margin-top: 8px">
                                请连接iOS设备并信任此电脑
                            </p>
                        </div>

                        <div v-else class="device-list">
                            <div v-for="device in devices" :key="device.udid" class="device-item" :class="{
                                selected: selectedDevice === device.udid,
                            }" @click="selectDevice(device.udid)">
                                <div class="device-info">
                                    <h3>{{ device.name }}</h3>
                                    <p>
                                        {{ device.productType }} · iOS
                                        {{ device.productVersion }}
                                    </p>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- Tunnel 服务状态 -->
                    <div class="card" v-if="selectedDevice && isIOS17Device">
                        <div class="card-title">
                            <span>Tunnel 服务</span>
                            <span :class="[
                                'status-dot',
                                tunnelRunning ? 'running' : 'idle',
                            ]"></span>
                        </div>
                        <div class="tunnel-info">
                            <p style="
                                    margin-bottom: 12px;
                                    font-size: 12px;
                                    color: #666;
                                ">
                                {{
                                    tunnelRunning
                                        ? '✓ Tunnel 服务运行中'
                                        : '✗ Tunnel 服务未启动'
                                }}
                            </p>
                            <button v-if="!tunnelRunning" class="btn btn-primary" @click="startTunnel"
                                :disabled="startingTunnel">
                                {{
                                    startingTunnel ? '启动中...' : '启动 Tunnel'
                                }}
                            </button>
                            <button v-else class="btn btn-secondary" disabled>
                                服务已启动
                            </button>
                        </div>
                    </div>

                    <!-- 运行状态 -->
                    <div class="card" v-if="selectedDevice">
                        <div class="card-title">
                            <span>运行状态</span>
                            <span :class="['status-dot', stats.status]"></span>
                        </div>

                        <div class="stats-grid">
                            <div class="stat-item">
                                <div class="stat-value">
                                    {{ formatDistance(stats.totalDistance) }}
                                </div>
                                <div class="stat-label">总距离 (m)</div>
                            </div>
                            <div class="stat-item">
                                <div class="stat-value">
                                    {{ formatSpeed(stats.currentSpeed) }}
                                </div>
                                <div class="stat-label">当前配速</div>
                            </div>
                            <div class="stat-item">
                                <div class="stat-value">
                                    {{ formatTime(stats.elapsedTime) }}
                                </div>
                                <div class="stat-label">已用时间</div>
                            </div>
                            <div class="stat-item">
                                <div class="stat-value">
                                    {{ stats.currentLoop }}/{{
                                        config.loopCount || '∞'
                                    }}
                                </div>
                                <div class="stat-label">当前圈数</div>
                            </div>
                        </div>
                    </div>

                    <!-- 速度设置 -->
                    <div class="card" v-if="selectedDevice">
                        <div class="card-title">跑步设置</div>

                        <div class="slider-group">
                            <div class="slider-header">
                                <span class="slider-label">跑步速度</span>
                                <span class="slider-value">{{ config.speed.toFixed(1) }} km/h</span>
                            </div>
                            <input type="range" v-model.number="config.speed" min="3" max="20" step="0.5"
                                :disabled="stats.status === 'running'" />
                        </div>

                        <div class="slider-group">
                            <div class="slider-header">
                                <span class="slider-label">速度随机波动</span>
                                <span class="slider-value">±{{
                                    config.speedVariation.toFixed(1)
                                }}
                                    km/h</span>
                            </div>
                            <input type="range" v-model.number="config.speedVariation" min="0" max="3" step="0.1"
                                :disabled="stats.status === 'running'" />
                        </div>

                        <div class="slider-group">
                            <div class="slider-header">
                                <span class="slider-label">路线随机偏移</span>
                                <span class="slider-value">±{{
                                    config.routeVariation.toFixed(0)
                                }}
                                    米</span>
                            </div>
                            <input type="range" v-model.number="config.routeVariation" min="0" max="10" step="1"
                                :disabled="stats.status === 'running'" />
                        </div>

                        <div class="form-group">
                            <label>循环次数 (0=无限)</label>
                            <input type="number" class="form-control" v-model.number="config.loopCount" min="0"
                                max="100" :disabled="stats.status === 'running'" />
                        </div>
                    </div>

                    <!-- 路线设置 -->
                    <div class="card" v-if="selectedDevice">
                        <div class="card-title">
                            <span>路线设置</span>
                            <span style="
                                    font-size: 12px;
                                    color: var(--text-secondary);
                                ">
                                {{ routePoints.length }} 个点
                            </span>
                        </div>

                        <button class="btn btn-secondary btn-block" @click="clearRoute" :disabled="stats.status === 'running' ||
                            routePoints.length === 0
                            ">
                            清除路线
                        </button>

                        <div class="route-points" v-if="routePoints.length > 0">
                            <div class="route-point" v-for="(point, index) in routePoints.slice(
                                0,
                                5
                            )" :key="index">
                                <span class="route-point-index">{{
                                    index + 1
                                    }}</span>
                                <span>{{ point.latitude.toFixed(5) }},
                                    {{ point.longitude.toFixed(5) }}</span>
                            </div>
                            <div v-if="routePoints.length > 5" class="route-point">
                                <span>... 还有
                                    {{ routePoints.length - 5 }} 个点</span>
                            </div>
                        </div>
                    </div>

                    <!-- 控制按钮 -->
                    <div class="card" v-if="selectedDevice">
                        <div class="card-title">运行控制</div>

                        <div class="btn-group" v-if="stats.status === 'idle'">
                            <button class="btn btn-success btn-block" @click="startRun"
                                :disabled="routePoints.length < 2">
                                开始跑步
                            </button>
                        </div>

                        <div class="btn-group" v-else-if="stats.status === 'running'">
                            <button class="btn btn-warning" @click="pauseRun">
                                暂停
                            </button>
                            <button class="btn btn-danger" @click="stopRun">
                                停止
                            </button>
                        </div>

                        <div class="btn-group" v-else-if="stats.status === 'paused'">
                            <button class="btn btn-success" @click="resumeRun">
                                继续
                            </button>
                            <button class="btn btn-danger" @click="stopRun">
                                停止
                            </button>
                        </div>

                        <button class="btn btn-secondary btn-block" @click="resetLocation" style="margin-top: 10px"
                            :disabled="stats.status === 'running'">
                            重置真实位置
                        </button>
                    </div>
                </div>
            </aside>

            <!-- 主内容区 -->
            <main class="main-content">
                <div class="map-container">
                    <div id="map"></div>

                    <div class="map-controls">
                        <button class="map-btn" @click="centerMap" title="定位到路线中心">
                            C
                        </button>
                        <button class="map-btn" @click="toggleDrawMode" :class="{ active: drawMode }" title="绘制模式">
                            D
                        </button>
                        <button class="map-btn" @click="toggleLogPanel" :class="{ active: showLogPanel }" title="日志面板">
                            L
                        </button>
                    </div>
                </div>

                <!-- 日志面板 -->
                <div class="log-panel" v-if="showLogPanel">
                    <div class="log-header">
                        <span>运行日志</span>
                        <div class="log-actions">
                            <button class="btn btn-sm btn-secondary" @click="clearLogs">
                                清除
                            </button>
                            <button class="btn btn-sm btn-secondary" @click="toggleLogPanel">
                                关闭
                            </button>
                        </div>
                    </div>
                    <div class="log-content" ref="logContentRef">
                        <div v-for="(log, index) in logs" :key="index" :class="['log-entry', 'log-' + log.level]">
                            <span class="log-time">{{ log.time }}</span>
                            <span class="log-level">{{
                                log.level.toUpperCase()
                                }}</span>
                            <span class="log-module">[{{ log.module }}]</span>
                            <span class="log-message">{{ log.message }}</span>
                        </div>
                        <div v-if="logs.length === 0" class="log-empty">
                            暂无日志
                        </div>
                    </div>
                </div>

                <!-- 底部状态栏 -->
                <div class="status-bar">
                    <div class="status-indicator">
                        <span :class="[
                            'status-dot',
                            selectedDevice ? 'connected' : '',
                        ]"></span>
                        <span>{{
                            selectedDevice ? '设备已连接' : '未连接设备'
                            }}</span>
                    </div>
                    <div>
                        <span v-if="stats.currentLat && stats.currentLon">
                            当前位置: {{ stats.currentLat.toFixed(5) }},
                            {{ stats.currentLon.toFixed(5) }}
                        </span>
                    </div>
                    <div>
                        <span>路线距离:
                            {{ formatDistance(routeDistance) }}m</span>
                    </div>
                </div>
            </main>
        </div>

        <!-- Toast 提示 -->
        <div v-if="toast.show" :class="['toast', toast.type]">
            {{ toast.message }}
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, nextTick } from 'vue'
import L from 'leaflet'
import type { Map, LayerGroup, CircleMarker } from 'leaflet'
import type {
    Device,
    RunConfig,
    RunStats,
    Coordinate,
    RouteCenter,
    LogEntry,
    Toast,
} from './types'

// Wails 运行时绑定
const go = window.go

// 响应式状态
const devices = ref<Device[]>([])
const selectedDevice = ref<string | null>(null)
const loading = ref(false)
const isIOS17Device = ref(false)
const tunnelRunning = ref(false)
const startingTunnel = ref(false)

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
const routeCenter = reactive<RouteCenter>({ lat: 39.9042, lon: 116.4074 })
const routeDistance = ref(0)

// 地图相关
let map: Map | null = null
let routeLayer: LayerGroup | null = null
let currentMarker: CircleMarker | null = null
const drawMode = ref(false)

// 平滑动画相关
let targetPosition: { lat: number; lon: number } | null = null
let currentAnimatedPosition: { lat: number; lon: number } | null = null
let animationFrameId: number | null = null

// 日志相关
const logs = ref<LogEntry[]>([])
const showLogPanel = ref(false)
const logContentRef = ref<HTMLDivElement | null>(null)

// UI 相关
const toast = reactive<Toast>({
    show: false,
    message: '',
    type: 'info',
})

// 初始化地图
function initMap() {
    const initLat =
        routePoints.value.length > 0
            ? routePoints.value[0].latitude
            : routeCenter.lat
    const initLon =
        routePoints.value.length > 0
            ? routePoints.value[0].longitude
            : routeCenter.lon

    map = L.map('map').setView([initLat, initLon], 16)

    // 使用高德地图瓦片 (GCJ-02 坐标系)
    L.tileLayer(
        'https://webrd0{s}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}',
        {
            subdomains: ['1', '2', '3', '4'],
            attribution: '© 高德地图',
        }
    ).addTo(map)

    // 创建路线图层
    routeLayer = L.layerGroup().addTo(map)

    // 点击添加路线点
    map.on('click', (e) => {
        if (drawMode.value && stats.status === 'idle') {
            addRoutePoint(e.latlng.lat, e.latlng.lng)
        }
    })

    // 如果有保存的路线，显示在地图上
    if (routePoints.value.length > 0) {
        nextTick(() => {
            updateRouteOnMap()
            calculateRouteDistance()
        })
    }
}

// 设置事件监听
function setupEventListeners() {
    if (window.runtime) {
        window.runtime.EventsOn('run:update', (data: RunStats) => {
            Object.assign(stats, data)
            updateCurrentPosition()
        })

        window.runtime.EventsOn('run:completed', () => {
            showToast('跑步完成！', 'success')
            stats.status = 'idle'
        })

        window.runtime.EventsOn('run:error', (error: string) => {
            showToast(error, 'error')
        })

        window.runtime.EventsOn('log:entry', (entry: LogEntry) => {
            addLog(entry)
        })
    }
}

// 加载保存的路线
function loadSavedRoute() {
    try {
        const savedData = localStorage.getItem('iOSGhostRun_route')
        if (savedData) {
            const data = JSON.parse(savedData)
            if (data.routePoints && data.routePoints.length > 0) {
                routePoints.value = data.routePoints
                routeCenter.lat = data.routePoints[0].latitude
                routeCenter.lon = data.routePoints[0].longitude
                console.log(
                    '已加载保存的路线，共',
                    routePoints.value.length,
                    '个点'
                )
            }
        }
    } catch (error) {
        console.error('加载保存的路线失败:', error)
    }
}

// 保存路线到本地存储
function saveRoute() {
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

// 加载历史日志
async function loadLogs() {
    try {
        if (go?.logger?.Service) {
            const result = await go.logger.Service.GetLogs()
            logs.value = result || []
        }
    } catch (error) {
        console.error('加载日志失败:', error)
    }
}

// 添加日志
function addLog(entry: LogEntry) {
    logs.value.push(entry)
    if (logs.value.length > 500) {
        logs.value.shift()
    }
    nextTick(() => {
        if (logContentRef.value) {
            logContentRef.value.scrollTop = logContentRef.value.scrollHeight
        }
    })
}

// 清除日志
async function clearLogs() {
    try {
        if (go?.logger?.Service) {
            await go.logger.Service.ClearLogs()
            logs.value = []
        }
    } catch (error) {
        console.error('清除日志失败:', error)
    }
}

// 切换日志面板
function toggleLogPanel() {
    showLogPanel.value = !showLogPanel.value
}

// 刷新设备列表
async function refreshDevices() {
    loading.value = true
    try {
        if (go?.device?.Manager) {
            const result = await go.device.Manager.RefreshDevices()
            devices.value = result || []
        }
    } catch (error) {
        showToast('刷新设备失败: ' + error, 'error')
    } finally {
        loading.value = false
    }
}

// 选择设备
async function selectDevice(udid: string) {
    try {
        if (go?.device?.Manager) {
            await go.device.Manager.SelectDevice(udid)
            selectedDevice.value = udid
            showToast('设备已连接，正在检查...', 'info')

            // 检查是否是 iOS 17+ 设备
            const isIOS17 = await go.device.Manager.IsIOS17OrAbove()
            isIOS17Device.value = isIOS17

            if (isIOS17) {
                const tunnelStatus = await go.device.Manager.CheckTunnelStatus()
                tunnelRunning.value = tunnelStatus.running

                if (!tunnelStatus.running) {
                    showToast(
                        'iOS 17+ 设备检测到！Tunnel 服务未启动，请点击下方"启动 Tunnel"按钮启动服务',
                        'warning'
                    )
                    return
                }
            }

            await checkAndMountDeveloperImage()
        }
    } catch (error) {
        showToast('选择设备失败: ' + error, 'error')
    }
}

// 启动 Tunnel 服务
async function startTunnel() {
    try {
        startingTunnel.value = true
        showToast('正在启动 Tunnel 服务...', 'info')

        if (go?.device?.Manager) {
            const result = await go.device.Manager.StartTunnel()

            if (result.success) {
                tunnelRunning.value = true
                showToast(result.message, 'success')

                // 启动成功后，继续挂载开发者镜像
                await checkAndMountDeveloperImage()
            } else {
                showToast(`启动 Tunnel 失败: ${result.message}`, 'error')
            }
        }
    } catch (error) {
        showToast('启动 Tunnel 异常: ' + error, 'error')
    } finally {
        startingTunnel.value = false
    }
}

// 检查并挂载开发者镜像
async function checkAndMountDeveloperImage() {
    try {
        if (go?.device?.Manager) {
            const status = await go.device.Manager.GetDeveloperImageStatus()

            if (status.mounted) {
                showToast('开发者镜像已就绪', 'success')
                return
            }

            showToast('正在下载并挂载开发者镜像，请稍候...', 'info')
            await go.device.Manager.MountDeveloperImage()
            showToast('开发者镜像挂载成功！', 'success')
        }
    } catch (error) {
        const errorMsg = String(error)
        if (errorMsg.includes('already a developer image mounted')) {
            showToast('开发者镜像已就绪', 'success')
        } else {
            showToast(
                '开发者镜像挂载失败: ' +
                errorMsg +
                '。\n请确保设备已解锁并信任此电脑。',
                'error'
            )
        }
    }
}

// 添加路线点
function addRoutePoint(lat: number, lon: number) {
    routePoints.value.push({ latitude: lat, longitude: lon })
    updateRouteOnMap()
    calculateRouteDistance()
    saveRoute()
}

// 清除路线
function clearRoute() {
    routePoints.value = []
    routeDistance.value = 0
    routeLayer?.clearLayers()
    localStorage.removeItem('iOSGhostRun_route')
}

// 更新地图上的路线
function updateRouteOnMap() {
    if (!routeLayer || !map) return
    routeLayer.clearLayers()

    if (routePoints.value.length === 0) return

    const latlngs = routePoints.value.map(
        (p) => [p.latitude, p.longitude] as [number, number]
    )

    const polyline = L.polyline(latlngs, {
        color: '#4a9eff',
        weight: 4,
        opacity: 0.8,
    })
    routeLayer.addLayer(polyline)

    routePoints.value.forEach((point, index) => {
        let color: string
        let radius: number

        if (index === 0) {
            color = '#52c41a'
            radius = 10
        } else if (index === routePoints.value.length - 1) {
            color = '#ff4d4f'
            radius = 10
        } else {
            color = '#4a9eff'
            radius = 6
        }

        const marker = L.circleMarker([point.latitude, point.longitude], {
            radius,
            fillColor: color,
            color: '#fff',
            weight: 2,
            fillOpacity: 1,
        })
        routeLayer!.addLayer(marker)
    })

    map.fitBounds(polyline.getBounds(), { padding: [50, 50] })
}

// 更新当前位置标记
function updateCurrentPosition() {
    if (!stats.currentLat || !stats.currentLon || !map) return

    const gcj02 = wgs84ToGcj02(stats.currentLat, stats.currentLon)
    targetPosition = { lat: gcj02.lat, lon: gcj02.lon }

    // 如果是第一次设置位置，直接跳转
    if (!currentAnimatedPosition) {
        currentAnimatedPosition = { ...targetPosition }
        if (!currentMarker) {
            currentMarker = L.circleMarker([gcj02.lat, gcj02.lon], {
                radius: 12,
                fillColor: '#ff9500',
                color: '#fff',
                weight: 3,
                fillOpacity: 1,
            }).addTo(map)
        } else {
            currentMarker.setLatLng([gcj02.lat, gcj02.lon])
        }
        return
    }

    // 启动平滑动画
    if (!animationFrameId) {
        animateMarker()
    }
}

// 平滑动画函数
function animateMarker() {
    if (!targetPosition || !currentAnimatedPosition || !currentMarker) {
        animationFrameId = null
        return
    }

    // 线性插值系数
    const lerpFactor = 0.15

    // 计算新位置
    const newLat =
        currentAnimatedPosition.lat +
        (targetPosition.lat - currentAnimatedPosition.lat) * lerpFactor
    const newLon =
        currentAnimatedPosition.lon +
        (targetPosition.lon - currentAnimatedPosition.lon) * lerpFactor

    // 检查是否接近目标
    const distance =
        Math.abs(targetPosition.lat - newLat) +
        Math.abs(targetPosition.lon - newLon)

    if (distance < 0.0000001) {
        // 已到达目标，停止动画
        currentAnimatedPosition = { ...targetPosition }
        currentMarker.setLatLng([targetPosition.lat, targetPosition.lon])
        animationFrameId = null
        return
    }

    // 更新位置
    currentAnimatedPosition = { lat: newLat, lon: newLon }
    currentMarker.setLatLng([newLat, newLon])

    // 继续动画
    animationFrameId = requestAnimationFrame(animateMarker)
}

// 计算路线距离
async function calculateRouteDistance() {
    if (routePoints.value.length < 2) {
        routeDistance.value = 0
        return
    }

    try {
        if (go?.runner?.Service) {
            const result = await go.runner.Service.CalculateRouteStats(
                routePoints.value
            )
            routeDistance.value = result.distance || 0
        }
    } catch (error) {
        console.error('计算距离失败:', error)
    }
}

// 开始跑步
async function startRun() {
    try {
        if (go?.runner?.Service) {
            await go.runner.Service.SetConfig(config)

            let routeToUse = routePoints.value
            if (go?.location?.Service) {
                routeToUse = await go.location.Service.ConvertToWGS84(
                    routePoints.value,
                    'GCJ02'
                )
            }

            await go.runner.Service.SetRoute(routeToUse)
            await go.runner.Service.Start()
            stats.status = 'running'
            showToast('开始跑步！', 'success')
        }
    } catch (error) {
        showToast('启动失败: ' + error, 'error')
    }
}

// 暂停跑步
async function pauseRun() {
    try {
        if (go?.runner?.Service) {
            await go.runner.Service.Pause()
            stats.status = 'paused'
        }
    } catch (error) {
        showToast('暂停失败: ' + error, 'error')
    }
}

// 恢复跑步
async function resumeRun() {
    try {
        if (go?.runner?.Service) {
            await go.runner.Service.Resume()
            stats.status = 'running'
        }
    } catch (error) {
        showToast('恢复失败: ' + error, 'error')
    }
}

// 停止跑步
async function stopRun() {
    try {
        if (go?.runner?.Service) {
            await go.runner.Service.Stop()
            stats.status = 'idle'

            // 停止动画
            if (animationFrameId) {
                cancelAnimationFrame(animationFrameId)
                animationFrameId = null
            }
            targetPosition = null
            currentAnimatedPosition = null

            if (currentMarker && map) {
                map.removeLayer(currentMarker)
                currentMarker = null
            }
            showToast('跑步已停止', 'info')
        }
    } catch (error) {
        showToast('停止失败: ' + error, 'error')
    }
}

// 重置位置
async function resetLocation() {
    try {
        if (go?.device?.Manager) {
            await go.device.Manager.ResetSimLocation()
            showToast('位置已重置', 'success')
        }
    } catch (error) {
        showToast('重置失败: ' + error, 'error')
    }
}

// 地图定位到路线中心
function centerMap() {
    if (!map) return

    if (routePoints.value.length > 0) {
        const latlngs = routePoints.value.map(
            (p) => [p.latitude, p.longitude] as [number, number]
        )
        const bounds = L.latLngBounds(latlngs)
        map.fitBounds(bounds, { padding: [50, 50] })
    } else {
        map.setView([routeCenter.lat, routeCenter.lon], 15)
    }
}

// 切换绘制模式
function toggleDrawMode() {
    drawMode.value = !drawMode.value
    showToast(
        drawMode.value
            ? '绘制模式已开启，点击地图添加路线点'
            : '绘制模式已关闭',
        'info'
    )
}

// 格式化距离
function formatDistance(meters: number): number {
    return Math.round(meters)
}

// 格式化速度为配速
function formatSpeed(kmh: number): string {
    if (kmh <= 0) return '--\'--"'
    const minPerKm = 60 / kmh
    const min = Math.floor(minPerKm)
    const sec = Math.round((minPerKm - min) * 60)
    return `${min}'${sec.toString().padStart(2, '0')}"`
}

// 格式化时间
function formatTime(seconds: number): string {
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = seconds % 60
    if (h > 0) {
        return `${h}:${m.toString().padStart(2, '0')}:${s
            .toString()
            .padStart(2, '0')}`
    }
    return `${m}:${s.toString().padStart(2, '0')}`
}

// WGS84 转 GCJ02
function wgs84ToGcj02(lat: number, lon: number): { lat: number; lon: number } {
    const a = 6378245.0
    const ee = 0.00669342162296594323
    const PI = Math.PI

    const outOfChina = (lat: number, lon: number): boolean => {
        return lon < 72.004 || lon > 137.8347 || lat < 0.8293 || lat > 55.8271
    }

    const transformLat = (x: number, y: number): number => {
        let ret =
            -100.0 +
            2.0 * x +
            3.0 * y +
            0.2 * y * y +
            0.1 * x * y +
            0.2 * Math.sqrt(Math.abs(x))
        ret +=
            ((20.0 * Math.sin(6.0 * x * PI) + 20.0 * Math.sin(2.0 * x * PI)) *
                2.0) /
            3.0
        ret +=
            ((20.0 * Math.sin(y * PI) + 40.0 * Math.sin((y / 3.0) * PI)) *
                2.0) /
            3.0
        ret +=
            ((160.0 * Math.sin((y / 12.0) * PI) +
                320 * Math.sin((y * PI) / 30.0)) *
                2.0) /
            3.0
        return ret
    }

    const transformLon = (x: number, y: number): number => {
        let ret =
            300.0 +
            x +
            2.0 * y +
            0.1 * x * x +
            0.1 * x * y +
            0.1 * Math.sqrt(Math.abs(x))
        ret +=
            ((20.0 * Math.sin(6.0 * x * PI) + 20.0 * Math.sin(2.0 * x * PI)) *
                2.0) /
            3.0
        ret +=
            ((20.0 * Math.sin(x * PI) + 40.0 * Math.sin((x / 3.0) * PI)) *
                2.0) /
            3.0
        ret +=
            ((150.0 * Math.sin((x / 12.0) * PI) +
                300.0 * Math.sin((x / 30.0) * PI)) *
                2.0) /
            3.0
        return ret
    }

    if (outOfChina(lat, lon)) {
        return { lat, lon }
    }

    let dLat = transformLat(lon - 105.0, lat - 35.0)
    let dLon = transformLon(lon - 105.0, lat - 35.0)
    const radLat = (lat / 180.0) * PI
    let magic = Math.sin(radLat)
    magic = 1 - ee * magic * magic
    const sqrtMagic = Math.sqrt(magic)
    dLat = (dLat * 180.0) / (((a * (1 - ee)) / (magic * sqrtMagic)) * PI)
    dLon = (dLon * 180.0) / ((a / sqrtMagic) * Math.cos(radLat) * PI)

    return {
        lat: lat + dLat,
        lon: lon + dLon,
    }
}

// 显示提示
function showToast(message: string, type: Toast['type'] = 'info') {
    toast.show = true
    toast.message = message
    toast.type = type
    setTimeout(() => {
        toast.show = false
    }, 3000)
}

// 生命周期
onMounted(() => {
    loadSavedRoute()
    initMap()
    refreshDevices()
    setupEventListeners()
    loadLogs()
})
</script>
