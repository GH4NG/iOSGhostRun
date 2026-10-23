<template>
    <Card :aria-busy="pending" class="manual-gps-panel panel-card flex flex-col gap-0 p-0 overflow-hidden font-sans">
        <div class="panel-header">
            <div class="flex min-w-0 items-center gap-2">
                <div class="panel-icon">
                    <Crosshair2Icon class="h-4 w-4 text-primary" />
                </div>
                <div class="min-w-0">
                    <span class="panel-title">手动控制</span>
                    <span class="panel-description">摇杆与键盘实时定位</span>
                </div>
            </div>
            <Badge :variant="active ? 'default' : 'secondary'"
                class="shrink-0 text-xs font-semibold px-2 py-0 h-5 rounded-full border-none shadow-sm shadow-black/5">
                {{ pending ? '操作中…' : active ? '控制中' : '待机' }}
            </Badge>
        </div>
        <div class="flex flex-col gap-3 bg-card/70 p-3">
            <p class="text-xs text-muted-foreground">拖动摇杆或按 WASD / 方向键，松手停步</p>
            <div ref="pad" role="group" aria-label="方向摇杆，上北下南左西右东" :tabindex="active && !pending ? 0 : -1"
                :aria-disabled="!active || pending" :class="{ 'opacity-50': !active || pending }"
                class="manual-joystick relative mx-auto shrink-0 rounded-full border border-primary/25 bg-primary/5 touch-none select-none outline-none focus-visible:ring-2 focus-visible:ring-primary"
                @pointerdown="onPointerDown" @pointermove="onPointerMove" @pointerup="onPointerEnd"
                @pointercancel="onPointerEnd" @lostpointercapture="onPointerEnd" @contextmenu.prevent>
                <span class="absolute top-1 left-1/2 -translate-x-1/2 text-xs text-muted-foreground">北</span>
                <span class="absolute bottom-1 left-1/2 -translate-x-1/2 text-xs text-muted-foreground">南</span>
                <span class="absolute left-2 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">西</span>
                <span class="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">东</span>
                <div class="pointer-events-none absolute left-1/2 top-1/2 h-10 w-10 rounded-full bg-primary shadow-lg shadow-primary/25"
                    :style="{
                        transform: `translate(calc(-50% + ${direction.east * 40}px), calc(-50% - ${direction.north * 40}px))`
                    }" />
            </div>
            <div class="manual-speed space-y-3">
                <div class="flex h-8 items-center justify-between gap-3">
                    <label for="manual-gps-speed" class="text-xs font-semibold text-muted-foreground">移动速度</label>
                    <div class="flex items-center gap-2">
                        <input id="manual-gps-speed" v-model.number="speed" type="number" min="1" max="20" step="0.5"
                            :disabled="pending"
                            class="[appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none h-8 w-16 rounded-md border border-border/40 bg-background px-2 text-right text-sm font-mono tabular-nums text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40" />
                        <span class="w-9 text-xs text-muted-foreground">km/h</span>
                    </div>
                </div>
                <Slider v-model="speedArray" :min="1" :max="20" :step="0.5" :disabled="pending" class="h-4"
                    aria-label="手动定位速度" />
            </div>
            <div class="border-t border-border/40 pt-3 text-[11px] leading-5 text-muted-foreground">
                <p>拖动地图，让中央定位点选定起点。</p>
                <p>开始后仅摇杆 / WASD 移动设备位置。</p>
                <p>松手 / 失焦停步，关闭恢复真实定位。</p>
            </div>
            <Button class="h-10 w-full gap-2 text-sm font-semibold" :variant="active ? 'destructive' : 'default'"
                :disabled="pending || (!active && (disabled || !udid))" @click="toggle">
                <StopIcon v-if="active" class="h-4 w-4" />
                <PlayIcon v-else class="h-4 w-4" />
                {{ pending ? '操作中…' : active ? '停止控制' : '开始控制' }}
            </Button>
        </div>
    </Card>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Crosshair2Icon, PlayIcon, StopIcon } from '@radix-icons/vue'
