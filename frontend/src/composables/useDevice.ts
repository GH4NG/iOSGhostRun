import { ref, computed } from 'vue'
import {
  GetDevices,
  SelectDevice as SelectDeviceAPI,
  ResetLocation as ResetLocationAPI,
  CheckDeveloperImage,
  MountDeveloperImage
} from '../../wailsjs/go/device/Manager'
import type { device } from '../../wailsjs/go/models'

export type Device = device.DeviceInfo

// 单例状态 - 在函数外部定义以便跨组件共享
const devices = ref<Device[]>([])
const selectedDevice = ref<Device | null>(null)
const loading = ref(false)

export function useDevice() {
  const isIOS17Device = computed(() => {
    if (!selectedDevice.value) return false
    return selectedDevice.value.supportsRsd
  })

  async function refreshDevices() {
    loading.value = true
    try {
      const result = await GetDevices()
      devices.value = result || []
    } finally {
      loading.value = false
    }
  }

  async function selectDevice(udid: string) {
    const device = devices.value.find(d => d.udid === udid)
    if (!device) {
      throw new Error('设备未找到')
    }

    await SelectDeviceAPI(udid)
    selectedDevice.value = device
  }

  async function checkAndMountDeveloperImage() {
    if (!selectedDevice.value) {
      throw new Error('请先选择设备')
    }

    // 先检查镜像状态
    const mounted = await CheckDeveloperImage()
    if (mounted) {
      return { message: '开发者镜像已就绪' }
    }

    // 需要挂载
    await MountDeveloperImage()
    return { message: '开发者镜像挂载成功' }
  }

  async function resetLocation() {
    if (!selectedDevice.value) {
      throw new Error('请先选择设备')
    }
    await ResetLocationAPI()
  }

  return {
    devices,
    selectedDevice,
    loading,
    isIOS17Device,
    refreshDevices,
    selectDevice,
    checkAndMountDeveloperImage,
    resetLocation
  }
}
