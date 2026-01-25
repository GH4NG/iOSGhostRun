<template>
  <Card class="flex flex-col bg-card border-border shadow-sm overflow-hidden text-foreground">
    <div class="flex items-center justify-between p-5 border-b border-border/40">
      <div class="flex items-center gap-3">
        <div class="p-1.5 rounded-lg bg-primary/10">
          <ActivityLogIcon class="w-5 h-5 text-primary" />
        </div>
        <span class="text-xs font-black uppercase tracking-widest text-foreground/80">跑步控制</span>
      </div>
      <Badge :variant="isPaused ? 'warning' : isRunning ? 'default' : 'secondary'"
        class="text-[9px] font-black px-2 py-0 h-5 border-none shadow-sm shadow-black/5">
        {{ statusText }}
      </Badge>
    </div>

    <div class="p-6 flex flex-col gap-8">
      <!-- 速度设置 -->
      <div class="space-y-4">
        <div class="flex justify-between items-center group/item">
          <label
            class="text-[10px] font-black text-muted-foreground/60 uppercase tracking-[0.2em] group-hover/item:text-primary transition-colors">平均速度</label>
          <div class="flex items-baseline gap-1">
            <span class="text-xl font-black font-mono tracking-tighter text-primary">{{
              speed.toFixed(1)
            }}</span>
            <span class="text-[10px] font-bold text-muted-foreground/40 uppercase">km/h</span>
          </div>
        </div>
        <Slider v-model="speedArray" :min="1" :max="20" :step="0.5" :disabled="isRunning" class="py-1" />
      </div>

      <!-- 速度随机波动 -->
      <div class="space-y-4">
        <div class="flex justify-between items-center group/item">
          <label
            class="text-[10px] font-black text-muted-foreground/60 uppercase tracking-[0.2em] group-hover/item:text-primary transition-colors">波动偏差</label>
          <div class="flex items-baseline gap-1">
            <span class="text-xl font-black font-mono tracking-tighter text-primary">{{
              speedVariance
            }}</span>
            <span class="text-[10px] font-bold text-muted-foreground/40 uppercase">%</span>
          </div>
        </div>
        <Slider v-model="speedVarianceArray" :min="0" :max="30" :step="1" :disabled="isRunning" class="py-1" />
      </div>

      <!-- 路线偏移 -->
      <div class="space-y-4">
        <div class="flex justify-between items-center group/item">
          <label
            class="text-[10px] font-black text-muted-foreground/60 uppercase tracking-[0.2em] group-hover/item:text-primary transition-colors">路经补正</label>
          <div class="flex items-baseline gap-1">
            <span class="text-xl font-black font-mono tracking-tighter text-primary">{{
              routeOffset
            }}</span>
            <span class="text-[10px] font-bold text-muted-foreground/40 uppercase">meters</span>
          </div>
        </div>
        <Slider v-model="routeOffsetArray" :min="0" :max="10" :step="0.5" :disabled="isRunning" class="py-1" />
      </div>

      <!-- 状态信息 -->
      <div v-if="status"
        class="grid grid-cols-3 gap-3 p-4 bg-background border border-border/40 rounded-2xl shadow-inner relative overflow-hidden animate-in fade-in scale-in-95 duration-500">
        <div
          class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-primary/50 to-transparent">
        </div>
        <div class="flex flex-col items-center gap-1.5">
          <span class="text-[9px] font-black text-muted-foreground uppercase opacity-40">Progress</span>
          <span class="text-sm font-black mono tracking-tighter">{{ status.currentIndex }}/{{ status.totalPoints
            }}</span>
        </div>
        <div class="flex flex-col items-center gap-1.5 border-x border-border/40">
          <span class="text-[9px] font-black text-muted-foreground uppercase opacity-40">Distance</span>
          <span class="text-sm font-black mono tracking-tighter text-primary">{{
            formatDist(status.distance)
          }}</span>
        </div>
        <div class="flex flex-col items-center gap-1.5">
          <span class="text-[9px] font-black text-muted-foreground uppercase opacity-40">Time</span>
          <span class="text-sm font-black mono tracking-tighter">{{
            formatTimeValue(status.elapsedTimeMs)
          }}</span>
        </div>
      </div>

      <!-- 控制按钮 -->
      <div class="flex gap-3">
        <Tooltip v-if="!isRunning && !isPaused">
          <TooltipTrigger asChild>
            <Button
              class="flex-1 h-12 gap-2 text-xs font-black uppercase tracking-widest shadow-lg shadow-primary/20 transition-all hover:scale-105 active:scale-95"
              :disabled="!canStart" @click="startRun">
              <PlayIcon class="w-4 h-4 fill-current" />
              <span>开始任务</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>开始同步位置至设备</TooltipContent>
        </Tooltip>

        <Tooltip v-if="isRunning">
          <TooltipTrigger asChild>
            <Button variant="secondary"
              class="flex-1 h-12 gap-2 text-xs font-black uppercase tracking-widest bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 border-amber-500/20 transition-all hover:scale-105 active:scale-95"
              @click="pauseRun">
              <PauseIcon class="w-4 h-4 fill-current" />
              <span>暂停</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>暂停当前同步任务</TooltipContent>
        </Tooltip>

        <Tooltip v-if="isPaused">
          <TooltipTrigger asChild>
            <Button
              class="flex-1 h-12 gap-2 text-xs font-black uppercase tracking-widest shadow-lg shadow-primary/20 transition-all hover:scale-105 active:scale-95"
              @click="resumeRun">
              <PlayIcon class="w-4 h-4 fill-current" />
              <span>恢复</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>恢复位置同步</TooltipContent>
        </Tooltip>

        <Tooltip v-if="isRunning || isPaused">
          <TooltipTrigger asChild>
            <Button variant="destructive"
              class="flex-1 h-12 gap-2 text-xs font-black uppercase tracking-widest shadow-lg shadow-destructive/20 transition-all hover:scale-105 active:scale-95"
              @click="stopRun">
              <StopIcon class="w-4 h-4 fill-current" />
              <span>停止</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>终止当前同步任务</TooltipContent>
        </Tooltip>
      </div>
    </div>
  </Card>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { ActivityLogIcon, PlayIcon, PauseIcon, StopIcon } from '@radix-icons/vue'
