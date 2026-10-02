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
                  class="flex items-center gap-1 p-0 shadow-2xl border-border/30 bg-card/80 backdrop-blur-md ring-1 ring-white/10 group rounded-xl overflow-hidden transition-all duration-500 ease-[cubic-bezier(0.19,1,0.22,1)]"
                  :class="[
                    isSearchExpanded
                      ? 'w-full px-1 py-1'
                      : 'w-10 h-10 ring-0 border-transparent bg-accent text-primary h-10 w-10 border-border/30 !bg-card/85'
                  ]">
                  <Button size="icon" variant="ghost" aria-label="搜索地点" :aria-expanded="isSearchExpanded"
                    class="h-10 w-10 shrink-0 rounded-lg text-primary hover:bg-primary/10 transition-all"
                    @click="isSearchExpanded = !isSearchExpanded">
                    <MagnifyingGlassIcon class="w-6 h-6" />
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
                class="map-search-results app-scrollbar absolute top-full mt-2 w-full max-h-64 overflow-y-auto z-30 shadow-2xl border-border/30 bg-card/95 backdrop-blur-xl ring-1 ring-white/10 p-1.5 flex flex-col gap-1">
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
                class="h-10 w-10 shadow-2xl border-border/30 bg-card/85 backdrop-blur-md ring-1 ring-white/10 transition-all duration-300"
                :class="[
                  isDrawingMode
                    ? 'bg-primary text-primary-foreground border-primary hover:bg-primary/90 shadow-primary/30'
                    : 'text-muted-foreground hover:bg-accent'
                ]" :disabled="disabled" :aria-label="isDrawingMode ? '停止绘制' : '开启路径绘制模式'"
                :aria-pressed="isDrawingMode" @click="isDrawingMode = !isDrawingMode">
                <Pencil1Icon class="w-6 h-6" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">{{ isDrawingMode ? '停止绘制' : '开启路径绘制模式' }}</TooltipContent>
          </Tooltip>
          <Tooltip v-if="isDrawingMode">
            <TooltipTrigger asChild>
              <Button variant="outline" size="icon" aria-label="撤回上一步" :disabled="!canUndo"
                class="h-10 w-10 shadow-2xl border-border/30 bg-card/85 backdrop-blur-md ring-1 ring-white/10 text-muted-foreground hover:bg-accent transition-all duration-300"
                @click="undoLastPoint">
                <ResetIcon class="w-6 h-6" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom" :side-offset="8">撤回上一步</TooltipContent>
          </Tooltip>

          <!-- 图层切换 -->
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="outline" size="icon" aria-label="切换地图图层" :aria-expanded="!isLayerSwitcherCollapsed"
                class="h-10 w-10 shadow-2xl border-border/30 bg-card/85 backdrop-blur-md ring-1 ring-white/10 transition-all duration-300"
                :class="[
                  !isLayerSwitcherCollapsed
                    ? 'bg-primary text-primary-foreground border-primary shadow-primary/30'
                    : 'text-muted-foreground hover:bg-accent'
                ]" @click="isLayerSwitcherCollapsed = !isLayerSwitcherCollapsed">
                <LayersIcon class="w-6 h-6" :class="{ 'text-primary': isLayerSwitcherCollapsed }" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom" align="end">切换地图图层</TooltipContent>
          </Tooltip>

          <Transition enter-active-class="transition duration-300 ease-out"
            enter-from-class="opacity-0 translate-x-4 scale-95" enter-to-class="opacity-100 translate-x-0 scale-100"
            leave-active-class="transition duration-200 ease-in" leave-from-class="opacity-100 translate-x-0 scale-100"
            leave-to-class="opacity-0 translate-x-4 scale-95">
            <Card v-show="!isLayerSwitcherCollapsed"
              class="absolute top-full right-0 mt-2 z-30 w-48 overflow-hidden shadow-2xl border-border/30 bg-card/95 backdrop-blur-xl ring-1 ring-white/10 p-1.5 flex flex-col gap-1">
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
            class="flex flex-row items-center justify-between gap-3 p-2.5 px-3 shadow-2xl border-border/30 bg-card/80 backdrop-blur-md ring-1 ring-white/10 pointer-events-auto hover:bg-card/90 transition-all rounded-xl">
            <!-- Points -->
            <div class="flex items-center gap-2.5">
              <DrawingPinIcon class="w-4 h-4 text-primary" />
              <div class="flex items-baseline gap-1">
                <span class="text-sm font-semibold mono tracking-tight break-all">{{ routePoints.length }}</span>
                <span class="text-[11px] font-medium tracking-wide text-muted-foreground/70">PTS</span>
              </div>
            </div>

            <Separator orientation="vertical" class="h-4 bg-border/40" />

            <!-- Distance -->
            <div class="flex items-center gap-2.5 min-w-0">
              <RulerHorizontalIcon class="w-4 h-4 text-primary" />
              <div class="flex items-baseline gap-1">
                <span class="text-sm font-semibold mono tracking-tight text-primary break-all">{{
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
        <Card class="map-route-panel gap-0 p-0 shadow-2xl border-border/30 bg-card/90 backdrop-blur-md ring-1 ring-white/10 rounded-xl pointer-events-auto">
          <button type="button" :aria-expanded="!isRouteCollapsed" aria-controls="route-panel-content"
            class="map-route-toggle h-10 shrink-0 flex items-center justify-between gap-2 px-3 cursor-pointer select-none border-border/30 hover:bg-primary/5 transition-all"
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

    <!-- 地图容器 -->
    <div ref="mapContainer" class="w-full h-full grayscale-[0.2] contrast-[1.1]"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed, onUnmounted, nextTick } from 'vue'
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
import Map from 'ol/Map'
import View from 'ol/View'
import TileLayer from 'ol/layer/Tile'
import VectorLayer from 'ol/layer/Vector'
import VectorSource from 'ol/source/Vector'
import XYZ from 'ol/source/XYZ'
import { fromLonLat, toLonLat } from 'ol/proj'
import Feature from 'ol/Feature'
import Point from 'ol/geom/Point'
import LineString from 'ol/geom/LineString'
import { Style, Stroke, Circle, Fill } from 'ol/style'
import 'ol/ol.css'
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

let map: Map | null = null
let routeSource: VectorSource | null = null
let positionSource: VectorSource | null = null
let baseLayers: Record<string, TileLayer> = {}

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

// 路线线条样式
const routeStyle = new Style({
  stroke: new Stroke({
    color: '#10b981',
    width: 5,
    lineCap: 'round',
    lineJoin: 'round'
  })
})

// 路线点样式
const startPointStyle = new Style({
  image: new Circle({
    radius: 7,
    fill: new Fill({ color: '#10b981' }),
    stroke: new Stroke({ color: '#ffffff', width: 2.5 })
  }),
  zIndex: 10
})

const midPointStyle = new Style({
  image: new Circle({
    radius: 5,
    fill: new Fill({ color: '#3b82f6' }),
    stroke: new Stroke({ color: '#ffffff', width: 2 })
  }),
  zIndex: 5
})

const endPointStyle = new Style({
  image: new Circle({
    radius: 7,
    fill: new Fill({ color: '#ef4444' }),
    stroke: new Stroke({ color: '#ffffff', width: 2.5 })
  }),
  zIndex: 10
})

// 当前位置样式 - 更显眼的设计，实时显示跑步位置
const currentPositionStyle = new Style({
  image: new Circle({
    radius: 10,
    fill: new Fill({ color: '#06b6d4' }),
    stroke: new Stroke({ color: '#ffffff', width: 3 })
  }),
  zIndex: 100
})

function initBaseLayers() {
  baseLayers = {
    'amap-vec': new TileLayer({
      source: new XYZ({
        url: 'http://wprd0{1-4}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&style=7&x={x}&y={y}&z={z}',
        crossOrigin: 'anonymous'
      }),
      visible: true
    }),
    'amap-img': new TileLayer({
      source: new XYZ({
        url: 'https://webst0{1-4}.is.autonavi.com/appmaptile?style=6&x={x}&y={y}&z={z}',
        crossOrigin: 'anonymous'
      }),
      visible: false
    })
  }
}

function switchLayer(id: string) {
  currentLayerId.value = id
  Object.keys(baseLayers).forEach(key => {
    baseLayers[key].setVisible(key === id)
  })
  // 切换图层时重新显示路由，以应用正确的坐标转换
  updateRouteDisplay()
}

function toMapCoordinate(lat: number, lon: number) {
  if (currentLayerId.value.startsWith('amap')) {
    ;[lat, lon] = WGS84ToGCJ02(lat, lon)
  }
  return fromLonLat([lon, lat])
}

onMounted(() => {
  if (!mapContainer.value) return

  initBaseLayers()
  routeSource = new VectorSource()
  positionSource = new VectorSource()

  map = new Map({
    target: mapContainer.value,
    controls: [],
    layers: [
      ...Object.values(baseLayers),
      new VectorLayer({
        source: routeSource,
        style: feature => {
          if (feature.getGeometry()?.getType() === 'LineString') {
            return routeStyle
          }
          return midPointStyle
        }
      }),
      new VectorLayer({
        source: positionSource,
        style: currentPositionStyle
      })
    ],
    view: new View({
      center: fromLonLat([116.4, 39.9]),
      zoom: 12,
      maxZoom: 18
    })
  })

  map.on('click', evt => {
    if (props.disabled || !isDrawingMode.value) return

    let coords = toLonLat(evt.coordinate) // [lon, lat]

    // 高德地图返回 GCJ-02，需要转换为 WGS84 保存
    if (currentLayerId.value.startsWith('amap')) {
      // 高德地图，将 GCJ-02 转换为 WGS84
      // GCJ02ToWGS84 的参数顺序是 (lat, lon)
      const [wgsLat, wgsLon] = GCJ02ToWGS84(coords[1], coords[0])
      coords = [wgsLon, wgsLat]
    }

    const newPoint: RoutePoint = {
      lat: coords[1],
      lon: coords[0]
    }

    routePoints.value = [...routePoints.value, newPoint]
  })

  updateRouteDisplay()

  nextTick(() => {
    resizeTimer = window.setTimeout(() => {
      map?.updateSize()
    }, 200)
  })

  if (mapContainer.value && typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(entries => {
      const r = entries[0].contentRect
      if (r.width > 0 && r.height > 0 && map) {
        map.updateSize()
      }
    })
    resizeObserver.observe(mapContainer.value)
  }
})

onUnmounted(() => {
  if (searchTimer !== null) clearTimeout(searchTimer)
  if (resizeTimer !== null) clearTimeout(resizeTimer)
  searchController?.abort()
  resizeObserver?.disconnect()
  map?.setTarget(undefined)
  map?.dispose()
  map = null
  routeSource = null
  positionSource = null
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
    if (!positionSource || !map) return
    positionSource.clear()

    if (pos) {
      const feature = new Feature({
        geometry: new Point(toMapCoordinate(pos.lat, pos.lon))
      })
      positionSource.addFeature(feature)
    }
  },
  { immediate: true }
)

function updateRouteDisplay() {
  if (!routeSource) return
  routeSource.clear()

  const points = props.modelValue
  if (points.length === 0) return

  const coords = points.map(p => toMapCoordinate(p.lat, p.lon))
  points.forEach((_, index) => {
    const feature = new Feature({
      geometry: new Point(coords[index])
    })

    // 根据索引设置不同样式
    if (index === 0) {
      feature.setStyle(startPointStyle)
    } else if (index === points.length - 1) {
      feature.setStyle(endPointStyle)
    } else {
      feature.setStyle(midPointStyle)
    }

    routeSource!.addFeature(feature)
  })

  if (points.length >= 2) {
    const lineFeature = new Feature({
      geometry: new LineString(coords)
    })
    routeSource!.addFeature(lineFeature)
  }
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
    map.getView().animate({
      center: toMapCoordinate(lat, lon),
      zoom: 16,
      duration: 500
    })
  }

  searchResults.value = []
  searchQuery.value = ''
}

const { fitToRoute, centerOnPosition } = {
  fitToRoute: () => {
    if (!map || !routeSource || props.modelValue.length === 0) return
    const extent = routeSource.getExtent()
    if (!extent) return
    map.getView().fit(extent, { padding: [50, 50, 50, 50], duration: 500 })
  },
  centerOnPosition: (lat: number, lon: number) => {
    if (!map) return
    map.getView().animate({
      center: toMapCoordinate(lat, lon),
      zoom: 16,
      duration: 300
    })
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

  .map-stats :deep(.mono) {
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
