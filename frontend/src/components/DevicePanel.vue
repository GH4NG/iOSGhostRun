<template>
  <Card :aria-busy="isBusy" class="panel-card flex flex-col gap-0 p-0 overflow-hidden font-sans">
    <div class="panel-header">
      <div class="flex min-w-0 items-center gap-2">
        <div class="panel-icon">
          <MobileIcon class="w-4 h-4 text-primary" />
        </div>
        <div class="min-w-0">
          <span class="panel-title">设备管理</span>
          <span class="panel-description">USB 连接与镜像状态</span>
        </div>
      </div>
      <Badge v-if="devices.length > 0" variant="outline"
        class="shrink-0 h-5 rounded-full border-primary/35 bg-primary/15 px-2.5 text-xs font-semibold text-primary">
        {{ devices.length }} 台</Badge>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon" aria-label="刷新设备列表"
            class="w-8 h-8 shrink-0 rounded-lg border border-transparent hover:border-primary/20 hover:bg-primary/10 hover:text-primary transition-all active:scale-90"
            @click="refreshDevices" :disabled="isBusy || props.disabled">
            <UpdateIcon class="w-4 h-4" :class="{ 'animate-spin': loading }" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="right">刷新设备列表</TooltipContent>
      </Tooltip>
    </div>

    <div v-if="devices.length === 0"
      class="flex flex-col items-center justify-center py-7 px-4 text-center bg-secondary/5">
      <div
        class="mb-4 flex h-14 w-14 items-center justify-center rounded-3xl border border-dashed border-border/60 bg-background/70">
        <MobileIcon class="h-7 w-7 text-muted-foreground/30" />
      </div>
      <p class="text-sm font-semibold text-foreground/90">{{ loading ? '正在检测设备...' : '未检测到 iOS 设备' }}</p>
      <p class="text-xs leading-relaxed text-muted-foreground mt-1.5">请确保设备已通过 USB 连接并信任此电脑</p>
    </div>

    <div v-else class="flex flex-col gap-3 p-3 overflow-y-auto max-h-[300px] no-scrollbar bg-card/70">
      <button v-for="device in devices" :key="device.UDID" type="button"
        :disabled="isBusy || props.disabled"
        class="flex flex-col p-3 rounded-xl border text-left cursor-pointer disabled:cursor-wait disabled:opacity-60 transition-all duration-300 relative group overflow-hidden"
        :class="[
          selectedUdid === device.UDID
            ? 'bg-primary/10 border-primary/35 shadow-lg shadow-primary/10'
            : 'bg-secondary/20 border-transparent hover:bg-secondary/40 hover:border-border/60 hover:shadow-md hover:shadow-black/5'
        ]" @click="selectDevice(device)">
        <div v-if="selectedUdid === device.UDID" class="absolute inset-y-3 left-0 w-1 rounded-r-full bg-primary"></div>
        <div class="flex items-center gap-3">
          <div
            class="w-10 h-10 shrink-0 flex items-center justify-center bg-background/80 rounded-lg border border-border/50 group-hover:border-primary/30 group-hover:shadow-lg transition-all duration-300">
            <MobileIcon class="w-6 h-6 text-primary/80" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-sm font-semibold tracking-tight truncate">
              {{ device.DeviceName || '未知设备' }}
            </div>
            <div class="text-xs font-medium text-foreground/75 mt-1 flex flex-wrap items-center gap-x-2 gap-y-1">
              <span>{{ device.ProductType }}</span>
              <span class="w-1 h-1 rounded-full bg-border"></span>
              <span>iOS {{ device.ProductVersion }}</span>
            </div>
          </div>
        </div>
        <p v-if="selectingUdid === device.UDID" role="status" class="mt-3 text-xs text-primary">
          正在准备设备与开发者镜像…
        </p>
        <div v-if="selectedUdid === device.UDID"
          class="flex gap-2 mt-4 pl-1 animate-in fade-in slide-in-from-top-2 duration-500">
          <Badge variant="outline"
            class="bg-primary/15 text-primary border-primary/40 text-xs font-semibold py-0 h-6 rounded-full px-2.5">
            已选择</Badge>
          <Badge v-if="imageMounted" variant="outline"
            class="bg-emerald-500/15 text-emerald-800 dark:text-emerald-200 border-emerald-700/40 dark:border-emerald-300/40 text-xs font-semibold py-0 h-6 rounded-full px-2.5">
            镜像已就绪</Badge>
        </div>
      </button>
    </div>
  </Card>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { MobileIcon, UpdateIcon } from '@radix-icons/vue'
import { DevicesService } from '../../bindings/iOSGhostRun/services'
import type { DeviceInfo as ServiceDeviceInfo } from '../../bindings/iOSGhostRun/services/models'
import { useNotification } from '../composables/useNotification'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const { showError: showErrorDialog, showSuccess, showInfo, showWarning } = useNotification()

const props = defineProps<{
  modelValue: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [udid: string]
  'busy-change': [busy: boolean]
}>()

const devices = ref<ServiceDeviceInfo[]>([])
const loading = ref(false)
const selectingUdid = ref('')
const isBusy = computed(() => loading.value || selectingUdid.value !== '')
let disposed = false

watch(isBusy, busy => emit('busy-change', busy), { flush: 'sync' })
watch(() => props.disabled, disabled => {
    if (!disabled && devices.value.length === 0) void refreshDevices()
})
const selectedUdid = ref(props.modelValue)
const imageMounted = ref(false)

async function refreshDevices() {
  if (disposed || isBusy.value || props.disabled) return
  loading.value = true
  try {
    const result = await DevicesService.ListDevices()
    if (disposed) return
    devices.value = result || []

    // 如果当前选中的设备不在列表中，清除选择
    if (!devices.value.find(d => d.UDID === selectedUdid.value)) {
      selectedUdid.value = ''
      imageMounted.value = false
      emit('update:modelValue', '')
    }

    if (devices.value.length > 0) {
      showSuccess(`检测到 ${devices.value.length} 个 iOS 设备`)
    } else {
      showInfo('未检测到 iOS 设备，请确保设备已通过USB连接')
    }
  } catch (e) {
    if (disposed) return
    const message = e instanceof Error ? e.message : '未知错误'
    const lowerMessage = message.toLowerCase()

    if (
      message.includes('信任此电脑') ||
      lowerMessage.includes('invalidhostid') ||
      lowerMessage.includes('invalid host id')
    ) {
      showWarning(message, 6000)
    } else {
      showErrorDialog(`操作失败: ${message}`)
    }
    devices.value = []
  } finally {
    loading.value = false
  }
}

async function selectDevice(device: ServiceDeviceInfo) {
  if (disposed || isBusy.value || props.disabled ||
    (selectedUdid.value === device.UDID && imageMounted.value)) return

  selectingUdid.value = device.UDID
  try {
    await DevicesService.SelectDevice(device.UDID)
    if (disposed) return
    selectedUdid.value = device.UDID
    imageMounted.value = true
    emit('update:modelValue', device.UDID)
    showSuccess(`已选择设备: ${device.DeviceName || '未知设备'}`)
  } catch (e) {
    if (disposed) return
    showErrorDialog(`操作失败: ${e instanceof Error ? e.message : '未知错误'}`)
  } finally {
    selectingUdid.value = ''
  }
}

onMounted(() => {
  refreshDevices()
})

onUnmounted(() => {
  disposed = true
})
</script>

<style scoped>
.no-scrollbar::-webkit-scrollbar {
  display: none;
}

.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