import { Events } from '@wailsio/runtime'
import { RunningService } from '../../bindings/iOSGhostRun/services'
import { formatDistance, formatTime, type RoutePoint } from '../utils/routeStorage'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Slider } from '@/components/ui/slider'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

interface RunningStatus {
  state: 'idle' | 'running' | 'paused'
  currentIndex: number
  totalPoints: number
  currentLat: number
  currentLon: number
  speed: number
  distance: number
  elapsedTimeMs: number
}

const props = defineProps<{
  udid: string
  routePoints: RoutePoint[]
}>()

const emit = defineEmits<{
  'position-update': [pos: { lat: number; lon: number }]
  completed: []
}>()

const speed = ref(8)
const speedVariance = ref(10)
const routeOffset = ref(2)
const status = ref<RunningStatus | null>(null)

const speedArray = computed({
  get: () => [speed.value],
  set: val => (speed.value = val[0])
})
const speedVarianceArray = computed({
  get: () => [speedVariance.value],
  set: val => (speedVariance.value = val[0])
})
const routeOffsetArray = computed({
  get: () => [routeOffset.value],
  set: val => (routeOffset.value = val[0])
})

const isRunning = computed(() => status.value?.state === 'running')
const isPaused = computed(() => status.value?.state === 'paused')
const canStart = computed(() => props.udid && props.routePoints.length >= 2)

const statusText = computed(() => {
  if (isRunning.value) return '运行中'
  if (isPaused.value) return '已暂停'
  return '待机'
})

function formatDist(km: number) {
  return formatDistance(km)
}

function formatTimeValue(ms: number) {
  return formatTime(ms)
}

async function startRun() {
  if (!canStart.value) return

  try {
    await RunningService.SetRandomization(speedVariance.value / 100, routeOffset.value)
    await RunningService.StartRun(props.udid, props.routePoints, speed.value)
    await updateStatus()
  } catch (e) {
    console.error('Failed to start run:', e)
  }
}

async function pauseRun() {
  try {
    await RunningService.PauseRun()
    await updateStatus()
  } catch (e) {
    console.error('Failed to pause:', e)
  }
}

async function resumeRun() {
  try {
    await RunningService.ResumeRun()
    await updateStatus()
  } catch (e) {
    console.error('Failed to resume:', e)
  }
}

async function stopRun() {
  try {
    await RunningService.StopRun()
    await updateStatus()
  } catch (e) {
    console.error('Failed to stop:', e)
  }
}

async function updateStatus() {
  try {
    status.value = (await RunningService.GetStatus()) as RunningStatus
  } catch (e) {
    console.error('Failed to get status:', e)
  }
}

onMounted(() => {
  updateStatus()

  // 监听位置更新事件
  Events.On('running:position', (ev: any) => {
    const data = ev.data
    emit('position-update', { lat: data.lat, lon: data.lon })
  })

  // 监听完成事件
  Events.On('running:completed', (ev: any) => {
    status.value = ev.data as RunningStatus
    emit('completed')
  })
})

onUnmounted(() => {
  Events.Off('running:position')
  Events.Off('running:completed')
})

// 监听速度变化
watch(speed, async newSpeed => {
  if (isRunning.value) {
    try {
      await RunningService.SetSpeed(newSpeed)
    } catch (e) {
      console.error('Failed to set speed:', e)
    }
  }
})
</script>

<style scoped></style>
