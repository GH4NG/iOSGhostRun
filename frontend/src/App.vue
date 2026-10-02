<template>
  <TooltipProvider>
    <div
      class="h-dvh w-full flex flex-col bg-background text-foreground overflow-hidden font-sans selection:bg-primary/20 selection:text-primary">
      <!-- 通知系统 -->
      <Notification />

      <!-- 自定义标题栏 -->
      <header class="h-11 shrink-0 border-b border-border bg-card/70 backdrop-blur-md relative flex items-center"
        :class="isMacOS ? 'px-3' : 'pl-4 pr-1 justify-between'" style="--wails-draggable: drag">
        <template v-if="isMacOS">
          <div class="flex items-center gap-2" style="--wails-draggable: no-drag">
            <TitlebarButton class="group w-3 h-3 rounded-full bg-[#ff5f57] hover:brightness-95" aria-label="关闭"
              @click="requestClose">
              <span
                class="text-[8px] leading-none text-black/70 opacity-0 group-hover:opacity-100 transition-opacity">×</span>
            </TitlebarButton>
            <TitlebarButton class="group w-3 h-3 rounded-full bg-[#febc2e] hover:brightness-95" aria-label="最小化"
              @click="onMinimise">
              <span
                class="text-[8px] leading-none text-black/70 opacity-0 group-hover:opacity-100 transition-opacity">-</span>
            </TitlebarButton>
            <TitlebarButton class="group w-3 h-3 rounded-full bg-[#28c840] hover:brightness-95"
              :aria-label="isMaximized ? '还原' : '最大化'" @click="onToggleMaximise()">
              <span class="text-[8px] leading-none text-black/70 opacity-0 group-hover:opacity-100 transition-opacity">
                {{ isMaximized ? '−' : '+' }}
              </span>
            </TitlebarButton>
            <TitlebarButton
              class="relative ml-2 h-7 w-7 rounded-md text-muted-foreground hover:text-foreground hover:bg-secondary/60 flex items-center justify-center"
              :aria-label="updateInfo?.available ? `安装 v${updateInfo.latestVersion}` : '检查更新'"
              :disabled="isCheckingUpdate" @click="openUpdater">
              <UpdateIcon class="w-3.5 h-3.5" :class="{ 'animate-spin': isCheckingUpdate }" />
              <span v-if="updateInfo?.available"
                class="absolute right-0.5 top-0.5 h-1.5 w-1.5 rounded-full bg-primary"></span>
            </TitlebarButton>
          </div>

          <div class="absolute inset-0 flex items-center justify-center pointer-events-none">
            <span class="text-xs font-bold tracking-[0.12em] uppercase text-muted-foreground">iOSGhostRun</span>
          </div>
        </template>

        <template v-else>
          <div class="flex items-center gap-2 min-w-0">
            <div class="w-2 h-2 rounded-full bg-primary/80"></div>
            <span
              class="text-xs font-bold tracking-[0.12em] uppercase text-muted-foreground truncate">iOSGhostRun</span>
          </div>

          <div class="flex items-center gap-1" style="--wails-draggable: no-drag">
            <TitlebarButton
              class="relative w-9 h-7 rounded-md text-muted-foreground hover:text-foreground hover:bg-secondary/60 flex items-center justify-center"
              :aria-label="updateInfo?.available ? `安装 v${updateInfo.latestVersion}` : '检查更新'"
              :disabled="isCheckingUpdate" @click="openUpdater">
              <UpdateIcon class="w-4 h-4" :class="{ 'animate-spin': isCheckingUpdate }" />
              <span v-if="updateInfo?.available"
                class="absolute right-1.5 top-1 h-1.5 w-1.5 rounded-full bg-primary"></span>
            </TitlebarButton>
            <TitlebarButton
              class="w-9 h-7 rounded-md text-muted-foreground hover:text-foreground hover:bg-secondary/60 flex items-center justify-center"
              aria-label="最小化" @click="onMinimise">
              <MinusIcon class="w-4 h-4" />
            </TitlebarButton>
            <TitlebarButton
              class="w-9 h-7 rounded-md text-muted-foreground hover:text-foreground hover:bg-secondary/60 flex items-center justify-center"
              :aria-label="isMaximized ? '还原' : '最大化'" @click="onToggleMaximise()">
              <SquareIcon v-if="!isMaximized" class="w-4 h-4" />
              <CopyIcon v-else class="w-4 h-4" />
            </TitlebarButton>
            <TitlebarButton
              class="w-9 h-7 rounded-md text-muted-foreground hover:text-white hover:bg-destructive flex items-center justify-center"
              aria-label="关闭" @click="requestClose">
              <Cross1Icon class="w-4 h-4" />
            </TitlebarButton>
          </div>
        </template>
      </header>

      <div class="app-workspace relative flex-1 min-h-0 flex overflow-hidden" :class="{ 'is-compact': isCompactLayout }">
        <button v-if="isCompactLayout && !isSidebarCollapsed" type="button" aria-label="关闭控制面板"
          class="absolute inset-0 z-30 bg-black/30 backdrop-blur-[1px]" @click="isSidebarCollapsed = true"></button>
        <!-- 左侧面板 -->
        <aside
          class="app-sidebar min-h-0 shrink-0 border-r border-border/70 bg-card/95 backdrop-blur-md overflow-hidden shadow-2xl z-40 transition-[width] duration-300"
          :class="isSidebarCollapsed ? 'w-14' : 'w-80'">
          <div v-if="isSidebarCollapsed" class="h-full flex flex-col items-center py-4 px-2 gap-3">
            <TitlebarButton
              class="h-10 w-10 rounded-xl border border-border/60 bg-background/70 text-muted-foreground hover:text-foreground hover:bg-secondary/40 flex items-center justify-center"
              aria-label="展开左侧栏" @click="isSidebarCollapsed = false">
              <ChevronRightIcon class="w-4 h-4" />
            </TitlebarButton>
            <div class="w-8 h-px bg-border/60"></div>
            <div
              class="writing-mode-vertical text-sm font-medium text-muted-foreground select-none">
              控制面板
            </div>
          </div>

          <div v-show="!isSidebarCollapsed" class="h-full flex flex-col min-h-0">
            <div class="shrink-0 flex items-center justify-between px-4 pt-4 pb-2">
              <div class="text-sm font-medium text-muted-foreground">
                控制面板
              </div>
              <TitlebarButton
                class="h-8 w-8 rounded-lg border border-border/60 bg-background/70 text-muted-foreground hover:text-foreground hover:bg-secondary/40 flex items-center justify-center"
                aria-label="收起左侧栏" @click="isSidebarCollapsed = true">
                <ChevronLeftIcon class="w-4 h-4" />
              </TitlebarButton>
            </div>

            <div class="flex-1 min-h-0">
              <ScrollArea class="h-full">
                <div class="flex flex-col gap-4 px-3 pt-2 pb-5">
                  <DevicePanel v-model="selectedUdid" />
                  <RunningControl :udid="selectedUdid" :route-points="routePoints" @position-update="onPositionUpdate"
                    @state-change="onRunStateChange" />
                </div>
              </ScrollArea>
            </div>
          </div>
        </aside>

        <!-- 右侧区域: 地图 + 日志 -->
        <main class="app-main flex-1 flex flex-col min-w-0 min-h-0 relative bg-background">
          <!-- 地图区域 -->
          <div class="flex-1 relative min-h-0">
            <MapEditor ref="mapEditor" v-model="routePoints" :current-position="currentPosition"
              :disabled="isRunning" />
          </div>

          <!-- 日志区域 -->
          <div
            class="app-log-panel min-h-0 shrink-0 bg-card/85 backdrop-blur-xl border-t border-border z-30 transition-[height] duration-300 flex flex-col shadow-[0_-10px_30px_rgba(0,0,0,0.1)]"
            :class="{ 'is-open': !isLogCollapsed }">
            <button type="button" :aria-expanded="!isLogCollapsed" aria-controls="log-panel-content"
              class="h-9 shrink-0 flex items-center px-4 gap-3 cursor-pointer select-none bg-secondary/10 hover:bg-secondary/20 text-muted-foreground hover:text-foreground transition-all group border-b border-transparent"
              :class="{ 'border-border/30': !isLogCollapsed }" @click="isLogCollapsed = !isLogCollapsed">
              <div class="p-1.5 rounded-lg bg-secondary/50 group-hover:scale-110 transition-transform">
                <ChevronDownIcon v-if="!isLogCollapsed" class="w-4 h-4" />
                <ChevronUpIcon v-else class="w-4 h-4" />
              </div>
              <span class="text-sm font-semibold">系统日志</span>
              <div v-if="isLogCollapsed" class="ml-auto w-1.5 h-1.5 rounded-full bg-primary animate-pulse"></div>
            </button>
            <div id="log-panel-content" v-show="!isLogCollapsed" class="flex-1 min-h-0">
              <LogPanel />
            </div>
          </div>
        </main>
      </div>

      <!-- 应用更新 -->
      <div v-if="showUpdateDialog" class="fixed inset-0 z-[10000] flex items-center justify-center px-6"
        style="--wails-draggable: no-drag">
        <div class="absolute inset-0 bg-black/45 backdrop-blur-[2px]" @click="closeUpdater"></div>
        <div class="relative w-full max-w-lg max-h-[calc(100dvh-2rem)] overflow-y-auto rounded-2xl border border-border bg-card shadow-2xl p-5 space-y-5">
          <div class="flex items-start gap-3">
            <div class="mt-0.5 rounded-xl bg-primary/15 p-2.5 text-primary">
              <UpdateIcon class="h-5 w-5" :class="{ 'animate-spin': isCheckingUpdate || isInstallingUpdate }" />
            </div>
            <div class="min-w-0 flex-1 space-y-1">
              <h2 class="text-lg font-bold">应用更新</h2>
              <p v-if="isCheckingUpdate" class="text-sm text-muted-foreground">正在检查最新版本…</p>
              <p v-else-if="isInstallingUpdate" class="text-sm text-muted-foreground">
                正在下载并校验更新包<span v-if="updateProgress > 0">（{{ updateProgress }}%）</span>，请勿关闭程序…
              </p>
              <p v-else-if="updateInfo?.available" class="text-sm text-muted-foreground">
                v{{ updateInfo.currentVersion }} → <span class="font-semibold text-foreground">v{{ updateInfo.latestVersion }}</span>
                · {{ formatBytes(updateInfo.assetSize) }}
              </p>
              <p v-else-if="updateError" class="text-sm text-muted-foreground">无法获取更新信息。</p>
              <p v-else class="text-sm text-muted-foreground">当前已是最新版本<span v-if="updateInfo">（v{{ updateInfo.currentVersion }}）</span>。</p>
            </div>
          </div>

          <div v-if="updateError" class="rounded-xl border border-destructive/35 bg-destructive/10 px-3 py-2.5 text-sm text-destructive">
            {{ updateError }}
          </div>

          <div v-if="isInstallingUpdate" class="h-1.5 overflow-hidden rounded-full bg-secondary/70">
            <div class="h-full rounded-full bg-primary transition-[width] duration-200"
              :class="{ 'animate-pulse': updateProgress === 0 }"
              :style="{ width: updateProgress > 0 ? `${updateProgress}%` : '12%' }"></div>
          </div>

          <div v-if="updateInfo?.available && updateInfo.releaseNotes" class="space-y-2">
            <div class="text-sm font-semibold text-muted-foreground">更新说明</div>
            <div class="max-h-52 overflow-y-auto whitespace-pre-wrap rounded-xl border border-border/70 bg-background/60 p-3 text-sm leading-6 text-muted-foreground">{{ updateInfo.releaseNotes }}</div>
          </div>

          <p v-if="updateInfo?.available && !isLinux" class="text-xs leading-5 text-muted-foreground">
            安装包会先进行加密摘要校验。下载完成后应用将自动退出、替换并重新启动。
          </p>
          <p v-else-if="updateInfo?.available" class="text-xs leading-5 text-muted-foreground">
            当前 Wails 版本无法自动替换运行中的 AppImage。下载新版本并覆盖原 AppImage 即可，旁边的 iOSGhostRun-data 目录会保留。
          </p>

          <div class="flex flex-wrap items-center justify-end gap-2">
            <button class="px-4 h-9 rounded-md border border-border hover:bg-secondary/40 transition-colors disabled:opacity-50"
              :disabled="isCheckingUpdate || isInstallingUpdate" @click="closeUpdater">
              稍后
            </button>
            <button v-if="!updateInfo?.available"
              class="px-4 h-9 rounded-md bg-primary text-primary-foreground hover:opacity-90 transition-opacity disabled:opacity-50"
              :disabled="isCheckingUpdate || isInstallingUpdate" @click="checkForUpdate(true)">
              重新检查
            </button>
            <button v-else
              class="px-4 h-9 rounded-md bg-primary text-primary-foreground hover:opacity-90 transition-opacity disabled:opacity-50"
              :disabled="isCheckingUpdate || isInstallingUpdate" @click="isLinux ? openReleasePage() : installUpdate()">
              {{ isLinux ? '打开下载页面' : isInstallingUpdate ? '正在准备…' : '立即更新' }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="showCloseDialog" class="fixed inset-0 z-[10000] flex items-center justify-center px-6"
        style="--wails-draggable: no-drag">
        <div class="absolute inset-0 bg-black/45 backdrop-blur-[2px]" @click="showCloseDialog = false"></div>
        <div class="relative w-full max-w-md max-h-[calc(100dvh-2rem)] overflow-y-auto rounded-2xl border border-border bg-card shadow-2xl p-5 space-y-5">
          <div class="space-y-2">
            <h2 class="text-lg font-bold">关闭 iOSGhostRun</h2>
            <p class="text-sm text-muted-foreground">你希望直接退出程序，还是最小化到任务栏？</p>
          </div>
          <div class="flex flex-wrap items-center justify-end gap-2">
            <button class="px-4 h-9 rounded-md border border-border hover:bg-secondary/40 transition-colors"
              @click="showCloseDialog = false">
              取消
            </button>
            <button class="px-4 h-9 rounded-md border border-border hover:bg-secondary/40 transition-colors"
              @click="minimiseToTaskbar">
              最小化到任务栏
            </button>
            <button
              class="px-4 h-9 rounded-md bg-destructive text-destructive-foreground hover:opacity-90 transition-opacity"
              @click="quitApp">
              关闭程序
            </button>
          </div>
        </div>
      </div>

      <!-- 开发者模式提醒弹窗 -->
      <div v-if="showDeveloperModeAlert" class="fixed inset-0 z-[10000] flex items-center justify-center px-6"
        style="--wails-draggable: no-drag">
        <div class="absolute inset-0 bg-black/45 backdrop-blur-[2px]" @click="showDeveloperModeAlert = false"></div>
        <div class="relative w-full max-w-md max-h-[calc(100dvh-2rem)] overflow-y-auto rounded-2xl border border-border bg-card shadow-2xl p-5 space-y-5">
          <div class="space-y-2">
            <h2 class="text-lg font-bold">启用开发者模式</h2>
            <p class="text-sm text-muted-foreground">{{ developerModeAlertMessage }}</p>
          </div>
          <div class="flex items-center justify-end gap-2">
            <button class="px-4 h-9 rounded-md bg-primary text-primary-foreground hover:opacity-90 transition-opacity"
              @click="showDeveloperModeAlert = false">
              我已启用
            </button>
          </div>
        </div>
      </div>
    </div>
  </TooltipProvider>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import {
  ChevronDownIcon,
  ChevronUpIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  MinusIcon,
  SquareIcon,
  CopyIcon,
  Cross1Icon,
  UpdateIcon
} from '@radix-icons/vue'
import { Browser, Events, System, Window } from '@wailsio/runtime'
import type { RunningState, UpdateInfo } from '../bindings/iOSGhostRun/services/models'
import { CheckForUpdate, DownloadAndInstall } from '../bindings/iOSGhostRun/services/updateservice'
import MapEditor from './components/MapEditor.vue'
import LogPanel from './components/LogPanel.vue'
import DevicePanel from './components/DevicePanel.vue'
import RunningControl from './components/RunningControl.vue'
import Notification from './components/Notification.vue'
import { useRoutesStore, type RoutePoint } from './stores/routes'
import { ScrollArea } from '@/components/ui/scroll-area'
import { TooltipProvider } from '@/components/ui/tooltip'
import { TitlebarButton } from '@/components/ui/titlebar-button'
import { useNotification } from './composables/useNotification'

const mapEditor = ref<InstanceType<typeof MapEditor> | null>(null)
const selectedUdid = ref('')
const routePoints = ref<RoutePoint[]>([])
const currentPosition = ref<{ lat: number; lon: number } | null>(null)
const isRunning = ref(false)
const isLogCollapsed = ref(true)
const showCloseDialog = ref(false)
const isMacOS = ref(System.IsMac() || navigator.userAgent.includes('Mac OS X'))
const isLinux = System.IsLinux()
const showDeveloperModeAlert = ref(false)
const developerModeAlertMessage = ref('')
const isMaximized = ref(false)
const isSidebarCollapsed = ref(false)
const isCompactLayout = ref(false)
const updateInfo = ref<UpdateInfo | null>(null)
const showUpdateDialog = ref(false)
const isCheckingUpdate = ref(false)
const isInstallingUpdate = ref(false)
const updateProgress = ref(0)
const updateError = ref('')
const { showError } = useNotification()
let compactMediaQuery: MediaQueryList | null = null
let updateCheckTimer: number | null = null

function syncCompactLayout() {
  const compact = compactMediaQuery?.matches ?? false
  if (compact && !isCompactLayout.value) isSidebarCollapsed.value = true
  isCompactLayout.value = compact
}

function onPositionUpdate(pos: { lat: number; lon: number }) {
  currentPosition.value = pos
}

function onLocatingPoint(point: RoutePoint) {
  if (mapEditor.value) {
    mapEditor.value.centerOnPosition(point.lat, point.lon)
  }
}

function onRunStateChange(state: RunningState) {
  isRunning.value = state === 'running' || state === 'paused'
  if (!isRunning.value) {
    currentPosition.value = null
  }
}

async function onMinimise() {
  await Window.Minimise()
}

async function onToggleMaximise() {
  await Window.ToggleMaximise()
  isMaximized.value = !isMaximized.value
}

function requestClose() {
  showCloseDialog.value = true
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '未知大小'
  const units = ['B', 'KB', 'MB', 'GB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / Math.pow(1024, index)).toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

async function checkForUpdate(manual = false) {
  if (isCheckingUpdate.value || isInstallingUpdate.value) return
  isCheckingUpdate.value = true
  updateError.value = ''
  try {
    updateInfo.value = await CheckForUpdate()
    if (manual || updateInfo.value.available) showUpdateDialog.value = true
  } catch (error) {
    updateError.value = errorMessage(error)
    if (manual) showUpdateDialog.value = true
  } finally {
    isCheckingUpdate.value = false
  }
}

function openUpdater() {
  showUpdateDialog.value = true
  if (!updateInfo.value) void checkForUpdate(true)
}

function closeUpdater() {
  if (!isInstallingUpdate.value) showUpdateDialog.value = false
}

async function installUpdate() {
  if (!updateInfo.value?.available || isInstallingUpdate.value) return
  isInstallingUpdate.value = true
  updateProgress.value = 0
  updateError.value = ''
  try {
    await DownloadAndInstall(updateInfo.value.latestVersion)
  } catch (error) {
    updateError.value = errorMessage(error)
    showError(`更新失败：${updateError.value}`, 6000)
    isInstallingUpdate.value = false
  }
}

function openReleasePage() {
  void Browser.OpenURL('https://github.com/GH4NG/iOSGhostRun/releases/latest')
}

async function minimiseToTaskbar() {
  showCloseDialog.value = false
  await Window.Minimise()
}

async function quitApp() {
  showCloseDialog.value = false
  await Events.Emit('app:close-quit')
}

onMounted(() => {
  if (typeof window.matchMedia === 'function') {
    compactMediaQuery = window.matchMedia('(max-width: 760px)')
    syncCompactLayout()
    compactMediaQuery.addEventListener('change', syncCompactLayout)
  }
  isMacOS.value = System.IsMac() || navigator.userAgent.includes('Mac OS X')
  const routesStore = useRoutesStore()
  // 加载上次路线
  const lastRoute = routesStore.getLastRoute()
  if (lastRoute) {
    routePoints.value = lastRoute
    nextTick(() => {
      if (routePoints.value.length > 0) {
        onLocatingPoint(routePoints.value[0])
      }
    })
  }

  offCloseRequested = Events.On('app:close-requested', () => {
    showCloseDialog.value = true
  })

  offDeveloperModeAlert = Events.On('developer-mode-menu-revealed', event => {
    developerModeAlertMessage.value = event.data
    showDeveloperModeAlert.value = true
  })

  offUpdateProgress = Events.On('wails:updater:download-progress', event => {
    let data = event.data
    if (typeof data === 'string') {
      try {
        data = JSON.parse(data)
      } catch {
        return
      }
    }
    if (data?.total > 0) {
      updateProgress.value = Math.min(100, Math.round((data.written / data.total) * 100))
    }
  })

  // 启动后静默检查；只有发现新版本时才打扰用户。
  updateCheckTimer = window.setTimeout(() => void checkForUpdate(false), 1500)
})

let offCloseRequested: (() => void) | null = null
let offDeveloperModeAlert: (() => void) | null = null
let offUpdateProgress: (() => void) | null = null

onUnmounted(() => {
  if (updateCheckTimer !== null) window.clearTimeout(updateCheckTimer)
  compactMediaQuery?.removeEventListener('change', syncCompactLayout)
  if (offCloseRequested) {
    offCloseRequested()
    offCloseRequested = null
  }
  if (offDeveloperModeAlert) {
    offDeveloperModeAlert()
    offDeveloperModeAlert = null
  }
  offUpdateProgress?.()
  offUpdateProgress = null
})
</script>

<style>
.app-log-panel {
  height: 36px;
}

.app-log-panel.is-open {
  height: min(320px, 38%);
}

.is-compact .app-sidebar {
  position: absolute;
  inset: 0 auto 0 0;
  max-width: calc(100% - 48px);
}

.is-compact .app-main {
  margin-left: 56px;
}

@media (max-height: 480px) {
  .app-log-panel.is-open {
    height: 44%;
  }
}

.writing-mode-vertical {
  writing-mode: vertical-rl;
  text-orientation: mixed;
}

.no-scrollbar::-webkit-scrollbar {
  display: none;
}

.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
