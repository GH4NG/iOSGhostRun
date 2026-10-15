<template>
  <Card
    class="panel-card flex flex-col gap-0 p-0 overflow-hidden font-sans">
    <div class="panel-header">
      <div class="flex min-w-0 items-center gap-2">
        <div class="panel-icon">
          <ActivityLogIcon class="w-4 h-4 text-primary" />
        </div>
        <div class="min-w-0">
          <span class="panel-title">跑步控制</span>
          <span class="panel-description">速度、偏移与循环策略</span>
        </div>
      </div>
      <Badge :variant="isPaused ? 'warning' : isRunning ? 'default' : 'secondary'"
        class="shrink-0 text-xs font-semibold px-2 py-0 h-5 rounded-full border-none shadow-sm shadow-black/5">
        {{ statusText }}
      </Badge>
    </div>

    <div class="p-3 flex flex-col gap-3 bg-card/70">
      <!-- 速度设置 -->
      <div class="space-y-3 rounded-lg border border-border/40 bg-background/45 p-3">
        <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 group/item">
          <label for="run-speed"
            class="text-xs font-semibold text-muted-foreground group-hover/item:text-primary transition-colors">平均速度</label>
          <div class="flex items-center gap-2">
            <input id="run-speed" v-model.number="displaySpeed" type="number" :min="speedRange.min" :max="speedRange.max"
              :step="speedRange.step" :disabled="isRunning"
              class="h-8 w-14 px-2 text-right text-sm font-mono tabular-nums bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <Select v-model="speedUnit" :disabled="isRunning">
              <SelectTrigger size="sm"
                class="w-20 border-border/50 bg-card/80 px-2 text-xs font-medium">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="option in speedUnitOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <Slider v-model="speedArray" :min="speedRange.min" :max="speedRange.max" :step="speedRange.step"
          :disabled="isRunning" class="py-1" />
      </div>

      <!-- 速度波动 -->
      <div class="space-y-3 rounded-lg border border-border/40 bg-background/45 p-3">
        <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 group/item">
          <label for="run-variance"
            class="text-xs font-semibold text-muted-foreground group-hover/item:text-primary transition-colors">速度波动</label>
          <div class="flex items-center gap-2">
            <input id="run-variance" v-model.number="speedVariance" type="number" min="0" max="30" step="1" :disabled="isRunning"
              class="h-8 w-14 px-2 text-right text-sm font-mono tabular-nums bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <span class="text-xs font-medium text-muted-foreground w-20 pl-2 text-left">±%</span>
          </div>
        </div>
        <Slider v-model="speedVarianceArray" :min="0" :max="30" :step="1" :disabled="isRunning" class="py-1" />
      </div>

      <!-- 路线偏移 -->
      <div class="space-y-3 rounded-lg border border-border/40 bg-background/45 p-3">
        <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 group/item">
          <label for="run-offset"
            class="text-xs font-semibold text-muted-foreground group-hover/item:text-primary transition-colors">路线偏移</label>
          <div class="flex items-center gap-2">
            <input id="run-offset" v-model.number="routeOffset" type="number" min="0" max="10" step="0.5" :disabled="isRunning"
              class="h-8 w-14 px-2 text-right text-sm font-mono tabular-nums bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <span class="text-xs font-medium text-muted-foreground w-20 pl-2 text-left">m</span>
          </div>
        </div>
        <Slider v-model="routeOffsetArray" :min="0" :max="10" :step="0.5" :disabled="isRunning" class="py-1" />
      </div>

      <!-- 循环圈数 -->
      <div class="space-y-3 rounded-lg border border-border/40 bg-background/45 p-3">
        <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 group/item">
          <label for="run-loops"
            class="text-xs font-semibold text-muted-foreground group-hover/item:text-primary transition-colors">循环圈数</label>
          <div class="flex items-center gap-2">
            <input id="run-loops" v-model.number="loopCount" type="number" min="1" max="10" step="1" :disabled="isRunning"
              class="h-8 w-14 px-2 text-right text-sm font-mono tabular-nums bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <span class="text-xs font-medium text-muted-foreground w-20 pl-2 text-left">圈</span>
          </div>
        </div>
        <Slider v-model="loopCountArray" :min="1" :max="10" :step="1" :disabled="isRunning" class="py-1" />
      </div>

      <!-- 状态信息 -->
      <div v-if="status"
        class="grid grid-cols-3 gap-2 p-3 bg-background/70 border border-border/40 rounded-2xl shadow-inner relative overflow-hidden animate-in fade-in scale-in-95 duration-500">
        <div
          class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-primary/50 to-transparent">
        </div>
        <div class="flex flex-col items-center gap-1.5 rounded-xl bg-secondary/20 py-2">
          <span class="text-xs font-medium text-muted-foreground">进度</span>
          <span class="text-sm font-semibold font-mono tabular-nums tracking-tight">{{ status.currentIndex }}/{{ status.totalPoints
            }}</span>
        </div>
        <div class="flex flex-col items-center gap-1.5 rounded-xl bg-primary/10 py-2">
          <span class="text-xs font-medium text-muted-foreground">距离</span>
          <span class="text-sm font-semibold font-mono tabular-nums tracking-tight text-primary">{{
            formatDist(status.distance)
          }}</span>
        </div>
        <div class="flex flex-col items-center gap-1.5 rounded-xl bg-secondary/20 py-2">
          <span class="text-xs font-medium text-muted-foreground">用时</span>
          <span class="text-sm font-semibold font-mono tabular-nums tracking-tight">{{
            formatTimeValue(status.elapsedTimeMs)
          }}</span>
        </div>
      </div>

      <!-- 控制按钮 -->
      <div class="flex gap-2 pt-1">
        <Tooltip v-if="!isRunning && !isPaused">
          <TooltipTrigger asChild>
            <Button
              class="flex-1 h-10 gap-2 text-sm font-semibold shadow-lg shadow-primary/20 transition-all active:opacity-90"
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
              class="flex-1 h-10 gap-2 text-sm font-semibold bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 border-amber-500/20 transition-all active:opacity-90"
              @click="pauseRun">
              <PauseIcon class="w-4 h-4 fill-current" />
              <span>暂停</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>暂停当前任务</TooltipContent>
        </Tooltip>

        <Tooltip v-if="isPaused">
          <TooltipTrigger asChild>
            <Button
              class="flex-1 h-10 gap-2 text-sm font-semibold shadow-lg shadow-primary/20 transition-all active:opacity-90"
              @click="resumeRun">
              <PlayIcon class="w-4 h-4 fill-current" />
              <span>恢复</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>恢复当前任务</TooltipContent>
        </Tooltip>

        <Tooltip v-if="isRunning || isPaused">
          <TooltipTrigger asChild>
            <Button variant="destructive"
              class="flex-1 h-10 gap-2 text-sm font-semibold shadow-lg shadow-destructive/20 transition-all active:opacity-90"
              @click="stopRun">
              <StopIcon class="w-4 h-4 fill-current" />
              <span>停止</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>终止当前任务</TooltipContent>
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
import type { RunningStatus } from '../../bindings/iOSGhostRun/services/models'
import { formatDistance, formatTime, type RoutePoint } from '../lib/routeUtils'
import { useNotification } from '../composables/useNotification'
import { useRunningParamsStore } from '../stores/runningParams'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Slider } from '@/components/ui/slider'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

