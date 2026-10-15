<template>
  <Card :aria-busy="isOperating"
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
              :step="speedRange.step" :disabled="parametersDisabled"
              class="h-8 w-14 px-2 text-right text-sm font-mono tabular-nums bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <Select v-model="speedUnit" :disabled="parametersDisabled">
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
          :disabled="parametersDisabled" class="py-1" />
      </div>

      <!-- 速度波动 -->
      <div class="space-y-3 rounded-lg border border-border/40 bg-background/45 p-3">
        <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 group/item">
          <label for="run-variance"
            class="text-xs font-semibold text-muted-foreground group-hover/item:text-primary transition-colors">速度波动</label>
          <div class="flex items-center gap-2">
            <input id="run-variance" v-model.number="speedVariance" type="number" min="0" max="30" step="1" :disabled="parametersDisabled"
              class="h-8 w-14 px-2 text-right text-sm font-mono tabular-nums bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <span class="text-xs font-medium text-muted-foreground w-20 pl-2 text-left">±%</span>
          </div>
        </div>
        <Slider v-model="speedVarianceArray" :min="0" :max="30" :step="1" :disabled="parametersDisabled" class="py-1" />
      </div>

      <!-- 路线偏移 -->
      <div class="space-y-3 rounded-lg border border-border/40 bg-background/45 p-3">
        <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 group/item">
          <label for="run-offset"
            class="text-xs font-semibold text-muted-foreground group-hover/item:text-primary transition-colors">路线偏移</label>
          <div class="flex items-center gap-2">
            <input id="run-offset" v-model.number="routeOffset" type="number" min="0" max="10" step="0.5" :disabled="parametersDisabled"
              class="h-8 w-14 px-2 text-right text-sm font-mono tabular-nums bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <span class="text-xs font-medium text-muted-foreground w-20 pl-2 text-left">m</span>
          </div>
        </div>
        <Slider v-model="routeOffsetArray" :min="0" :max="10" :step="0.5" :disabled="parametersDisabled" class="py-1" />
      </div>

      <!-- 循环圈数 -->
      <div class="space-y-3 rounded-lg border border-border/40 bg-background/45 p-3">
        <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 group/item">
          <label for="run-loops"
            class="text-xs font-semibold text-muted-foreground group-hover/item:text-primary transition-colors">循环圈数</label>
          <div class="flex items-center gap-2">
            <input id="run-loops" v-model.number="loopCount" type="number" min="1" max="10" step="1" :disabled="parametersDisabled"
              class="h-8 w-14 px-2 text-right text-sm font-mono tabular-nums bg-background border border-border/40 rounded-md focus:border-primary/40 focus:ring-1 focus:ring-primary/20 outline-none transition-all disabled:opacity-50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
            <span class="text-xs font-medium text-muted-foreground w-20 pl-2 text-left">圈</span>
          </div>
        </div>
        <Slider v-model="loopCountArray" :min="1" :max="10" :step="1" :disabled="parametersDisabled" class="py-1" />
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
              <span>{{ pendingAction === 'start' ? '启动中…' : '开始任务' }}</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>开始同步位置至设备</TooltipContent>
        </Tooltip>

        <Tooltip v-if="isRunning">
          <TooltipTrigger asChild>
            <Button variant="secondary"
              class="flex-1 h-10 gap-2 text-sm font-semibold bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 border-amber-500/20 transition-all active:opacity-90"
              :disabled="isOperating || props.disabled" @click="pauseRun">
              <PauseIcon class="w-4 h-4 fill-current" />
              <span>{{ pendingAction === 'pause' ? '暂停中…' : '暂停' }}</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>暂停当前任务</TooltipContent>
        </Tooltip>

        <Tooltip v-if="isPaused">
          <TooltipTrigger asChild>
            <Button
              class="flex-1 h-10 gap-2 text-sm font-semibold shadow-lg shadow-primary/20 transition-all active:opacity-90"
              :disabled="isOperating || props.disabled" @click="resumeRun">
              <PlayIcon class="w-4 h-4 fill-current" />
              <span>{{ pendingAction === 'resume' ? '恢复中…' : '恢复' }}</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>恢复当前任务</TooltipContent>
        </Tooltip>

        <Tooltip v-if="isRunning || isPaused">
          <TooltipTrigger asChild>
            <Button variant="destructive"
              class="flex-1 h-10 gap-2 text-sm font-semibold shadow-lg shadow-destructive/20 transition-all active:opacity-90"
              :disabled="isOperating || props.disabled" @click="stopRun">
              <StopIcon class="w-4 h-4 fill-current" />
              <span>{{ pendingAction === 'stop' ? '停止中…' : '停止' }}</span>
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
  disabled?: boolean
}>()

