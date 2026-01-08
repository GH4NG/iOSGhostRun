<template>
  <div id="app">
    <div class="app-container">
      <!-- 侧边栏 -->
      <aside class="sidebar">
        <div class="sidebar-header">
          <h1>🏃 iOSGhostRun</h1>
        </div>

        <div class="sidebar-content">
          <!-- 设备面板 -->
          <DevicePanel
            :devices="device.devices.value"
            :selected-device="device.selectedDevice.value?.udid ?? null"
            :loading="device.loading.value"
            @refresh="handleRefreshDevices"
            @select="handleSelectDevice"
          />

          <!-- 运行状态 -->
          <RunStats
            v-if="device.selectedDevice.value"
            :stats="runner.stats"
            :loop-count="runner.config.loopCount"
          />

          <!-- 速度设置 -->
          <RunConfig
            v-model="runner.config"
            :disabled="runner.stats.status === 'running'"
          />

          <!-- 路线设置 -->
          <RouteConfig
            :route-points="runner.routePoints.value"
            :disabled="runner.stats.status === 'running'"
            @clear="handleClearRoute"
          />

          <!-- 控制按钮 -->
          <RunControls
            v-if="device.selectedDevice.value"
            :status="runner.stats.status"
            :can-start="runner.routePoints.value.length >= 2"
            @start="handleStartRun"
            @pause="handlePauseRun"
            @resume="handleResumeRun"
            @stop="handleStopRun"
            @reset="handleResetLocation"
          />
        </div>
      </aside>

      <!-- 主内容区 -->
      <main class="main-content">
        <MapView
          :draw-mode="mapCtrl.drawMode.value"
          :show-log="logger.showLogPanel.value"
          @center="handleCenterMap"
          @toggle-draw="handleToggleDrawMode"
          @toggle-log="logger.toggleLogPanel"
        />

        <!-- 日志面板 -->
        <LogPanel
          v-if="logger.showLogPanel.value"
          :logs="logger.logs.value"
          @clear="logger.clearLogs"
          @close="logger.toggleLogPanel"
        />

        <!-- 底部状态栏 -->
        <StatusBar
          :connected="!!device.selectedDevice.value"
          :current-lat="runner.stats.currentLat"
          :current-lon="runner.stats.currentLon"
          :route-distance="runner.routeDistance.value"
        />
      </main>
    </div>

    <!-- Toast 提示 -->
    <Toast
      :show="toast.show.value"
      :message="toast.message.value"
      :type="toast.type.value"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, watch } from 'vue'

// 组件
import DevicePanel from '@/components/Sidebar/DevicePanel.vue'
import RunStats from '@/components/Sidebar/RunStats.vue'
import RunConfig from '@/components/Sidebar/RunConfig.vue'
import RouteConfig from '@/components/Sidebar/RouteConfig.vue'
import RunControls from '@/components/Sidebar/RunControls.vue'
import MapView from '@/components/MapView.vue'
import LogPanel from '@/components/LogPanel.vue'
import StatusBar from '@/components/StatusBar.vue'
import Toast from '@/components/Toast.vue'

// Composables
import { useDevice, useRunner, useMap, useToast, useLogger } from '@/composables'

// 初始化 composables
const device = useDevice()
const runner = useRunner()
const mapCtrl = useMap()
const toast = useToast()
const logger = useLogger()

// 监听路线变化，更新地图
watch(
  () => runner.routePoints.value,
  (points) => {
    mapCtrl.updateRouteOnMap(points)
  },
  { deep: true }
)

// ===== 事件处理 =====

async function handleRefreshDevices() {
  try {
    await device.refreshDevices()
  } catch (error) {
    toast.showToast('刷新设备失败: ' + error, 'error')
  }
}

async function handleSelectDevice(udid: string) {
  try {
    toast.showToast('设备已连接，正在检查...', 'info')
    await device.selectDevice(udid)
    await handleMountDeveloperImage()
  } catch (error) {
    toast.showToast('选择设备失败: ' + error, 'error')
  }
}