type SpeedUnit = 'km/h' | 'm/s' | 'mph'

const props = defineProps<{
  udid: string
  routePoints: RoutePoint[]
}>()

const emit = defineEmits<{
  'position-update': [pos: { lat: number; lon: number }]
  'state-change': [state: RunningStatus['state']]
  completed: []
}>()

const { showError: showErrorDialog, showSuccess } = useNotification()
const paramsStore = useRunningParamsStore()

const speed = computed({
  get: () => paramsStore.params.speed,
  set: val => paramsStore.setParams({ speed: val })
})

const speedUnit = computed({
  get: () => paramsStore.params.speedUnit,
  set: val => paramsStore.setParams({ speedUnit: val as SpeedUnit })
})

const speedVariance = computed({
  get: () => paramsStore.params.speedVariance,
  set: val => paramsStore.setParams({ speedVariance: clamp(Number.isFinite(val) ? val : 0, 0, 30) })
})

const routeOffset = computed({
  get: () => paramsStore.params.routeOffset,
  set: val => paramsStore.setParams({ routeOffset: clamp(Number.isFinite(val) ? val : 0, 0, 10) })
})

const loopCount = computed({
  get: () => paramsStore.params.loopCount,
  set: val => paramsStore.setParams({ loopCount: val })
})

const status = ref<RunningStatus | null>(null)
let statusRevision = 0
let disposed = false
let statusInterval: ReturnType<typeof setInterval> | null = null
let offPosition: (() => void) | null = null
let offCompleted: (() => void) | null = null
let offError: (() => void) | null = null

