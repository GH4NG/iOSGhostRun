<template>
  <div class="relative w-full h-full bg-secondary/5 overflow-hidden text-foreground">
    <!-- 搜索栏-->
    <div class="absolute top-6 left-6 z-[100] pointer-events-none">
      <div
        class="relative flex flex-col items-start gap-3 pointer-events-auto transition-all duration-500 ease-[cubic-bezier(0.19,1,0.22,1)]"
        :class="[isSearchExpanded ? 'w-80' : 'w-10']">
        <Tooltip>
          <TooltipTrigger asChild>
            <div
              class="flex items-center gap-1 p-0 shadow-2xl border-border/30 bg-card/80 backdrop-blur-md ring-1 ring-white/10 group rounded-xl overflow-hidden transition-all duration-500 ease-[cubic-bezier(0.19,1,0.22,1)]"
              :class="[
                isSearchExpanded
                  ? 'w-full px-1 py-1'
                  : 'w-10 h-10 ring-0 border-transparent bg-accent text-primary h-10 w-10 border-border/30 !bg-card/85'
              ]">
              <Button size="icon" variant="ghost"
                class="h-10 w-10 shrink-0 rounded-lg text-primary hover:bg-primary/10 transition-all"
                @click="isSearchExpanded = !isSearchExpanded">
                <MagnifyingGlassIcon class="w-6 h-6" />
              </Button>
              <Input v-if="isSearchExpanded" v-model="searchQuery" type="text" placeholder="Search location..."
                class="h-9 border-none bg-transparent focus-visible:ring-0 text-xs font-black uppercase tracking-widest placeholder:text-muted-foreground/30 animate-in fade-in slide-in-from-left-2 duration-500"
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
            class="mt-1 w-full max-h-64 overflow-y-auto z-[101] shadow-2xl border-border/30 bg-card/85 backdrop-blur-xl ring-1 ring-white/10 no-scrollbar p-1.5 flex flex-col gap-1">
            <div v-for="(result, idx) in searchResults" :key="idx"
              class="px-4 py-3 text-[10px] font-black uppercase tracking-wider cursor-pointer rounded-xl transition-colors hover:bg-primary/10 hover:text-primary leading-tight"
              @click="selectSearchResult(result)">
              {{ result.display_name }}
            </div>
          </Card>
        </Transition>
      </div>
    </div>

    <!-- 绘制模式 -->
    <div class="absolute top-6 right-6 z-[100] pointer-events-auto">
      <div class="pointer-events-auto flex gap-3">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="outline" size="icon"
              class="h-10 w-10 shadow-2xl border-border/30 bg-card/85 backdrop-blur-md ring-1 ring-white/10 transition-all duration-300"
              :class="[
                isDrawingMode
                  ? 'bg-primary text-primary-foreground border-primary hover:bg-primary/90 scale-110 shadow-primary/30'
                  : 'text-muted-foreground hover:bg-accent'
              ]" @click="isDrawingMode = !isDrawingMode">
              <Pencil1Icon class="w-6 h-6" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">{{ isDrawingMode ? '停止绘制' : '开启路径绘制模式' }}</TooltipContent>
        </Tooltip>
      </div>
    </div>

    <!-- 图层切换 -->
    <div class="absolute top-[78px] right-6 z-[100] flex flex-col items-end gap-3 pointer-events-auto">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="outline" size="icon"
            class="h-10 w-10 shadow-2xl border-border/30 bg-card/85 backdrop-blur-md ring-1 ring-white/10 transition-all duration-300"
            :class="[
              !isLayerSwitcherCollapsed
                ? 'bg-primary text-primary-foreground border-primary scale-110 shadow-primary/30'
                : 'text-muted-foreground hover:bg-accent'
            ]" @click="isLayerSwitcherCollapsed = !isLayerSwitcherCollapsed">
            <LayersIcon class="w-6 h-6" :class="{ 'text-primary': isLayerSwitcherCollapsed }" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="left">切换地图图层</TooltipContent>
      </Tooltip>

      <Transition enter-active-class="transition duration-300 ease-out"
        enter-from-class="opacity-0 translate-x-4 scale-95" enter-to-class="opacity-100 translate-x-0 scale-100"
        leave-active-class="transition duration-200 ease-in" leave-from-class="opacity-100 translate-x-0 scale-100"
        leave-to-class="opacity-0 translate-x-4 scale-95">
        <Card v-show="!isLayerSwitcherCollapsed"
          class="w-48 overflow-hidden shadow-2xl border-border/30 bg-card/85 backdrop-blur-xl ring-1 ring-white/10 p-1.5 flex flex-col gap-1">
          <div v-for="layer in availableLayers" :key="layer.id"
            class="flex items-center gap-3 px-3 py-2.5 rounded-xl cursor-pointer transition-all text-[11px] font-bold uppercase tracking-wider"
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

    <div class="absolute bottom-8 left-8 z-[100] pointer-events-none">
      <Card
        class="flex items-center gap-6 p-3 px-6 shadow-2xl border-border/30 bg-card/80 backdrop-blur-md ring-1 ring-white/10 pointer-events-auto hover:bg-card/90 transition-all rounded-xl">
        <!-- Points -->
        <div class="flex items-center gap-2.5">
          <DrawingPinIcon class="w-4 h-4 text-primary" />
          <div class="flex items-baseline gap-1">
            <span class="text-sm font-black mono tracking-tighter">{{ routePoints.length }}</span>
            <span class="text-[9px] font-black uppercase tracking-widest text-muted-foreground/50">Pts</span>
          </div>
        </div>

        <Separator orientation="vertical" class="h-4 bg-border/40" />

        <!-- Distance -->
        <div class="flex items-center gap-2.5">
          <RulerHorizontalIcon class="w-4 h-4 text-primary" />
          <div class="flex items-baseline gap-1">
            <span class="text-sm font-black mono tracking-tighter text-primary">{{
              formattedDistance.split(' ')[0]
            }}</span>
            <span class="text-[9px] font-black uppercase tracking-widest text-muted-foreground/50">{{
              formattedDistance.split(' ')[1]
            }}</span>
          </div>
        </div>
      </Card>
    </div>

    <!-- 路线管理 -->
    <div class="absolute bottom-8 right-8 z-[1000] transition-all duration-500 ease-[cubic-bezier(0.19,1,0.22,1)]"
      :class="[isRouteCollapsed ? 'w-48' : 'w-80']">
      <Card
        class="shadow-2xl border-border/30 bg-card/80 backdrop-blur-md ring-1 ring-white/10 overflow-hidden rounded-xl">
        <div
          class="h-12 flex items-center justify-between px-4 cursor-pointer select-none border-b border-border/30 hover:bg-primary/5 transition-all active:scale-[0.98]"
          @click="isRouteCollapsed = !isRouteCollapsed">
          <div
            class="flex items-center gap-3 text-xs font-bold uppercase tracking-widest text-muted-foreground group-hover:text-foreground transition-colors">
            <div class="p-1 rounded-md bg-primary/10">
              <ChevronDownIcon v-if="!isRouteCollapsed" class="w-3.5 h-3.5 text-primary" />
              <ChevronUpIcon v-else class="w-3.5 h-3.5 text-primary" />
            </div>
            <span class="text-foreground/80">路线管理</span>
          </div>
          <Badge variant="secondary" v-if="isRouteCollapsed"
            class="text-[10px] font-black px-2 h-5 rounded-md bg-primary/20 text-primary border-primary/20">
            {{ routePoints.length }}
            <span class="ml-1 opacity-60">PTS</span>
          </Badge>
        </div>
        <div v-show="!isRouteCollapsed" class="max-h-[450px] overflow-hidden">
          <ScrollArea class="h-full">
            <RouteManager v-model="routePoints" :current-layer-id="currentLayerId" @locating-point="onLocatingPoint" />
          </ScrollArea>
        </div>
      </Card>
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
  CursorArrowIcon as Navigation,
  SunIcon as Earth,
  Pencil1Icon
} from '@radix-icons/vue'
import Map from 'ol/Map'
import View from 'ol/View'
import TileLayer from 'ol/layer/Tile'
import VectorLayer from 'ol/layer/Vector'
import VectorSource from 'ol/source/Vector'
import XYZ from 'ol/source/XYZ'
import OSM from 'ol/source/OSM'
import { fromLonLat, toLonLat } from 'ol/proj'
import Feature from 'ol/Feature'
import Point from 'ol/geom/Point'
import LineString from 'ol/geom/LineString'
import { Style, Stroke, Circle, Fill } from 'ol/style'
import 'ol/ol.css'
import { type RoutePoint, calculateRouteDistance, formatDistance, saveLastRoute } from '../lib/routeStorage'
import { WGS84ToGCJ02, GCJ02ToWGS84 } from '../lib/transform'
import { useNotification } from '../composables/useNotification'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
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
watch(searchQuery, newVal => {
  if (searchTimer) clearTimeout(searchTimer)

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
  { id: 'amap-img', name: '高德-卫星', icon: Globe },
  { id: 'osm', name: 'OpenStreetMap', icon: Globe }
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
    }),
    osm: new TileLayer({
      source: new OSM(),
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

    // 根据当前图层进行坐标转换
    // 高德地图返回 GCJ-02，需要转换为 WGS84 保存
    // OSM 已经是 WGS84
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

    const newPoints = [...props.modelValue, newPoint]
    emit('update:modelValue', newPoints)
    saveLastRoute(newPoints)
  })

  updateRouteDisplay()

  nextTick(() => {
    setTimeout(() => {
      if (map) (map as any).updateSize()
    }, 200)
  })

  let resizeObserver: ResizeObserver | null = null
  if (mapContainer.value && typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(entries => {
      const r = entries[0].contentRect
      if (r.width > 0 && r.height > 0 && map) {
        ; (map as any).updateSize()
      }
    })
    resizeObserver.observe(mapContainer.value)
  }

  onUnmounted(() => {
    if (resizeObserver) resizeObserver.disconnect()
  })
})

