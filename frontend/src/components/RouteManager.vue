<template>
  <Card class="flex flex-col bg-card border-none shadow-none text-foreground">
    <div class="flex items-center justify-between p-5 border-b border-border/40">
      <div class="flex items-center gap-3">
        <div class="p-1.5 rounded-lg bg-primary/10">
          <PinFilledIcon class="w-5 h-5 text-primary" />
        </div>
        <span class="text-xs font-black uppercase tracking-widest text-foreground/80">路线管理</span>
      </div>
      <div class="flex items-center gap-1">
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

        <AlertDialog v-model:open="showClearConfirm">
          <AlertDialogTrigger asChild>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon"
                  class="w-8 h-8 rounded-full hover:bg-destructive/10 hover:text-destructive transition-all active:scale-90"
                  :disabled="routePoints.length === 0">
                  <TrashIcon class="w-4 h-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="left">清空当前路线</TooltipContent>
            </Tooltip>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>确定清空当前路线吗？</AlertDialogTitle>
              <AlertDialogDescription>此操作将清除地图上所有已绘制的点，且无法撤销。</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>取消</AlertDialogCancel>
              <AlertDialogAction @click="executeClearRoute" class="bg-destructive text-white hover:bg-destructive/90">
                确认清空</AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>

    <div class="p-6 flex flex-col gap-6">
      <!-- 导入路线区域 -->
      <div v-if="showImport"
        class="flex flex-col gap-2 p-4 bg-secondary/10 rounded-xl border border-border/40 animate-in fade-in slide-in-from-top-2">
        <div class="text-[10px] font-black text-muted-foreground/50 uppercase tracking-widest mb-1">
          粘贴路线数据 (JSON)
        </div>
        <textarea v-model="importText" placeholder='{"lng":"116.29","lat":"40.00"}, ...'
          class="min-h-[100px] p-3 text-[10px] font-mono bg-background border border-border/40 rounded-lg focus:outline-none focus:ring-1 focus:ring-primary/40 resize-none"></textarea>
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
          <span
            class="text-[10px] leading-relaxed text-muted-foreground/40 font-medium">还没有保存的路线</span>
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
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="ghost" size="icon"
                      class="w-8 h-8 rounded-full text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-all opacity-0 group-hover:opacity-100"
                      @click.stop>
                      <TrashIcon class="w-4 h-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>删除路线</TooltipContent>
                </Tooltip>
              </AlertDialogTrigger>
              <AlertDialogContent @click.stop>
                <AlertDialogHeader>
                  <AlertDialogTitle>确定删除路线 "{{ route.name }}" 吗？</AlertDialogTitle>
                  <AlertDialogDescription>此操作不可恢复。</AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction @click="deleteRouteByName(route.name)"
                    class="bg-destructive text-white hover:bg-destructive/90">确认删除</AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
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
} from '../utils/routeStorage'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger
} from '@/components/ui/alert-dialog'

const props = defineProps<{
  modelValue: RoutePoint[]
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
const showClearConfirm = ref(false)

const canSave = computed(() => {
  return newRouteName.value.trim() && routePoints.value.length >= 2
})

function refreshRoutes() {
  savedRoutes.value = listRoutes()
}

function saveCurrentRoute() {
  if (!canSave.value) return

  saveRoute(newRouteName.value.trim(), routePoints.value)
  newRouteName.value = ''
  refreshRoutes()
}

function loadRouteByName(name: string) {
  const points = loadRoute(name)
  if (points && points.length > 0) {
    emit('update:modelValue', points)
    emit('locating-point', points[0])
  }
}

function deleteRouteByName(name: string) {
  deleteRoute(name)
  refreshRoutes()
}

function executeClearRoute() {
  emit('update:modelValue', [])
}

// 导入路线数据
function handleImport() {
  const text = importText.value.trim()
  if (!text) return

  try {
    let points: RoutePoint[] = []

    const matches = text.match(/\{"lng":"[^"]+","lat":"[^"]+"\}/g)
    if (matches && matches.length > 0) {
      points = matches.map(m => {
        const p = JSON.parse(m)
        return {
          lon: parseFloat(p.lng),
          lat: parseFloat(p.lat)
        }
      })
    } else {
      try {
        const parsed = JSON.parse(text)
        if (Array.isArray(parsed)) {
          points = parsed.map(p => ({
            lon: parseFloat(p.lng || p.lon),
            lat: parseFloat(p.lat)
          }))
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
    } else {
      alert('无法识别路线数据格式，请确保格式正确。')
    }
  } catch (e) {
    console.error('Import failed:', e)
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
