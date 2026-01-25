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
      <div class="space-y-2">
        <div class="flex justify-between items-center group/item">
          <label
            class="text-[10px] font-black text-muted-foreground/60 uppercase tracking-[0.2em] group-hover/item:text-primary transition-colors">平均速度</label>
          <div class="flex items-baseline gap-2">
            <input v-model.number="speed" type="number" min="1" max="20" step="0.5" :disabled="isRunning"
              class="w-16 px-2 py-1 text-sm font-mono bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <span class="text-[10px] font-bold text-muted-foreground/40 uppercase text-right w-8">KM/H</span>
          </div>
        </div>
        <Slider v-model="speedArray" :min="1" :max="20" :step="0.5" :disabled="isRunning" class="py-1" />
      </div>

      <!-- 速度随机波动 -->
      <div class="space-y-2">
        <div class="flex justify-between items-center group/item">
          <label
            class="text-[10px] font-black text-muted-foreground/60 uppercase tracking-[0.2em] group-hover/item:text-primary transition-colors">波动偏差</label>
          <div class="flex items-baseline gap-2">
            <input v-model.number="speedVariance" type="number" min="0" max="30" step="1" :disabled="isRunning"
              class="w-16 px-2 py-1 text-sm font-mono bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <span class="text-[10px] font-bold text-muted-foreground/40 uppercase text-right w-8">%</span>
          </div>
        </div>
        <Slider v-model="speedVarianceArray" :min="0" :max="30" :step="1" :disabled="isRunning" class="py-1" />
      </div>

      <!-- 路线偏移 -->
      <div class="space-y-2">
        <div class="flex justify-between items-center group/item">
          <label
            class="text-[10px] font-black text-muted-foreground/60 uppercase tracking-[0.2em] group-hover/item:text-primary transition-colors">路经补正</label>
          <div class="flex items-baseline gap-2">
            <input v-model.number="routeOffset" type="number" min="0" max="10" step="0.5" :disabled="isRunning"
              class="w-16 px-2 py-1 text-sm font-mono bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <span class="text-[10px] font-bold text-muted-foreground/40 uppercase text-right w-8">M</span>
          </div>
        </div>
        <Slider v-model="routeOffsetArray" :min="0" :max="10" :step="0.5" :disabled="isRunning" class="py-1" />
      </div>

      <!-- 循环圈数 -->
      <div class="space-y-2">
        <div class="flex justify-between items-center group/item">
          <label
            class="text-[10px] font-black text-muted-foreground/60 uppercase tracking-[0.2em] group-hover/item:text-primary transition-colors">循环圈数</label>
          <div class="flex items-baseline gap-2">
            <input v-model.number="loopCount" type="number" min="1" max="10" step="1" :disabled="isRunning"
              class="w-16 px-2 py-1 text-sm font-mono bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <span class="text-[10px] font-bold text-muted-foreground/40 uppercase text-right w-8">圈</span>
          </div>
        </div>
        <Slider v-model="loopCountArray" :min="1" :max="10" :step="1" :disabled="isRunning" class="py-1" />
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
import { formatDistance, formatTime, type RoutePoint } from '../lib/routeStorage'
import { useNotification } from '../composables/useNotification'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Slider } from '@/components/ui/slider'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const { showError: showErrorDialog, showSuccess } = useNotification()

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
const loopCount = ref(1)
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
const loopCountArray = computed({
  get: () => [loopCount.value],
  set: val => (loopCount.value = val[0])
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
    await RunningService.SetLoopCount(loopCount.value)
    await RunningService.SetRandomization(speedVariance.value / 100, routeOffset.value)
    await RunningService.StartRun(props.udid, props.routePoints, speed.value)
    await updateStatus()
  } catch (e) {
    showErrorDialog(`启动跑步失败: ${e instanceof Error ? e.message : '未知错误'}`)
  }
}

async function pauseRun() {
  try {
    await RunningService.PauseRun()
    await updateStatus()
  } catch (e) {
    showErrorDialog(`暂停失败: ${e instanceof Error ? e.message : '未知错误'}`)
  }
}

async function resumeRun() {
  try {
    // 恢复前重新应用设置，以支持暂停期间的参数修改
    await RunningService.SetSpeed(speed.value)
    await RunningService.SetRandomization(speedVariance.value / 100, routeOffset.value)
    await RunningService.ResumeRun()
    await updateStatus()
  } catch (e) {
    showErrorDialog(`恢复失败: ${e instanceof Error ? e.message : '未知错误'}`)
  }
}

async function stopRun() {
  try {
    await RunningService.StopRun()
    await updateStatus()
  } catch (e) {
    showErrorDialog(`停止失败: ${e instanceof Error ? e.message : '未知错误'}`)
  }
}

async function updateStatus() {
  try {
    status.value = (await RunningService.GetStatus()) as RunningStatus
  } catch (e) {
    showErrorDialog(`获取状态失败: ${e instanceof Error ? e.message : '未知错误'}`)
  }
}

onMounted(() => {
  updateStatus()

  // 定期更新状态
  const statusInterval = setInterval(() => {
    updateStatus()
  }, 500)

  // 监听位置更新事件
  Events.On('running:position', (ev: any) => {
    const data = ev.data as RunningStatus
    emit('position-update', { lat: data.currentLat, lon: data.currentLon })
    status.value = data
  })

  // 监听完成事件
  Events.On('running:completed', (ev: any) => {
    status.value = ev.data as RunningStatus
    emit('completed')
    showSuccess(`跑步任务已完成！共运行 ${status.value.totalPoints} 个位置点`)
  })

  onUnmounted(() => {
    clearInterval(statusInterval)
    Events.Off('running:position')
    Events.Off('running:completed')
  })
})

// 监听速度变化
watch(speed, async newSpeed => {
  if (isRunning.value) {
    try {
      await RunningService.SetSpeed(newSpeed)
    } catch (e) {
      showErrorDialog(`设置速度失败: ${e instanceof Error ? e.message : '未知错误'}`)
    }
  }
})

// 监听圈数变化
watch(loopCount, async newLoopCount => {
  if (!isRunning.value) {
    try {
      await RunningService.SetLoopCount(newLoopCount)
    } catch (e) {
      showErrorDialog(`设置圈数失败: ${e instanceof Error ? e.message : '未知错误'}`)
    }
  }
})
</script>

<style scoped></style>