watch(
  () => props.modelValue,
  newPoints => {
    updateRouteDisplay()
    // 保存上次路线
    if (newPoints && newPoints.length > 0) {
      saveLastRoute(newPoints)
    }
  },
  { deep: true }
)

watch(
  () => props.currentPosition,
  pos => {
    if (!positionSource || !map) return
    positionSource.clear()

    if (pos) {
      // 根据当前图层对实时位置进行坐标转换
      let displayLon = pos.lon
      let displayLat = pos.lat

      if (currentLayerId.value.startsWith('amap')) {
        // 高德地图，将 WGS84 转换为 GCJ-02 显示
        ;[displayLat, displayLon] = WGS84ToGCJ02(pos.lat, pos.lon)
      }
      // OSM 和其他直接使用 WGS84

      const feature = new Feature({
        geometry: new Point(fromLonLat([displayLon, displayLat]))
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

  points.forEach((p, index) => {
    // 根据当前图层进行坐标转换
    // 路线点存储的是 WGS84，需要根据当前图层转换为对应坐标系显示
    let displayLon = p.lon
    let displayLat = p.lat

    if (currentLayerId.value.startsWith('amap')) {
      // 高德地图，将 WGS84 转换为 GCJ-02 显示
      // WGS84ToGCJ02 的参数顺序是 (lat, lon)，返回 [lat, lon]
      ;[displayLat, displayLon] = WGS84ToGCJ02(p.lat, p.lon)
    }

    const feature = new Feature({
      geometry: new Point(fromLonLat([displayLon, displayLat]))
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
    const coords = points.map(p => {
      let displayLon = p.lon
      let displayLat = p.lat

      if (currentLayerId.value.startsWith('amap')) {
        // WGS84ToGCJ02 的参数顺序是 (lat, lon)，返回 [lat, lon]
        ;[displayLat, displayLon] = WGS84ToGCJ02(p.lat, p.lon)
      }

      return fromLonLat([displayLon, displayLat])
    })
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
    const url = `https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(q)}&limit=5`
    const res = await fetch(url, {
      headers: { 'Accept-Language': 'zh-CN' }
    })
    searchResults.value = await res.json()
  } catch (e) {
    showErrorDialog(`操作失败: ${e instanceof Error ? e.message : '未知错误'}`)
    searchResults.value = []
  }
}

function selectSearchResult(result: any) {
  const lat = parseFloat(result.lat)
  const lon = parseFloat(result.lon)

  if (map) {
    map.getView().animate({
      center: fromLonLat([lon, lat]),
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
    map.getView().fit(extent, { padding: [50, 50, 50, 50], duration: 500 })
  },
  centerOnPosition: (lat: number, lon: number) => {
    if (!map) return
    map.getView().animate({
      center: fromLonLat([lon, lat]),
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
.no-scrollbar::-webkit-scrollbar {
  display: none;
}

.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
