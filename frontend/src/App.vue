<template>
  <TooltipProvider>
    <div
      class="h-screen w-screen flex flex-col bg-background text-foreground overflow-hidden font-sans selection:bg-primary/20 selection:text-primary">
      <!-- 通知系统 -->
      <Notification />

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
    </div>
  </TooltipProvider>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { ChevronDownIcon, ChevronUpIcon } from '@radix-icons/vue'
import MapEditor from './components/MapEditor.vue'
import LogPanel from './components/LogPanel.vue'
import DevicePanel from './components/DevicePanel.vue'
import RunningControl from './components/RunningControl.vue'
import Notification from './components/Notification.vue'
import { loadRoute, type RoutePoint } from './lib/routeStorage'
import { ScrollArea } from '@/components/ui/scroll-area'
import { TooltipProvider } from '@/components/ui/tooltip'

const mapEditor = ref<InstanceType<typeof MapEditor> | null>(null)
const selectedUdid = ref('')
const routePoints = ref<RoutePoint[]>([])
const currentPosition = ref<{ lat: number; lon: number } | null>(null)
const isRunning = ref(false)
const isLogCollapsed = ref(true)

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

onMounted(() => {
  // 加载上次路线
  const lastRoute = loadRoute('last_route')
  if (lastRoute) {
    routePoints.value = lastRoute
    nextTick(() => {
      if (routePoints.value.length > 0) {
        onLocatingPoint(routePoints.value[0])
      }
    })
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
