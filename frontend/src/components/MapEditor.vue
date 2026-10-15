<template>
  <div class="map-editor relative w-full h-full bg-secondary/5 overflow-hidden text-foreground font-sans">
    <div class="map-overlays">
      <div class="map-toolbar">
        <!-- 搜索栏-->
        <div class="map-search" :class="{ 'is-expanded': isSearchExpanded }">
          <div
            class="relative flex flex-col items-start gap-3 pointer-events-auto w-full">
            <Tooltip>
              <TooltipTrigger asChild>
                <div
                  class="flex items-center gap-1 p-0 shadow-lg shadow-black/10 border-border/60 bg-card/80 backdrop-blur-md ring-1 ring-white/10 group rounded-xl overflow-hidden transition-all duration-500 ease-[cubic-bezier(0.19,1,0.22,1)]"
                  :class="[
                    isSearchExpanded
                      ? 'w-full h-10 px-1'
                      : 'h-10 w-10 justify-center ring-0 text-primary !bg-card/85'
                  ]">
                  <Button size="icon" variant="ghost" aria-label="搜索地点" :aria-expanded="isSearchExpanded"
                    class="h-8 w-8 shrink-0 rounded-lg text-primary hover:bg-primary/10 transition-all"
                    @click="isSearchExpanded = !isSearchExpanded">
                    <MagnifyingGlassIcon class="w-4 h-4" />
                  </Button>
                  <Input v-if="isSearchExpanded" v-model="searchQuery" type="text" placeholder="搜索地点…"
                    class="h-9 min-w-0 border-none bg-transparent focus-visible:ring-0 text-sm font-medium tracking-normal placeholder:text-muted-foreground/55 animate-in fade-in slide-in-from-left-2 duration-500"
                    @keydown.enter="searchLocation" autofocus />
                </div>
              </TooltipTrigger>
              <TooltipContent v-if="!isSearchExpanded" side="right">搜索地点</TooltipContent>
            </Tooltip>

            <!-- 搜索结果列表 -->
            <Transition enter-active-class="transition duration-300 ease-out"
              enter-from-class="opacity-0 -translate-y-2 scale-95" enter-to-class="opacity-100 translate-y-0 scale-100"
              leave-active-class="transition duration-200 ease-in" leave-from-class="opacity-100 translate-y-0 scale-100"
              leave-to-class="opacity-0 -translate-y-2 scale-95">
              <Card v-if="isSearchExpanded && searchResults.length"
                class="map-search-results app-scrollbar absolute top-full mt-2 w-full max-h-64 overflow-y-auto z-30 shadow-lg shadow-black/10 border-border/60 bg-card/95 backdrop-blur-xl ring-1 ring-white/10 p-1.5 flex flex-col gap-1">
                <div v-for="(result, idx) in searchResults" :key="idx"
                  class="px-4 py-3 text-xs font-medium cursor-pointer rounded-xl transition-colors hover:bg-primary/10 hover:text-primary leading-normal"
                  @click="selectSearchResult(result)">
                  {{ result.display_name }}
                </div>
              </Card>
            </Transition>
          </div>
        </div>

        <!-- 绘制模式 -->
        <div class="map-tools relative pointer-events-auto flex shrink-0 gap-2">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="outline" size="icon"
                class="h-10 w-10 shadow-lg shadow-black/10 border-border/60 bg-card/85 backdrop-blur-md ring-1 ring-white/10 transition-all duration-300"
                :class="[
                  isDrawingMode
                    ? 'bg-primary text-primary-foreground border-primary hover:bg-primary/90 shadow-primary/30'
                    : 'text-muted-foreground hover:bg-accent'
                ]" :disabled="disabled" :aria-label="isDrawingMode ? '停止绘制' : '开启路径绘制模式'"
                :aria-pressed="isDrawingMode" @click="isDrawingMode = !isDrawingMode">
                <Pencil1Icon class="w-4 h-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">{{ isDrawingMode ? '停止绘制' : '开启路径绘制模式' }}</TooltipContent>
          </Tooltip>
          <Tooltip v-if="isDrawingMode">
            <TooltipTrigger asChild>
              <Button variant="outline" size="icon" aria-label="撤回上一步" :disabled="!canUndo"
                class="h-10 w-10 shadow-lg shadow-black/10 border-border/60 bg-card/85 backdrop-blur-md ring-1 ring-white/10 text-muted-foreground hover:bg-accent transition-all duration-300"
                @click="undoLastPoint">
                <ResetIcon class="w-4 h-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom" :side-offset="8">撤回上一步</TooltipContent>
          </Tooltip>

          <!-- 图层切换 -->
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="outline" size="icon" aria-label="切换地图图层" :aria-expanded="!isLayerSwitcherCollapsed"
                class="h-10 w-10 shadow-lg shadow-black/10 border-border/60 bg-card/85 backdrop-blur-md ring-1 ring-white/10 transition-all duration-300"
                :class="[
                  !isLayerSwitcherCollapsed
                    ? 'bg-primary text-primary-foreground border-primary shadow-primary/30'
                    : 'text-muted-foreground hover:bg-accent'
                ]" @click="isLayerSwitcherCollapsed = !isLayerSwitcherCollapsed">
                <LayersIcon class="w-4 h-4" :class="{ 'text-primary': isLayerSwitcherCollapsed }" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom" align="end">切换地图图层</TooltipContent>
          </Tooltip>

          <Transition enter-active-class="transition duration-300 ease-out"
            enter-from-class="opacity-0 translate-x-4 scale-95" enter-to-class="opacity-100 translate-x-0 scale-100"
            leave-active-class="transition duration-200 ease-in" leave-from-class="opacity-100 translate-x-0 scale-100"
            leave-to-class="opacity-0 translate-x-4 scale-95">
            <Card v-show="!isLayerSwitcherCollapsed"
              class="absolute top-full right-0 mt-2 z-30 w-48 overflow-hidden shadow-lg shadow-black/10 border-border/60 bg-card/95 backdrop-blur-xl ring-1 ring-white/10 p-1.5 flex flex-col gap-1">
              <div v-for="layer in availableLayers" :key="layer.id"
                class="flex items-center gap-3 px-3 py-2.5 rounded-xl cursor-pointer transition-all text-xs font-semibold"
                :class="[
                  currentLayerId === layer.id
                    ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20'
                    : 'text-muted-foreground hover:bg-primary/10 hover:text-primary'
                ]" @click="switchLayer(layer.id)">
                <component :is="layer.icon" class="w-4.5 h-4.5" />
                <span class="truncate">{{ layer.name }}</span>
              </div>
            </Card>
          </Transition>
        </div>
      </div>

      <div class="map-footer" :class="{ 'is-route-open': !isRouteCollapsed }">
        <div class="map-stats pointer-events-none">
          <Card
            class="flex flex-row items-center justify-between min-h-10 gap-3 px-3 py-2 shadow-lg shadow-black/10 border-border/60 bg-card/80 backdrop-blur-md ring-1 ring-white/10 pointer-events-auto hover:bg-card/90 transition-all rounded-xl">
            <!-- Points -->
            <div class="flex items-center gap-2.5">
              <DrawingPinIcon class="w-4 h-4 text-primary" />
              <div class="flex items-baseline gap-1">
                <span class="text-sm font-semibold font-mono tabular-nums tracking-tight break-all">{{ routePoints.length }}</span>
                <span class="text-[11px] font-medium tracking-wide text-muted-foreground/70">PTS</span>
              </div>
            </div>

            <Separator orientation="vertical" class="h-4 bg-border/40" />

            <!-- Distance -->
            <div class="flex items-center gap-2.5 min-w-0">
              <RulerHorizontalIcon class="w-4 h-4 text-primary" />
              <div class="flex items-baseline gap-1">
                <span class="text-sm font-semibold font-mono tabular-nums tracking-tight text-primary break-all">{{
                  formattedDistance.split(' ')[0]
                }}</span>
                <span class="text-[11px] font-medium tracking-wide text-muted-foreground/70">{{
                  formattedDistance.split(' ')[1]
                }}</span>
              </div>
            </div>
          </Card>
        </div>

        <!-- 路线管理 -->
        <Card class="map-route-panel gap-0 p-0 shadow-lg shadow-black/10 border-border/60 bg-card/90 backdrop-blur-md ring-1 ring-white/10 rounded-xl pointer-events-auto">
          <button type="button" :aria-expanded="!isRouteCollapsed" aria-controls="route-panel-content"
            class="map-route-toggle h-10 shrink-0 flex items-center justify-between gap-2 px-3 cursor-pointer select-none border-border/60 hover:bg-primary/5 transition-all"
            :class="{ 'border-b': !isRouteCollapsed }"
            @click="isRouteCollapsed = !isRouteCollapsed">
            <div
              class="flex items-center gap-2 text-xs font-semibold tracking-normal text-muted-foreground group-hover:text-foreground transition-colors">
              <div class="p-1 rounded-md bg-primary/10">
                <ChevronDownIcon v-if="!isRouteCollapsed" class="w-3.5 h-3.5 text-primary" />
                <ChevronUpIcon v-else class="w-3.5 h-3.5 text-primary" />
              </div>
              <span class="whitespace-nowrap text-foreground/80">路线管理</span>
            </div>
            <Badge variant="secondary"
              class="shrink-0 text-[11px] font-semibold px-2 h-5 rounded-md bg-primary/20 text-primary border-primary/20">
              {{ routePoints.length }}
              <span class="ml-1 opacity-60">PTS</span>
            </Badge>
          </button>
          <div id="route-panel-content" v-show="!isRouteCollapsed" class="map-route-content app-scrollbar min-h-0 overflow-y-auto overscroll-contain">
            <RouteManager v-model="routePoints" :current-layer-id="currentLayerId" @locating-point="onLocatingPoint" />
          </div>
        </Card>
      </div>
    </div>

    <div v-if="isMapLoading || mapLoadError" class="absolute inset-0 z-40 flex items-center justify-center bg-background/35 backdrop-blur-[1px] pointer-events-none">
      <Card class="border-border/40 bg-card/90 px-4 py-3 text-sm text-muted-foreground shadow-xl">
        {{ isMapLoading ? '正在加载高德地图…' : mapLoadError }}
      </Card>
    </div>

    <!-- 地图容器 -->
    <div ref="mapContainer" class="w-full h-full grayscale-[0.2] contrast-[1.1]"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed, onUnmounted, nextTick } from 'vue'