import { Events } from '@wailsio/runtime'
import { RunningService } from '../../bindings/iOSGhostRun/services'
import type { RoutePoint } from '../lib/routeUtils'
import { useNotification } from '../composables/useNotification'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Slider } from '@/components/ui/slider'
import { Badge } from '@/components/ui/badge'

const props = defineProps<{
    udid: string
    disabled?: boolean
    visible?: boolean
    getOrigin: () => RoutePoint | null
}>()
const emit = defineEmits<{ 'active-change': [active: boolean] }>()
const { showError } = useNotification()
const active = ref(false)
const pending = ref(false)
const speed = ref(5)
const speedArray = computed({
    get: () => [speed.value],
    set: value => {
        speed.value = value[0]
    }
})
const pad = ref<HTMLDivElement | null>(null)
const keys = ref(new Set<string>())
const pointer = ref({ east: 0, north: 0 })
let pointerId: number | null = null
let timer: ReturnType<typeof setTimeout> | null = null
let request: Promise<void> | null = null
let started = false
let disposed = false
let offPosition: (() => void) | null = null

const direction = computed(() => {
    if (pointerId !== null) return pointer.value
    const pressed = keys.value
    const east =
        Number(pressed.has('d') || pressed.has('arrowright')) -
        Number(pressed.has('a') || pressed.has('arrowleft'))
    const north =
        Number(pressed.has('w') || pressed.has('arrowup')) -
        Number(pressed.has('s') || pressed.has('arrowdown'))
    const length = Math.max(1, Math.hypot(east, north))
    return { east: east / length, north: north / length }
})

function setActive(value: boolean) {
    active.value = value
    emit('active-change', value)
}

function clearTimer() {
    if (timer !== null) clearTimeout(timer)
    timer = null
}

function clearInput() {
    keys.value = new Set()
    pointer.value = { east: 0, north: 0 }
    const id = pointerId
    pointerId = null
    if (id !== null && pad.value?.hasPointerCapture(id)) pad.value.releasePointerCapture(id)
}

async function sendInput() {
    if (!started || pending.value || disposed || request) return
    clearTimer()
    const value = direction.value
    const kmh = Number.isFinite(speed.value) ? Math.max(1, Math.min(20, speed.value)) : 5
    request = (async () => {
        try {
            await RunningService.SetManualDirection(value.east, value.north, kmh)
        } catch (error) {
            if (active.value && !pending.value && !disposed) {
                showError(`手动控制失败：${String(error)}`)
                // 由同一停止路径收尾，不能在当前 request 内等待自身。
                void stop()
            }
        }
    })()
    await request
    request = null
    if (started && active.value && !pending.value && !disposed) timer = setTimeout(sendInput, 200)
}

async function toggle() {
    if (active.value) return stop()
    if (pending.value || props.disabled || !props.udid || disposed) return
    const origin = props.getOrigin()
    if (!origin) {
        showError('地图尚未就绪，请稍后重试')
        return
    }
    pending.value = true
    setActive(true)
    try {
        await RunningService.StartManualControl(props.udid, origin)
        started = true
        if (disposed) {
            await RunningService.StopRun()
            started = false
            setActive(false)
        }
    } catch (error) {
        setActive(false)
        if (!disposed) showError(`开启手动定位失败：${String(error)}`)
    } finally {
        pending.value = false
        if (started && !disposed) void sendInput()
    }
}

async function stop() {
    if (pending.value) return
    pending.value = true
    clearTimer()
    clearInput()
    try {
        if (request) await request
        if (started) await RunningService.StopRun()
        started = false
        setActive(false)
    } catch (error) {
        if (!disposed) showError(`关闭手动定位失败：${String(error)}`)
    } finally {
        pending.value = false
        if (started && !disposed) void sendInput()
    }
}