function applyStatus(data: RunningStatus) {
  statusRevision++
  status.value = data
}

const speedUnitOptions: Array<{ value: SpeedUnit; label: string }> = [
  { value: 'km/h', label: 'km/h' },
  { value: 'm/s', label: 'm/s' },
  { value: 'mph', label: 'mph' }
]

const SPEED_RANGES: Record<SpeedUnit, { min: number; max: number; step: number }> = {
  'km/h': { min: 1, max: 20, step: 0.5 },
  'm/s': { min: 0.5, max: 5.5, step: 0.1 },
  mph: { min: 0.5, max: 12.5, step: 0.5 }
}

const speedRange = computed(() => SPEED_RANGES[speedUnit.value])

function convertFromKMH(value: number, unit: SpeedUnit) {
  if (unit === 'm/s') return value / 3.6
  if (unit === 'mph') return value / 1.609344
  return value
}

function convertToKMH(value: number, unit: SpeedUnit) {
  if (unit === 'm/s') return value * 3.6
  if (unit === 'mph') return value * 1.609344
  return value
}

function roundToStep(value: number, step: number) {
  const rounded = Math.round(value / step) * step
  return Number(rounded.toFixed(2))
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max)
}

const displaySpeed = computed({
  get: () => roundToStep(convertFromKMH(speed.value, speedUnit.value), speedRange.value.step),
  set: val => {
    const normalized = clamp(
      Number.isFinite(val) ? val : speedRange.value.min,
      speedRange.value.min,
      speedRange.value.max
    )
    speed.value = Number(convertToKMH(normalized, speedUnit.value).toFixed(2))
  }
})

const speedArray = computed({
  get: () => [displaySpeed.value],
  set: val => (displaySpeed.value = val[0])
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

watch(
  () => status.value?.state,
  state => {
    if (state) emit('state-change', state)
  },
  { immediate: true }
)

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
  const revision = ++statusRevision
  try {
    const data = await RunningService.GetStatus()
    if (!disposed && revision === statusRevision) status.value = data
  } catch (e) {
    if (!disposed && revision === statusRevision) {
      showErrorDialog(`获取状态失败: ${e instanceof Error ? e.message : '未知错误'}`)
    }
  }
}

onMounted(() => {
  updateStatus()

  // 定期更新状态
  statusInterval = setInterval(() => {
    updateStatus()
  }, 500)

  // 监听位置更新事件
  offPosition = Events.On('running:position', ev => {
    const data = ev.data
    applyStatus(data)
    emit('position-update', { lat: data.currentLat, lon: data.currentLon })
  })

  // 监听完成事件
  offCompleted = Events.On('running:completed', ev => {
    applyStatus(ev.data)
    emit('completed')
    showSuccess(`跑步任务已完成！共运行 ${ev.data.totalPoints} 个位置点`)
  })

  offError = Events.On('running:error', ev => {
    updateStatus()
    showErrorDialog(`跑步任务失败: ${ev.data}`)
  })
})

onUnmounted(() => {
  disposed = true
  statusRevision++
  if (statusInterval !== null) clearInterval(statusInterval)
  offPosition?.()
  offCompleted?.()
  offError?.()
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