import { load as loadAMap } from '@amap/amap-jsapi-loader'
import {
  MagnifyingGlassIcon,
  DrawingPinIcon,
  RulerHorizontalIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  LayersIcon,
  GlobeIcon as Globe,
  Pencil1Icon,
  ResetIcon
} from '@radix-icons/vue'
import { type RoutePoint, calculateRouteDistance, formatDistance } from '../lib/routeUtils'
import { useRoutesStore } from '../stores/routes'
import { WGS84ToGCJ02, GCJ02ToWGS84 } from '../lib/transform'
import { useNotification } from '../composables/useNotification'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Badge } from '@/components/ui/badge'
import RouteManager from './RouteManager.vue'

declare global {
  interface Window {
    AMap?: any
    _AMapSecurityConfig?: { serviceHost: string }
  }
}

const props = defineProps<{
  modelValue: RoutePoint[]
  currentPosition?: { lat: number; lon: number } | null
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [points: RoutePoint[]]
}>()

const { showError: showErrorDialog } = useNotification()
const routesStore = useRoutesStore()

const mapContainer = ref<HTMLDivElement | null>(null)
const searchQuery = ref('')
const searchResults = ref<any[]>([])
const isDrawingMode = ref(false)
const isLayerSwitcherCollapsed = ref(true)
const isSearchExpanded = ref(false)
const isRouteCollapsed = ref(true)
const isMapLoading = ref(false)
const mapLoadError = ref('')

