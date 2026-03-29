<template>
  <TooltipProvider>
    <div
      class="h-screen w-screen flex flex-col bg-background text-foreground overflow-hidden font-sans selection:bg-primary/20 selection:text-primary">
      <!-- 通知系统 -->
      <Notification />

      <!-- 自定义标题栏 -->
      <header class="h-11 shrink-0 border-b border-border bg-card/70 backdrop-blur-md relative flex items-center"
        :class="isMacOS ? 'px-3' : 'pl-4 pr-1 justify-between'" style="--wails-draggable: drag">
        <template v-if="isMacOS">
          <div class="flex items-center gap-2" style="--wails-draggable: no-drag">
            <button
              class="group w-3 h-3 rounded-full bg-[#ff5f57] hover:brightness-95 transition flex items-center justify-center"
              aria-label="关闭" @click="requestClose">
              <span
                class="text-[8px] leading-none text-black/70 opacity-0 group-hover:opacity-100 transition-opacity">×</span>
            </button>
            <button
              class="group w-3 h-3 rounded-full bg-[#febc2e] hover:brightness-95 transition flex items-center justify-center"
              aria-label="最小化" @click="onMinimise">
              <span
                class="text-[8px] leading-none text-black/70 opacity-0 group-hover:opacity-100 transition-opacity">-</span>
            </button>
            <button
              class="group w-3 h-3 rounded-full bg-[#28c840] hover:brightness-95 transition flex items-center justify-center"
              aria-label="最大化或还原" @click="onToggleMaximise">
              <span
                class="text-[8px] leading-none text-black/70 opacity-0 group-hover:opacity-100 transition-opacity">+</span>
            </button>
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
            <button
              class="w-9 h-7 rounded-md text-muted-foreground hover:text-foreground hover:bg-secondary/60 transition-colors text-lg leading-none"
              aria-label="最小化" @click="onMinimise">
              -
            </button>
            <button
              class="w-9 h-7 rounded-md text-muted-foreground hover:text-foreground hover:bg-secondary/60 transition-colors text-sm leading-none"
              aria-label="最大化或还原" @click="onToggleMaximise">
              □
            </button>
            <button
              class="w-9 h-7 rounded-md text-muted-foreground hover:text-white hover:bg-destructive transition-colors text-base leading-none"
              aria-label="关闭" @click="requestClose">
              ×
            </button>
          </div>
        </template>
      </header>

      <div class="flex-1 flex overflow-hidden">
        <!-- 左侧面板 -->
        <aside
          class="w-80 border-r border-border bg-card/50 backdrop-blur-md flex flex-col p-0 gap-0 overflow-hidden shadow-2xl z-20">
          <div class="flex-1 min-h-0">
            <ScrollArea class="h-full px-4 pb-6">
              <div class="flex flex-col gap-6 pt-2">
                <DevicePanel v-model="selectedUdid" />
                <RunningControl :udid="selectedUdid" :route-points="routePoints" @position-update="onPositionUpdate"
                  @completed="onRunCompleted" />
              </div>
            </ScrollArea>
          </div>
        </aside>

        <!-- 右侧区域: 地图 + 日志 -->
        <main class="flex-1 flex flex-col min-w-0 relative bg-background">
          <!-- 地图区域 -->
          <div class="flex-1 relative min-h-0">
            <MapEditor ref="mapEditor" v-model="routePoints" :current-position="currentPosition"
              :disabled="isRunning" />
          </div>

          <!-- 日志区域 -->
          <div
            class="bg-card/85 backdrop-blur-xl border-t border-border z-30 transition-all duration-500 ease-[cubic-bezier(0.19,1,0.22,1)] flex flex-col shadow-[0_-10px_30px_rgba(0,0,0,0.1)]"
            :class="[isLogCollapsed ? 'h-12' : 'h-80']">
            <div
              class="h-12 flex items-center px-6 gap-3 cursor-pointer select-none bg-secondary/10 hover:bg-secondary/20 text-muted-foreground hover:text-foreground transition-all group border-b border-transparent"
              :class="{ 'border-border/30': !isLogCollapsed }" @click="isLogCollapsed = !isLogCollapsed">
              <div class="p-1.5 rounded-lg bg-secondary/50 group-hover:scale-110 transition-transform">
                <ChevronDownIcon v-if="!isLogCollapsed" class="w-4 h-4" />
                <ChevronUpIcon v-else class="w-4 h-4" />
              </div>
              <span class="text-xs font-black uppercase tracking-[0.2em]">系统日志</span>
              <div v-if="isLogCollapsed" class="ml-auto w-1.5 h-1.5 rounded-full bg-primary animate-pulse"></div>
            </div>
            <div v-show="!isLogCollapsed" class="flex-1 min-h-0">
              <LogPanel />
            </div>
          </div>
        </main>
      </div>

      <div v-if="showCloseDialog" class="fixed inset-0 z-[10000] flex items-center justify-center px-6"
        style="--wails-draggable: no-drag">
        <div class="absolute inset-0 bg-black/45 backdrop-blur-[2px]" @click="showCloseDialog = false"></div>
        <div class="relative w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl p-6 space-y-5">
          <div class="space-y-2">
            <h2 class="text-lg font-bold">关闭 iOSGhostRun</h2>
            <p class="text-sm text-muted-foreground">你希望直接退出程序，还是最小化到任务栏？</p>
          </div>
          <div class="flex items-center justify-end gap-2">
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
        <div class="relative w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl p-6 space-y-5">
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
import { ChevronDownIcon, ChevronUpIcon } from '@radix-icons/vue'
import { Events, System, Window } from '@wailsio/runtime'
import MapEditor from './components/MapEditor.vue'
import LogPanel from './components/LogPanel.vue'
import DevicePanel from './components/DevicePanel.vue'
import RunningControl from './components/RunningControl.vue'
import Notification from './components/Notification.vue'
import { useRoutesStore, type RoutePoint } from './stores/routes'
import { ScrollArea } from '@/components/ui/scroll-area'
import { TooltipProvider } from '@/components/ui/tooltip'

