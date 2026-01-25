<template>
  <Card class="flex flex-col bg-card border-border shadow-sm">
    <div class="flex items-center justify-between p-5 border-b border-border/40">
      <div class="flex items-center gap-3">
        <div class="p-1.5 rounded-lg bg-primary/10">
          <MobileIcon class="w-5 h-5 text-primary" />
        </div>
        <span class="text-xs font-black uppercase tracking-widest text-foreground/80">设备管理</span>
      </div>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon"
            class="w-8 h-8 rounded-full hover:bg-primary/10 hover:text-primary transition-all active:scale-90"
            @click="refreshDevices" :disabled="loading">
            <UpdateIcon class="w-4 h-4" :class="{ 'animate-spin': loading }" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="right">刷新设备列表</TooltipContent>
      </Tooltip>
    </div>

    <div v-if="devices.length === 0" class="flex flex-col items-center justify-center py-10 px-4 text-center">
      <p class="text-sm text-muted-foreground">{{ loading ? '正在检测设备...' : '未检测到 iOS 设备' }}</p>
      <p class="text-xs text-muted-foreground/60 mt-1">请确保设备已通过 USB 连接并信任此电脑</p>
    </div>

    <div v-else class="flex flex-col gap-2.5 p-4 overflow-y-auto max-h-[300px] no-scrollbar">
      <div v-for="device in devices" :key="device.UDID"
        class="flex flex-col p-4 rounded-xl border border-transparent cursor-pointer transition-all duration-300 relative group"
        :class="[
          selectedUdid === device.UDID
            ? 'bg-primary/5 border-primary/30 shadow-sm'
            : 'bg-secondary/20 hover:bg-secondary/40'
        ]" @click="selectDevice(device)">
        <div class="flex items-center gap-4">
          <div
            class="w-11 h-11 flex items-center justify-center bg-background rounded-xl border border-border/50 group-hover:border-primary/30 group-hover:shadow-lg transition-all duration-300">
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
            class="bg-primary/20 text-primary border-primary/30 text-[9px] font-black uppercase py-0 h-4 rounded-sm tracking-tighter">
            已选择</Badge>
          <Badge v-if="imageMounted" variant="outline"
            class="bg-emerald-500/20 text-emerald-500 border-emerald-500/30 text-[9px] font-black uppercase py-0 h-4 rounded-sm tracking-tighter">
            镜像已就绪</Badge>
        </div>
      </div>
    </div>
  </Card>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { MobileIcon, UpdateIcon } from '@radix-icons/vue'
import { DevicesService, ImageService } from '../../bindings/iOSGhostRun/services'
import type { DeviceInfo as ServiceDeviceInfo } from '../../bindings/iOSGhostRun/services/models'
import { useNotification } from '../composables/useNotification'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const { showError: showErrorDialog, showSuccess, showInfo } = useNotification()

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
      showInfo('未检测到 iOS 设备，请确保设备已连接')
    }
  } catch (e) {
    showErrorDialog(`操作失败: ${e instanceof Error ? e.message : '未知错误'}`)
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
    try {
      const mounted = await ImageService.CheckDeveloperImage(device.UDID)
      imageMounted.value = !!mounted
      if (imageMounted.value) {
        showInfo('开发者镜像已就绪，可以开始跑步任务')
      }
    } catch (e) {
      showErrorDialog(`操作失败: ${e instanceof Error ? e.message : '未知错误'}`)
      imageMounted.value = false
    }
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