const amapCredentials = {
  key: import.meta.env.VITE_AMAP_JS_API_KEY?.trim() ?? '',
  serviceHost: import.meta.env.VITE_AMAP_SERVICE_HOST?.trim() ?? ''
}

function onLocatingPoint(point: RoutePoint) {
  centerOnPosition(point.lat, point.lon)
}

let searchTimer: number | null = null
let searchController: AbortController | null = null
let resizeTimer: number | null = null
let resizeObserver: ResizeObserver | null = null
watch(searchQuery, newVal => {
  if (searchTimer) clearTimeout(searchTimer)
  searchController?.abort()

  const q = newVal.trim()
  if (!q) {
    searchResults.value = []
    return
  }

  searchTimer = window.setTimeout(() => {
    searchLocation()
  }, 500)
})

const availableLayers = [
  { id: 'amap-vec', name: '高德-矢量', icon: Globe },
  { id: 'amap-img', name: '高德-卫星', icon: Globe }
]

const currentLayerId = ref('amap-vec')

type AMapRuntime = any

let map: AMapRuntime | null = null
let mapClickHandler: ((event: AMapRuntime) => void) | null = null
let baseLayers: Record<string, AMapRuntime> = {}
let routeOverlays: AMapRuntime[] = []
let positionOverlay: AMapRuntime | null = null
let isUnmounted = false

