<template>
  <Card class="flex flex-col bg-card border-none shadow-none text-foreground">
    <div class="flex items-center justify-between p-5 border-b border-border/40">
      <div class="flex items-center gap-3">
        <div class="p-1.5 rounded-lg bg-primary/10">
          <PinFilledIcon class="w-5 h-5 text-primary" />
        </div>
        <span class="text-xs font-black uppercase tracking-widest text-foreground/80">路线管理 - 当前路线</span>
      </div>
      <div class="flex items-center gap-1">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon"
              class="w-8 h-8 rounded-full hover:bg-primary/10 hover:text-primary transition-all active:scale-90"
              @click="executeClearRoute">
              <PlusIcon class="w-4 h-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">新建路线</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon"
              class="w-8 h-8 rounded-full hover:bg-primary/10 hover:text-primary transition-all active:scale-90"
              @click="showImport = !showImport">
              <DownloadIcon class="w-4 h-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">导入路线</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon"
              class="w-8 h-8 rounded-full hover:bg-destructive/10 hover:text-destructive transition-all active:scale-90"
              :disabled="routePoints.length === 0" @click="executeClearRoute">
              <TrashIcon class="w-4 h-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">清空当前路线</TooltipContent>
        </Tooltip>
      </div>
    </div>

    <div class="p-6 flex flex-col gap-6">
      <!-- 导入路线区域 -->
      <div v-if="showImport"
        class="flex flex-col gap-3 p-4 bg-secondary/10 rounded-xl border border-border/40 animate-in fade-in slide-in-from-top-2">
        <div>
          <div class="text-[10px] font-black text-muted-foreground/50 uppercase tracking-widest mb-2">
            坐标系选择
          </div>
          <div class="relative">
            <Button variant="outline" size="sm"
              class="h-8 text-[10px] w-full justify-between bg-background border-border/40"
              @click="showCoordSystemMenu = !showCoordSystemMenu">
              <span class="text-xs">
                {{ coordSystemLabel }}
              </span>
              <span>▼</span>
            </Button>
            <div v-if="showCoordSystemMenu"
              class="absolute top-full left-0 right-0 mt-1 bg-background border border-border/40 rounded-lg shadow-lg z-50">
              <button @click="selectCoordSystem('wgs84')"
                class="w-full px-3 py-2 text-left text-[10px] hover:bg-primary/10 hover:text-primary transition-colors">
                WGS84(GPS标准)
              </button>
              <button @click="selectCoordSystem('gcj02')"
                class="w-full px-3 py-2 text-left text-[10px] hover:bg-primary/10 hover:text-primary transition-colors">
                GCJ-02(高德/腾讯)
              </button>
              <button @click="selectCoordSystem('bd09')"
                class="w-full px-3 py-2 text-left text-[10px] hover:bg-primary/10 hover:text-primary transition-colors last:rounded-b-lg">
                BD09(百度)
              </button>
            </div>
          </div>
        </div>
        <div>
          <div class="text-[10px] font-black text-muted-foreground/50 uppercase tracking-widest mb-2">
            粘贴路线数据 (JSON)
          </div>
          <textarea v-model="importText" placeholder='{"lng":"116.29","lat":"40.00"}, ...'
            class="min-h-[100px] p-3 text-[10px] font-mono bg-background border border-border/40 rounded-lg focus:outline-none focus:ring-1 focus:ring-primary/40 resize-none"></textarea>
        </div>
        <div class="flex gap-2 mt-2">
          <Button size="sm" class="flex-1 h-8 text-[10px] font-bold" @click="handleImport">确认导入</Button>
          <Button size="sm" variant="ghost" class="h-8 text-[10px] font-bold" @click="showImport = false">取消</Button>
        </div>
      </div>

      <!-- 保存路线 -->
      <div class="flex gap-2">
        <Input v-model="newRouteName" type="text" placeholder="给这条线路起个名字..."
          class="h-10 bg-secondary/20 border-border/40 focus:border-primary/40 focus:ring-primary/20 transition-all text-xs font-medium"
          @keydown.enter="saveCurrentRoute" />
        <Tooltip>
          <TooltipTrigger asChild>
            <Button size="icon"
              class="h-10 w-10 shrink-0 shadow-lg shadow-primary/20 transition-all hover:scale-105 active:scale-95"
              :disabled="!canSave" @click="saveCurrentRoute">
              <PlusIcon class="w-4 h-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>保存至本地数据库</TooltipContent>
        </Tooltip>
      </div>

      <!-- 已保存路线列表 -->
      <div class="flex flex-col gap-4">
        <div class="flex items-center justify-between px-1">
          <span class="text-[10px] font-black text-muted-foreground/50 uppercase tracking-[0.2em]">已保存列表</span>
          <Badge variant="outline"
            class="text-[9px] h-4 font-black bg-secondary/10 text-muted-foreground/60 border-none">{{ savedRoutes.length
            }}</Badge>
        </div>

        <div v-if="savedRoutes.length === 0"
          class="flex flex-col items-center justify-center py-10 border border-dashed border-border/30 rounded-2xl bg-secondary/5 text-center px-4">
          <PinFilledIcon class="w-8 h-8 text-muted-foreground/20 mb-3" />
          <span class="text-[10px] leading-relaxed text-muted-foreground/40 font-medium">还没有保存的路线</span>
        </div>

        <div v-else class="flex flex-col gap-2 max-h-[300px] overflow-y-auto no-scrollbar pr-1">
          <div v-for="route in savedRoutes" :key="route.name"
            class="flex items-center gap-3 p-4 bg-secondary/20 rounded-xl border border-transparent hover:border-primary/30 hover:bg-primary/5 transition-all duration-300 cursor-pointer group hover:shadow-lg hover:shadow-black/5"
            @click="loadRouteByName(route.name)">
            <div
              class="w-10 h-10 rounded-lg bg-background flex items-center justify-center border border-border/50 group-hover:border-primary/20 transition-colors">
              <RocketIcon class="w-5 h-5 text-primary/40 group-hover:text-primary/80 transition-colors" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-bold truncate group-hover:text-primary transition-colors tracking-tight">
                {{ route.name }}
              </div>
              <div class="text-[10px] uppercase font-black text-muted-foreground/40 mt-1 flex items-center gap-2">
                <span>{{ route.points.length }} <span class="font-bold">PTS</span></span>
                <span class="w-1 h-1 rounded-full bg-border"></span>
                <span>{{ formatDist(calculateDist(route.points)) }}</span>
              </div>
            </div>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon"
                  class="w-8 h-8 rounded-full text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-all opacity-0 group-hover:opacity-100"
                  @click.stop="deleteRouteByName(route.name)">
                  <TrashIcon class="w-4 h-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>删除路线</TooltipContent>
            </Tooltip>
          </div>
        </div>
      </div>
    </div>
  </Card>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { DrawingPinFilledIcon as PinFilledIcon, TrashIcon, PlusIcon, RocketIcon, DownloadIcon } from '@radix-icons/vue'
