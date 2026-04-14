<template>
  <Card class="flex flex-col overflow-hidden border-border/60 bg-card/90 shadow-xl shadow-black/5 ring-1 ring-white/5">
    <div class="relative flex items-center justify-between p-5 border-b border-border/40 bg-card/95">
      <div class="flex items-center gap-3">
        <div class="p-2 rounded-2xl bg-primary/10 border border-primary/15 shadow-inner shadow-primary/10">
          <MobileIcon class="w-5 h-5 text-primary" />
        </div>
        <div class="min-w-0">
          <span class="block text-xs font-black uppercase tracking-widest text-foreground/85">设备管理</span>
          <span class="block text-[10px] font-bold text-muted-foreground/55 mt-0.5">USB 连接与镜像状态</span>
        </div>
      </div>
      <Badge v-if="devices.length > 0" variant="outline"
        class="mr-1 h-5 rounded-full border-primary/20 bg-primary/10 px-2 text-[9px] font-black text-primary">
        {{ devices.length }} 台</Badge>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon"
            class="w-8 h-8 rounded-full border border-transparent hover:border-primary/20 hover:bg-primary/10 hover:text-primary transition-all active:scale-90"
            @click="refreshDevices" :disabled="loading">
            <UpdateIcon class="w-4 h-4" :class="{ 'animate-spin': loading }" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="right">刷新设备列表</TooltipContent>
      </Tooltip>
    </div>

    <div v-if="devices.length === 0"
      class="flex flex-col items-center justify-center py-11 px-5 text-center bg-secondary/5">
      <div
        class="mb-4 flex h-14 w-14 items-center justify-center rounded-3xl border border-dashed border-border/60 bg-background/70">
        <MobileIcon class="h-7 w-7 text-muted-foreground/30" />
      </div>
      <p class="text-sm font-bold text-foreground/75">{{ loading ? '正在检测设备...' : '未检测到 iOS 设备' }}</p>
      <p class="text-xs leading-relaxed text-muted-foreground/55 mt-1.5">请确保设备已通过 USB 连接并信任此电脑</p>
    </div>

    <div v-else class="flex flex-col gap-3 p-4 overflow-y-auto max-h-[300px] no-scrollbar bg-card/70">
      <div v-for="device in devices" :key="device.UDID"
        class="flex flex-col p-4 rounded-2xl border cursor-pointer transition-all duration-300 relative group overflow-hidden"
        :class="[
          selectedUdid === device.UDID
            ? 'bg-primary/10 border-primary/35 shadow-lg shadow-primary/10'
            : 'bg-secondary/20 border-transparent hover:bg-secondary/40 hover:border-border/60 hover:shadow-md hover:shadow-black/5'
        ]" @click="selectDevice(device)">
        <div v-if="selectedUdid === device.UDID" class="absolute inset-y-3 left-0 w-1 rounded-r-full bg-primary"></div>
        <div class="flex items-center gap-4">
          <div
            class="w-11 h-11 flex items-center justify-center bg-background/80 rounded-2xl border border-border/50 group-hover:border-primary/30 group-hover:shadow-lg transition-all duration-300">
            <MobileIcon class="w-6 h-6 text-primary/80" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-sm font-bold tracking-tight truncate">
              {{ device.DeviceName || '未知设备' }}
            </div>
            <div class="text-[10px] uppercase font-bold text-muted-foreground/60 mt-1 flex items-center gap-2">
              <span>{{ device.ProductType }}</span>
              <span class="w-1 h-1 rounded-full bg-border"></span>
              <span>iOS {{ device.ProductVersion }}</span>
            </div>
          </div>
        </div>
        <div v-if="selectedUdid === device.UDID"
          class="flex gap-2 mt-4 pl-1 animate-in fade-in slide-in-from-top-2 duration-500">
          <Badge variant="outline"
            class="bg-primary/20 text-primary border-primary/30 text-[9px] font-black uppercase py-0 h-5 rounded-full px-2 tracking-tighter">
            已选择</Badge>
          <Badge v-if="imageMounted" variant="outline"
            class="bg-emerald-500/20 text-emerald-500 border-emerald-500/30 text-[9px] font-black uppercase py-0 h-5 rounded-full px-2 tracking-tighter">
            镜像已就绪</Badge>
        </div>
      </div>
    </div>
  </Card>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
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
}>()

const emit = defineEmits<{
  'update:modelValue': [udid: string]
}>()

const devices = ref<ServiceDeviceInfo[]>([])
const loading = ref(false)
const selectedUdid = ref(props.modelValue)
const imageMounted = ref(false)

async function refreshDevices() {
  loading.value = true
  try {
    const result = await DevicesService.ListDevices()
    devices.value = result || []

    // 如果当前选中的设备不在列表中，清除选择
    if (!devices.value.find(d => d.UDID === selectedUdid.value)) {
      selectedUdid.value = ''
      emit('update:modelValue', '')
    }

    if (devices.value.length > 0) {
      showSuccess(`检测到 ${devices.value.length} 个 iOS 设备`)
    } else {
      showInfo('未检测到 iOS 设备，请确保设备已通过USB连接')
    }
  } catch (e) {
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
  try {
    await DevicesService.SelectDevice(device.UDID)
    selectedUdid.value = device.UDID
    emit('update:modelValue', device.UDID)
    showSuccess(`已选择设备: ${device.DeviceName || '未知设备'}`)
  } catch (e) {
    showErrorDialog(`操作失败: ${e instanceof Error ? e.message : '未知错误'}`)
  }
}

onMounted(() => {
  refreshDevices()
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