const emit = defineEmits<{
  'position-update': [pos: { lat: number; lon: number }]
  'state-change': [state: RunningStatus['state']]
  completed: []
  'busy-change': [busy: boolean]
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
const STATUS_CHECK_INTERVAL_MS = 5000
const pendingAction = ref<'start' | 'pause' | 'resume' | 'stop' | null>(null)
const isOperating = computed(() => pendingAction.value !== null)
const parametersDisabled = computed(() => isRunning.value || isOperating.value || props.disabled || !status.value)
let statusInterval: ReturnType<typeof setInterval> | null = null
let statusRequest: Promise<void> | null = null
let lastStatusAt = 0

watch(isOperating, busy => emit('busy-change', busy), { flush: 'sync' })
let offPosition: (() => void) | null = null
let offCompleted: (() => void) | null = null
let offError: (() => void) | null = null

function applyStatus(data: RunningStatus) {
  if (disposed) return
  lastStatusAt = Date.now()
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
const canStart = computed(() =>
  status.value?.state === 'idle' && props.udid && props.routePoints.length >= 2 &&
  !isOperating.value && !props.disabled
)

watch(
  () => status.value?.state,
  state => {
    if (state) emit('state-change', state)
  },
  { immediate: true }
)

const statusText = computed(() => {
  if (pendingAction.value) {
    return { start: '启动中…', pause: '暂停中…', resume: '恢复中…', stop: '停止中…' }[pendingAction.value]
  }
  if (!status.value) return '同步中…'
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

async function runAction(
  action: NonNullable<typeof pendingAction.value>,
  errorMessage: string,
  request: () => Promise<void>
) {
  if (disposed || isOperating.value || props.disabled) return
  pendingAction.value = action
  // 操作前发出的查询不应覆盖操作后的状态。
  statusRevision++
  try {
    await request()
  } catch (e) {
    if (!disposed) showErrorDialog(`${errorMessage}: ${e instanceof Error ? e.message : '未知错误'}`)
  } finally {
    // 先等旧查询收尾，再读取操作完成后的状态，避免复用操作前的快照。
    await syncStatusAfterPending(true)
    pendingAction.value = null
  }
}

async function startRun() {
  if (!canStart.value) return
  const params = { ...paramsStore.params }
  const udid = props.udid
  const route = props.routePoints.map(point => ({ ...point }))
  await runAction('start', '启动跑步失败', async () => {
    await RunningService.SetLoopCount(params.loopCount)
    await RunningService.SetRandomization(params.speedVariance / 100, params.routeOffset)
    await RunningService.StartRun(udid, route, params.speed)
  })
}

async function pauseRun() {
  if (!isRunning.value) return
  await runAction('pause', '暂停失败', async () => {
    await RunningService.PauseRun()
  })
}

async function resumeRun() {
  if (!isPaused.value) return
  const params = { ...paramsStore.params }
  await runAction('resume', '恢复失败', async () => {
    // 暂停期间只修改本地设置，恢复时统一应用。
    await RunningService.SetSpeed(params.speed)
    await RunningService.SetLoopCount(params.loopCount)
    await RunningService.SetRandomization(params.speedVariance / 100, params.routeOffset)
    await RunningService.ResumeRun()
  })
}

async function stopRun() {
  if (!isRunning.value && !isPaused.value) return
  await runAction('stop', '停止失败', async () => {
    await RunningService.StopRun()
  })
}

function updateStatus(reportError = false): Promise<void> {
  if (disposed) return Promise.resolve()
  // 初始同步、兜底轮询和错误事件共享同一在途请求。
  if (statusRequest) return statusRequest
  const revision = statusRevision
  statusRequest = (async () => {
    try {
      const data = await RunningService.GetStatus()
      if (!disposed && revision === statusRevision) {
        status.value = data
        lastStatusAt = Date.now()
      }
    } catch (e) {
      // 后台兜底失败保持静默，避免反复弹出相同错误。
      if (reportError && !disposed && revision === statusRevision) {
        showErrorDialog(`获取状态失败: ${e instanceof Error ? e.message : '未知错误'}`)
      }
    }
  })().finally(() => {
    statusRequest = null
  })
  return statusRequest
}

function onVisibilityChange() {
  if (!document.hidden && !isOperating.value) void updateStatus()
}

async function syncStatusAfterPending(reportError = false) {
  if (statusRequest) await statusRequest
  if (!disposed) await updateStatus(reportError)
}

onMounted(() => {
  // 先订阅事件再拉取快照，通过 revision 丢弃晚到的旧查询。
  offPosition = Events.On('running:position', ev => {
    const data = ev.data
    applyStatus(data)
    emit('position-update', { lat: data.currentLat, lon: data.currentLon })
  })

  offCompleted = Events.On('running:completed', ev => {
    applyStatus(ev.data)
    emit('completed')
    showSuccess(`跑步任务已完成！共运行 ${ev.data.totalPoints} 个位置点`)
  })

  offError = Events.On('running:error', ev => {
    statusRevision++
    void syncStatusAfterPending()
    showErrorDialog(`跑步任务失败: ${ev.data}`)
  })

  void updateStatus(true)
  statusInterval = setInterval(() => {
    if (document.hidden || isOperating.value || Date.now() - lastStatusAt < STATUS_CHECK_INTERVAL_MS) return
    void updateStatus()
  }, STATUS_CHECK_INTERVAL_MS)
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onUnmounted(() => {
  disposed = true
  statusRevision++
  if (statusInterval !== null) clearInterval(statusInterval)
  document.removeEventListener('visibilitychange', onVisibilityChange)
  offPosition?.()
  offCompleted?.()
  offError?.()
})

</script>

<style scoped></style>