const routePoints = computed<RoutePoint[]>({
  get: () => props.modelValue,
  set: (v: RoutePoint[]) => {
    emit('update:modelValue', v)
  }
})

const canUndo = computed(() => isDrawingMode.value && !props.disabled && routePoints.value.length > 0)

function undoLastPoint() {
  if (!canUndo.value) return
  routePoints.value = routePoints.value.slice(0, -1)
}

const formattedDistance = computed(() => {
  return formatDistance(calculateRouteDistance(routePoints.value))
})

interface AMapCredentials {
  key: string
  serviceHost: string
}

function loadAmapApi(credentials: AMapCredentials): Promise<AMapRuntime> {
  if (window.AMap) return Promise.resolve(window.AMap)

  window._AMapSecurityConfig = { serviceHost: credentials.serviceHost }
  return loadAMap({ key: credentials.key, version: '2.0' })
}

function createBaseLayers(AMap: AMapRuntime) {
  baseLayers = {
    'amap-vec': AMap.createDefaultLayer({ visible: true }),
    'amap-img': [
      new AMap.TileLayer.Satellite({ visible: true, opacity: 1, zIndex: 3 }),
      new AMap.TileLayer.RoadNet({ visible: true, opacity: 1, zIndex: 4 })
    ]
  }
}

function switchLayer(id: string) {
  if (!map || !baseLayers[id]) return
  currentLayerId.value = id
  map.setLayers(Array.isArray(baseLayers[id]) ? baseLayers[id] : [baseLayers[id]])
}

function destroyMap() {
  if (map && mapClickHandler) map.off('click', mapClickHandler)
  mapClickHandler = null
  map?.destroy()
  map = null
  routeOverlays = []
  positionOverlay = null
}

function toMapCoordinate(lat: number, lon: number): [number, number] {
  const [gcjLat, gcjLon] = WGS84ToGCJ02(lat, lon)
  return [gcjLon, gcjLat]
}