async function handleMountDeveloperImage() {
  try {
    toast.showToast('正在检查开发者镜像...', 'info')
    const result = await device.checkAndMountDeveloperImage()
    toast.showToast(result.message, 'success')
  } catch (error) {
    const errorMsg = String(error)
    if (errorMsg.includes('already a developer image mounted')) {
      toast.showToast('开发者镜像已就绪', 'success')
    } else {
      toast.showToast('开发者镜像挂载失败: ' + errorMsg, 'error')
    }
  }
}

function handleClearRoute() {
  runner.clearRoute()
  mapCtrl.clearRouteLayer()
}

async function handleStartRun() {
  try {
    await runner.start()
    toast.showToast('开始跑步！', 'success')
  } catch (error) {
    toast.showToast('启动失败: ' + error, 'error')
  }
}

async function handlePauseRun() {
  try {
    await runner.pause()
  } catch (error) {
    toast.showToast('暂停失败: ' + error, 'error')
  }
}

async function handleResumeRun() {
  try {
    await runner.resume()
  } catch (error) {
    toast.showToast('恢复失败: ' + error, 'error')
  }
}

async function handleStopRun() {
  try {
    await runner.stop()
    mapCtrl.clearCurrentMarker()
    toast.showToast('跑步已停止', 'info')
  } catch (error) {
    toast.showToast('停止失败: ' + error, 'error')
  }
}

async function handleResetLocation() {
  try {
    await device.resetLocation()
    toast.showToast('位置已重置', 'success')
  } catch (error) {
    toast.showToast('重置失败: ' + error, 'error')
  }
}

function handleCenterMap() {
  const defaultCenter = { lat: 39.9042, lon: 116.4074 }
  mapCtrl.centerMap(runner.routePoints.value, defaultCenter)
}

function handleToggleDrawMode() {
  mapCtrl.toggleDrawMode()
  toast.showToast(
    mapCtrl.drawMode.value ? '绘制模式已开启，点击地图添加路线点' : '绘制模式已关闭',
    'info'
  )
}

function handleMapClick(lat: number, lon: number) {
  if (runner.stats.status === 'idle') {
    runner.addRoutePoint(lat, lon)
  }
}

// 设置事件监听
function setupEventListeners() {
  if (window.runtime) {
    window.runtime.EventsOn('run:update', (data: any) => {
      runner.updateStats(data)
      if (data.currentLat && data.currentLon) {
        mapCtrl.updateCurrentPosition(data.currentLat, data.currentLon)
      }
    })

    window.runtime.EventsOn('run:completed', () => {
      toast.showToast('跑步完成！', 'success')
      runner.setStatusIdle()
    })

    window.runtime.EventsOn('run:error', (error: string) => {
      toast.showToast(error, 'error')
    })

    window.runtime.EventsOn('log:entry', (entry: any) => {
      logger.addLog(entry)
    })
  }
}

// 生命周期
onMounted(async () => {
  // 加载保存的路线
  runner.loadSavedRoute()

  // 初始化地图
  const initialCenter = runner.routePoints.value.length > 0
    ? { lat: runner.routePoints.value[0].latitude, lon: runner.routePoints.value[0].longitude }
    : { lat: 39.9042, lon: 116.4074 }

  mapCtrl.initMap('map', initialCenter, handleMapClick)

  // 如果有保存的路线，显示在地图上
  if (runner.routePoints.value.length > 0) {
    setTimeout(() => {
      mapCtrl.updateRouteOnMap(runner.routePoints.value)
    }, 100)
  }

  // 设置事件监听
  setupEventListeners()

  // 自动刷新设备列表
  try {
    await device.refreshDevices()
    if (device.devices.value.length > 0) {
      toast.showToast(`检测到 ${device.devices.value.length} 台设备`, 'success')
    }
  } catch (error) {
    toast.showToast('设备检测失败: ' + error, 'error')
  }

  // 加载日志
  await logger.loadLogs()
})
</script>

<style>
@import '@/styles/base.css';

.app-container {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

/* 侧边栏 */
.sidebar {
  width: var(--sidebar-width);
  background: var(--bg-sidebar);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-header {
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
}

.sidebar-header h1 {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-sidebar);
  margin: 0;
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

/* 卡片通用样式 */
.card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-color);
  border-radius: var(--radius);
  padding: 16px;
}

/* 主内容区 */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: var(--bg-main);
}
</style>