import {
  listRoutes,
  saveRoute,
  loadRoute,
  deleteRoute,
  calculateRouteDistance,
  formatDistance,
  type RoutePoint,
  type SavedRoute
} from '../lib/routeStorage'
import { GCJ02ToWGS84, WGS84ToGCJ02, BD09ToWGS84 } from '../lib/transform'
import { useNotification } from '../composables/useNotification'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const { showError: showErrorDialog, showSuccess } = useNotification()

const props = defineProps<{
  modelValue: RoutePoint[]
  currentLayerId?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [points: RoutePoint[]]
  'locating-point': [point: RoutePoint]
}>()

const newRouteName = ref('')
const savedRoutes = ref<SavedRoute[]>([])
const routePoints = computed(() => props.modelValue)

const showImport = ref(false)
const importText = ref('')
const importCoordSystem = ref<'wgs84' | 'gcj02' | 'bd09' | ''>('')
const showCoordSystemMenu = ref(false)

const canSave = computed(() => {
  return newRouteName.value.trim() && routePoints.value.length >= 2
})

const coordSystemLabel = computed(() => {
  const labels: Record<string, string> = {
    wgs84: 'WGS84',
    gcj02: 'GCJ-02',
    bd09: 'BD09'
  }
  return labels[importCoordSystem.value]
})