async function initializeMap(credentials: AMapCredentials) {
  if (!mapContainer.value || isUnmounted || isMapLoading.value) return

  isMapLoading.value = true
  mapLoadError.value = ''
  try {
    const AMap = await loadAmapApi(credentials)
    if (!mapContainer.value || isUnmounted) return

    createBaseLayers(AMap)
    map = new AMap.Map(mapContainer.value, {
      center: toMapCoordinate(39.9, 116.4),
      zoom: 12,
      zooms: [2, 18],
      viewMode: '2D',
      resizeEnable: true,
      showLabel: true,
      layers: [baseLayers['amap-vec']]
    })
    mapClickHandler = (event: AMapRuntime) => {
      if (props.disabled || !isDrawingMode.value) return

      const [lat, lon] = GCJ02ToWGS84(event.lnglat.getLat(), event.lnglat.getLng())
      routePoints.value = [...routePoints.value, { lat, lon }]
    }
    map.on('click', mapClickHandler)

    updateRouteDisplay()
    updatePositionOverlay(props.currentPosition)
    const initialPosition = props.currentPosition ?? props.modelValue[0]
    if (initialPosition) map.setZoomAndCenter(16, toMapCoordinate(initialPosition.lat, initialPosition.lon))
    await nextTick()
    resizeTimer = window.setTimeout(() => map?.resize(), 200)
  } catch (error) {
    if (isUnmounted) return
    destroyMap()
    mapLoadError.value = error instanceof Error ? error.message : '高德地图加载失败，请检查配置后重试。'
  } finally {
    isMapLoading.value = false
  }
}

onMounted(async () => {
  if (!mapContainer.value) return
  if (!amapCredentials.key || !amapCredentials.serviceHost) {
    mapLoadError.value = '高德地图配置不完整，请填写 frontend/.env 中的 JS API Key 和代理地址。'
  } else {
    await initializeMap(amapCredentials)
  }

  if (!mapContainer.value || typeof ResizeObserver === 'undefined') return

  resizeObserver = new ResizeObserver(entries => {
    const rect = entries[0]?.contentRect
    if (rect && rect.width > 0 && rect.height > 0) map?.resize()
  })
  resizeObserver.observe(mapContainer.value)
})

onUnmounted(() => {
  isUnmounted = true
  if (searchTimer !== null) clearTimeout(searchTimer)
  if (resizeTimer !== null) clearTimeout(resizeTimer)
  searchController?.abort()
  resizeObserver?.disconnect()
  destroyMap()
})

watch(
  () => props.modelValue,
  newPoints => {
    updateRouteDisplay()
    // 保存上次路线
    routesStore.saveLastRoute(newPoints)
  },
  { deep: true }
)

watch(
  () => props.currentPosition,
  pos => {
    updatePositionOverlay(pos)
  },
  { immediate: true }
)

function updateRouteDisplay() {
  if (!map || !window.AMap) return
  if (routeOverlays.length) map.remove(routeOverlays)
  routeOverlays = []

  const points = props.modelValue
  if (points.length === 0) return

  const coords = points.map(p => toMapCoordinate(p.lat, p.lon))
  if (points.length >= 2) {
    routeOverlays.push(new window.AMap.Polyline({
      path: coords,
      strokeColor: '#10b981',
      strokeWeight: 5,
      strokeOpacity: 1,
      lineJoin: 'round',
      lineCap: 'round',
      zIndex: 50,
      clickable: false
    }))
  }

  points.forEach((point, index) => {
    const isStart = index === 0
    const isEnd = !isStart && index === points.length - 1
    routeOverlays.push(new window.AMap.CircleMarker({
      center: coords[index],
      radius: isStart || isEnd ? 7 : 5,
      fillColor: isStart ? '#10b981' : isEnd ? '#ef4444' : '#3b82f6',
      fillOpacity: 1,
      strokeColor: '#ffffff',
      strokeWeight: isStart || isEnd ? 2.5 : 2,
      zIndex: 70 + index,
      clickable: false
    }))
  })

  map.add(routeOverlays)
}