const controlKeys = new Set(['w', 'a', 's', 'd', 'arrowup', 'arrowdown', 'arrowleft', 'arrowright'])
function isEditing(target: EventTarget | null) {
    return (
        target instanceof Element &&
        !!target.closest(
            'input, textarea, select, [contenteditable]:not([contenteditable="false"]), [role="dialog"], [role="alertdialog"], [role="combobox"], [role="menu"], [role="slider"]'
        )
    )
}
function onKeyDown(event: KeyboardEvent) {
    if (!active.value || pending.value || props.visible === false) return
    if (event.key === 'Escape' && !isEditing(event.target)) {
        event.preventDefault()
        void stop()
        return
    }
    if (
        event.altKey ||
        event.ctrlKey ||
        event.metaKey ||
        event.isComposing ||
        isEditing(event.target)
    )
        return
    const key = event.key.toLowerCase()
    if (!controlKeys.has(key)) return
    event.preventDefault()
    event.stopPropagation()
    keys.value = new Set(keys.value).add(key)
    if (!event.repeat) void sendInput()
}
function onKeyUp(event: KeyboardEvent) {
    const key = event.key.toLowerCase()
    if (!keys.value.has(key)) return
    event.preventDefault()
    event.stopPropagation()
    const next = new Set(keys.value)
    next.delete(key)
    keys.value = next
    void sendInput()
}
function halt() {
    clearInput()
    void sendInput()
}
function onFocusIn(event: FocusEvent) {
    if (isEditing(event.target)) halt()
}
function onVisibilityChange() {
    if (document.hidden) halt()
}
function onPointerMove(event: PointerEvent) {
    if (event.pointerId !== pointerId || !pad.value) return
    const rect = pad.value.getBoundingClientRect()
    let east = (event.clientX - rect.left - rect.width / 2) / 40
    let north = -(event.clientY - rect.top - rect.height / 2) / 40
    const length = Math.hypot(east, north)
    if (length < 0.1) east = north = 0
    else if (length > 1) {
        east /= length
        north /= length
    }
    pointer.value = { east, north }
    void sendInput()
}
function onPointerDown(event: PointerEvent) {
    if (
        !active.value ||
        pending.value ||
        props.visible === false ||
        pointerId !== null ||
        event.button !== 0
    )
        return
    event.preventDefault()
    keys.value = new Set()
    pointerId = event.pointerId
    pad.value?.focus()
    pad.value?.setPointerCapture(event.pointerId)
    onPointerMove(event)
}
function onPointerEnd(event: PointerEvent) {
    if (event.pointerId === pointerId) halt()
}

watch(
    () => props.visible,
    visible => {
        if (visible === false) halt()
    }
)
watch(
    () => props.udid,
    () => {
        if (active.value) void stop()
    }
)
onMounted(() => {

    window.addEventListener('keydown', onKeyDown, true)
    window.addEventListener('keyup', onKeyUp, true)
    window.addEventListener('blur', halt)
    document.addEventListener('focusin', onFocusIn)
    document.addEventListener('visibilitychange', onVisibilityChange)
    offPosition = Events.On('running:position', event => {
        if (active.value && !pending.value && event.data.state === 'idle') {
            started = false
            clearTimer()
            clearInput()
            setActive(false)
        }
    })
})
onUnmounted(() => {
    disposed = true
    clearTimer()
    clearInput()
    window.removeEventListener('keydown', onKeyDown, true)
    window.removeEventListener('keyup', onKeyUp, true)
    window.removeEventListener('blur', halt)
    document.removeEventListener('focusin', onFocusIn)
    document.removeEventListener('visibilitychange', onVisibilityChange)
    offPosition?.()
    // 启动中的请求由 toggle 收尾；已建立的会话走 StopRun 集中清理。
    if (!pending.value) void stop()
})
</script>

<style scoped>
.manual-gps-panel {
    width: 100%;
    min-width: 0;
}

/* 摇杆是固定交互区域，不能被卡片的纵向 flex 布局压缩。 */
.manual-joystick {
    width: 128px;
    height: 128px;
    min-height: 128px;
    flex: none;
}

.manual-joystick {
    margin-block: 4px;
}
</style>