const mapEditor = ref<InstanceType<typeof MapEditor> | null>(null)
const selectedUdid = ref('')
const routePoints = ref<RoutePoint[]>([])
const currentPosition = ref<{ lat: number; lon: number } | null>(null)
const isRunning = ref(false)
const isLogCollapsed = ref(true)
const showCloseDialog = ref(false)
const isMacOS = ref(System.IsMac())
const showDeveloperModeAlert = ref(false)
const developerModeAlertMessage = ref('')

function onPositionUpdate(pos: { lat: number; lon: number }) {
  currentPosition.value = pos
  isRunning.value = true
}

function onLocatingPoint(point: RoutePoint) {
  if (mapEditor.value) {
    mapEditor.value.centerOnPosition(point.lat, point.lon)
  }
}

function onRunCompleted() {
  isRunning.value = false
  currentPosition.value = null
}

async function onMinimise() {
  await Window.Minimise()
}

async function onToggleMaximise() {
  await Window.ToggleMaximise()
}

function requestClose() {
  showCloseDialog.value = true
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

  offDeveloperModeAlert = Events.On('developer-mode-menu-revealed', (event) => {
    developerModeAlertMessage.value = event.data
    showDeveloperModeAlert.value = true
  })
})

let offCloseRequested: (() => void) | null = null
let offDeveloperModeAlert: (() => void) | null = null

onUnmounted(() => {
  if (offCloseRequested) {
    offCloseRequested()
    offCloseRequested = null
  }
  if (offDeveloperModeAlert) {
    offDeveloperModeAlert()
    offDeveloperModeAlert = null
  }
})
</script>

<style>
.no-scrollbar::-webkit-scrollbar {
  display: none;
}

.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