function updatePositionOverlay(position: { lat: number; lon: number } | null | undefined) {
  if (!map || !window.AMap) return
  if (positionOverlay) map.remove(positionOverlay)
  positionOverlay = null
  if (!position) return

  positionOverlay = new window.AMap.CircleMarker({
    center: toMapCoordinate(position.lat, position.lon),
    radius: 10,
    fillColor: '#06b6d4',
    fillOpacity: 1,
    strokeColor: '#ffffff',
    strokeWeight: 3,
    zIndex: 200,
    clickable: false
  })
  map.add(positionOverlay)
}

async function searchLocation() {
  const q = searchQuery.value.trim()
  if (!q) return

  try {
    searchController?.abort()
    searchController = new AbortController()
    const url = `https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(q)}&limit=5`
    const res = await fetch(url, {
      headers: { 'Accept-Language': 'zh-CN' },
      signal: searchController.signal
    })
    if (!res.ok) throw new Error(`搜索请求失败 (${res.status})`)
    const results = await res.json()
    if (searchQuery.value.trim() === q) searchResults.value = results
  } catch (e) {
    if (e instanceof Error && e.name === 'AbortError') return
    showErrorDialog(`操作失败: ${e instanceof Error ? e.message : '未知错误'}`)
    searchResults.value = []
  }
}

function selectSearchResult(result: any) {
  const lat = parseFloat(result.lat)
  const lon = parseFloat(result.lon)

  if (map) {
    map.setZoomAndCenter(16, toMapCoordinate(lat, lon), false, 500)
  }

  searchResults.value = []
  searchQuery.value = ''
}

const { fitToRoute, centerOnPosition } = {
  fitToRoute: () => {
    if (!map || routeOverlays.length === 0) return
    map.setFitView(routeOverlays, false, [50, 50, 50, 50], 18)
  },
  centerOnPosition: (lat: number, lon: number) => {
    if (!map) return
    map.setZoomAndCenter(16, toMapCoordinate(lat, lon), false, 300)
  }
}

defineExpose({
  fitToRoute,
  centerOnPosition
})
</script>

<style scoped>
.map-overlays {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  pointer-events: none;
}

.map-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
  min-width: 0;
  z-index: 20;
}

.map-search {
  width: 40px;
  min-width: 0;
}

.map-search.is-expanded {
  flex: 1;
  max-width: 320px;
}

.map-footer {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: end;
  min-height: 0;
  margin-top: auto;
}

.map-stats {
  width: max-content;
  max-width: 100%;
  min-width: 0;
}

.map-route-panel {
  display: flex;
  flex-direction: column;
  width: 176px;
  min-height: 0;
  max-height: 100%;
  overflow: hidden;
}

.map-footer.is-route-open {
  flex: 1;
  grid-template-columns: minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr) auto;
}

.is-route-open .map-route-panel {
  grid-row: 1;
  justify-self: end;
  width: min(320px, 100%);
  max-height: min(448px, 100%);
}

.is-route-open .map-stats {
  grid-row: 2;
}

.map-route-content {
  flex: 1 1 auto;
}

.map-editor {
  container-type: size;
}

@container (max-width: 440px) {
  .map-overlays {
    padding: 12px;
    gap: 8px;
  }

  .map-toolbar {
    gap: 8px;
  }

  .map-stats :deep(.font-mono) {
    font-size: 12px;
  }

  .map-stats :deep(svg),
  .map-stats :deep([data-slot="separator"]) {
    display: none;
  }

  .map-route-panel {
    width: 160px;
  }
}

@container (max-width: 340px) {
  .map-footer:not(.is-route-open) {
    grid-template-columns: minmax(0, 1fr);
  }

  .map-footer:not(.is-route-open) .map-route-panel {
    grid-row: 1;
    justify-self: end;
  }
}

@container (max-height: 280px) {
  .map-overlays {
    padding: 8px;
    gap: 8px;
  }

  .map-route-toggle {
    height: 32px;
  }

  .map-stats :deep([data-slot="card"]) {
    padding-block: 4px;
  }
}

.no-scrollbar::-webkit-scrollbar {
  display: none;
}

.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
