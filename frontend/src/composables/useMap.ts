import { ref, nextTick, onUnmounted } from 'vue'
import L from 'leaflet'
import type { Map, LayerGroup, CircleMarker, LatLngExpression } from 'leaflet'
import type { Coordinate } from '@/types'

// 单例状态
let map: Map | null = null
let routeLayer: LayerGroup | null = null
let currentMarker: CircleMarker | null = null

// 平滑动画
let targetPosition: { lat: number; lon: number } | null = null
let currentAnimatedPosition: { lat: number; lon: number } | null = null
let animationFrameId: number | null = null
let lastFrameTime = 0

const drawMode = ref(false)

export function useMap() {

  function initMap(
    containerId: string,
    initialCenter: { lat: number; lon: number },
    onMapClick: (lat: number, lon: number) => void
  ): void {
    nextTick(() => {
      map = L.map(containerId).setView([initialCenter.lat, initialCenter.lon], 16)

      // 高德地图瓦片
      L.tileLayer(
        'https://webrd0{s}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}',
        {
          subdomains: ['1', '2', '3', '4'],
          attribution: '© 高德地图',
        }
      ).addTo(map)

      routeLayer = L.featureGroup().addTo(map)

      map.on('click', (e) => {
        if (drawMode.value) {
          onMapClick(e.latlng.lat, e.latlng.lng)
        }
      })
    })
  }

  function updateRouteOnMap(routePoints: Coordinate[]): void {
    if (!routeLayer || !map) return
    routeLayer.clearLayers()

    if (routePoints.length === 0) return

    const latlngs: LatLngExpression[] = routePoints.map(
      (p) => [p.latitude, p.longitude] as [number, number]
    )

    const polyline = L.polyline(latlngs, {
      color: '#4a9eff',
      weight: 4,
      opacity: 0.8,
    })
    routeLayer.addLayer(polyline)

    routePoints.forEach((point, index) => {
      let color: string
      let radius: number

      if (index === 0) {
        color = '#52c41a'
        radius = 10
      } else if (index === routePoints.length - 1) {
        color = '#ff4d4f'
        radius = 10
      } else {
        color = '#4a9eff'
        radius = 6
      }

      const marker = L.circleMarker([point.latitude, point.longitude], {
        radius,
        fillColor: color,
        color: '#fff',
        weight: 2,
        fillOpacity: 1,
      })
      routeLayer!.addLayer(marker)
    })

    map.fitBounds(polyline.getBounds(), { padding: [50, 50] })
  }

  function updateCurrentPosition(lat: number, lon: number): void {
    if (!map) return

    const gcj02 = wgs84ToGcj02(lat, lon)
    targetPosition = { lat: gcj02.lat, lon: gcj02.lon }

    if (!currentAnimatedPosition) {
      currentAnimatedPosition = { ...targetPosition }
      if (!currentMarker) {
        currentMarker = L.circleMarker([gcj02.lat, gcj02.lon], {
          radius: 12,
          fillColor: '#ff9500',
          color: '#fff',
          weight: 3,
          fillOpacity: 1,
        }).addTo(map)
      } else {
        currentMarker.setLatLng([gcj02.lat, gcj02.lon])
      }
      return
    }

    if (!animationFrameId) {
      animateMarker()
    }
  }

  function animateMarker(time = 0): void {
    // 节流：至少 16ms 间隔
    if (time - lastFrameTime < 16) {
      animationFrameId = requestAnimationFrame(animateMarker)
      return
    }
    lastFrameTime = time

    if (!targetPosition || !currentAnimatedPosition || !currentMarker) {
      animationFrameId = null
      return
    }

    const lerpFactor = 0.15
    const newLat = currentAnimatedPosition.lat + (targetPosition.lat - currentAnimatedPosition.lat) * lerpFactor
    const newLon = currentAnimatedPosition.lon + (targetPosition.lon - currentAnimatedPosition.lon) * lerpFactor

    const distance = Math.abs(targetPosition.lat - newLat) + Math.abs(targetPosition.lon - newLon)

    if (distance < 0.0000001) {
      currentAnimatedPosition = { ...targetPosition }
      currentMarker.setLatLng([targetPosition.lat, targetPosition.lon])
      animationFrameId = null
      return
    }

    currentAnimatedPosition = { lat: newLat, lon: newLon }
    currentMarker.setLatLng([newLat, newLon])

    animationFrameId = requestAnimationFrame(animateMarker)
  }

  function centerMap(routePoints: Coordinate[], defaultCenter: { lat: number; lon: number }): void {
    if (!map) return

    if (routePoints.length > 0) {
      const latlngs: LatLngExpression[] = routePoints.map(
        (p) => [p.latitude, p.longitude] as [number, number]
      )
      const bounds = L.latLngBounds(latlngs)
      map.fitBounds(bounds, { padding: [50, 50] })
    } else {
      map.setView([defaultCenter.lat, defaultCenter.lon], 15)
    }
  }

  function clearCurrentMarker(): void {
    if (animationFrameId) {
      cancelAnimationFrame(animationFrameId)
      animationFrameId = null
    }
    targetPosition = null
    currentAnimatedPosition = null

    if (currentMarker && map) {
      map.removeLayer(currentMarker)
      currentMarker = null
    }
  }

  function clearRouteLayer(): void {
    routeLayer?.clearLayers()
  }

  function toggleDrawMode(): void {
    drawMode.value = !drawMode.value
  }

  // WGS84 转 GCJ02
  function wgs84ToGcj02(lat: number, lon: number): { lat: number; lon: number } {
    const a = 6378245.0
    const ee = 0.00669342162296594323
    const PI = Math.PI

    const outOfChina = (lat: number, lon: number): boolean => {
      return lon < 72.004 || lon > 137.8347 || lat < 0.8293 || lat > 55.8271
    }

    const transformLat = (x: number, y: number): number => {
      let ret = -100.0 + 2.0 * x + 3.0 * y + 0.2 * y * y + 0.1 * x * y + 0.2 * Math.sqrt(Math.abs(x))
      ret += ((20.0 * Math.sin(6.0 * x * PI) + 20.0 * Math.sin(2.0 * x * PI)) * 2.0) / 3.0
      ret += ((20.0 * Math.sin(y * PI) + 40.0 * Math.sin((y / 3.0) * PI)) * 2.0) / 3.0
      ret += ((160.0 * Math.sin((y / 12.0) * PI) + 320 * Math.sin((y * PI) / 30.0)) * 2.0) / 3.0
      return ret
    }

    const transformLon = (x: number, y: number): number => {
      let ret = 300.0 + x + 2.0 * y + 0.1 * x * x + 0.1 * x * y + 0.1 * Math.sqrt(Math.abs(x))
      ret += ((20.0 * Math.sin(6.0 * x * PI) + 20.0 * Math.sin(2.0 * x * PI)) * 2.0) / 3.0
      ret += ((20.0 * Math.sin(x * PI) + 40.0 * Math.sin((x / 3.0) * PI)) * 2.0) / 3.0
      ret += ((150.0 * Math.sin((x / 12.0) * PI) + 300.0 * Math.sin((x / 30.0) * PI)) * 2.0) / 3.0
      return ret
    }

    if (outOfChina(lat, lon)) {
      return { lat, lon }
    }

    let dLat = transformLat(lon - 105.0, lat - 35.0)
    let dLon = transformLon(lon - 105.0, lat - 35.0)
    const radLat = (lat / 180.0) * PI
    let magic = Math.sin(radLat)
    magic = 1 - ee * magic * magic
    const sqrtMagic = Math.sqrt(magic)
    dLat = (dLat * 180.0) / (((a * (1 - ee)) / (magic * sqrtMagic)) * PI)
    dLon = (dLon * 180.0) / ((a / sqrtMagic) * Math.cos(radLat) * PI)

    return {
      lat: lat + dLat,
      lon: lon + dLon,
    }
  }

  onUnmounted(() => {
    if (animationFrameId) {
      cancelAnimationFrame(animationFrameId)
    }
  })

  return {
    drawMode,
    initMap,
    updateRouteOnMap,
    updateCurrentPosition,
    centerMap,
    clearCurrentMarker,
    clearRouteLayer,
    toggleDrawMode,
  }
}
