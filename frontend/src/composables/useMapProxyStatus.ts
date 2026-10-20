import { onMounted, onUnmounted, ref } from 'vue'
import { SettingsService } from '../../bindings/iOSGhostRun/services'
import type { MapProxyStatus } from '../../bindings/iOSGhostRun/services/models'

const status = ref<MapProxyStatus | null>(null)
const error = ref('')
const restarting = ref(false)
let pendingRefresh: Promise<void> | null = null
let timer: ReturnType<typeof setInterval> | undefined
let consumers = 0

function errorMessage(value: unknown): string {
    return value instanceof Error ? value.message : String(value)
}

async function refresh() {
    if (pendingRefresh) return pendingRefresh
    pendingRefresh = (async () => {
        try {
            status.value = await SettingsService.GetMapProxyStatus()
            error.value = ''
        } catch (value) {
            error.value = errorMessage(value)
        }
    })().finally(() => {
        pendingRefresh = null
    })
    return pendingRefresh
}

async function restart() {
    if (restarting.value) return false
    restarting.value = true
    try {
        // 等待正在进行的读取，避免旧状态覆盖重启结果。
        if (pendingRefresh) await pendingRefresh
        status.value = await SettingsService.RestartMapProxy()
        error.value = ''
        return status.value.running
    } catch (value) {
        error.value = errorMessage(value)
        return false
    } finally {
        restarting.value = false
    }
}

export function useMapProxyStatus() {
    onMounted(() => {
        if (++consumers === 1)
            timer = setInterval(() => {
                if (!restarting.value) void refresh()
            }, 3000)
        void refresh()
    })
    onUnmounted(() => {
        if (--consumers === 0) {
            clearInterval(timer)
            timer = undefined
        }
    })
    return { status, error, restarting, refresh, restart }
}