function selectCoordSystem(system: 'wgs84' | 'gcj02' | 'bd09') {
  importCoordSystem.value = system
  showCoordSystemMenu.value = false
}

function refreshRoutes() {
  savedRoutes.value = listRoutes()
}

function saveCurrentRoute() {
  if (!canSave.value) return

  saveRoute(newRouteName.value.trim(), routePoints.value)
  showSuccess(`路线 "${newRouteName.value.trim()}" 已保存`)
  newRouteName.value = ''
  refreshRoutes()
}

function loadRouteByName(name: string) {
  const points = loadRoute(name)
  if (points && points.length > 0) {
    emit('update:modelValue', points)
    emit('locating-point', points[0])
    showSuccess(`已加载路线 "${name}"，包含 ${points.length} 个位置点`)
  }
}

function deleteRouteByName(name: string) {
  deleteRoute(name)
  showSuccess(`路线 "${name}" 已删除`)
  refreshRoutes()
}

function executeClearRoute() {
  if (routePoints.value.length > 0) {
    emit('update:modelValue', [])
    showSuccess('已清空当前路线')
  } else {
    showSuccess('准备就绪，可以开始绘制新路线')
  }
}

// 导入路线数据
function handleImport() {
  const text = importText.value.trim()
  if (!text) return

  try {
    let points: RoutePoint[] = []

    let coordSystem = importCoordSystem.value

    const matches = text.match(/\{"lng":"[^"]+","lat":"[^"]+"\}/g)
    if (matches && matches.length > 0) {
      points = matches.map(m => {
        const p = JSON.parse(m)
        let lat = parseFloat(p.lat)
        let lon = parseFloat(p.lng)

        // 根据导入坐标系进行转换
        if (coordSystem === 'gcj02') {
          ;[lat, lon] = GCJ02ToWGS84(lat, lon)
        } else if (coordSystem === 'bd09') {
          ;[lat, lon] = BD09ToWGS84(lat, lon)
        }
        return { lat, lon }
      })
    } else {
      try {
        const parsed = JSON.parse(text)
        if (Array.isArray(parsed)) {
          points = parsed.map(p => {
            let lat = parseFloat(p.lat)
            let lon = parseFloat(p.lng || p.lon)

            // 根据导入坐标系进行转换
            if (coordSystem === 'gcj02') {
              ;[lat, lon] = GCJ02ToWGS84(lat, lon)
            } else if (coordSystem === 'bd09') {
              ;[lat, lon] = BD09ToWGS84(lat, lon)
            }
            return { lat, lon }
          })
        }
      } catch (e) {
        // Continue
      }
    }

    if (points.length > 0) {
      emit('update:modelValue', points)
      emit('locating-point', points[0])
      showImport.value = false
      importText.value = ''
      showSuccess(`已导入路线，包含 ${points.length} 个位置点`)
    } else {
      alert('无法识别路线数据格式，请确保格式正确。')
    }
  } catch (e) {
    showErrorDialog(`操作失败: ${e instanceof Error ? e.message : '未知错误'}`)
    alert('导入失败，请检查数据格式。')
  }
}

function calculateDist(points: RoutePoint[]) {
  return calculateRouteDistance(points)
}

function formatDist(km: number) {
  return formatDistance(km)
}

onMounted(() => {
  refreshRoutes()
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
