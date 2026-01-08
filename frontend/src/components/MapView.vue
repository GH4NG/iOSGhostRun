<template>
  <div class="map-wrapper">
    <div id="map" ref="mapContainer"></div>

    <div class="map-controls">
      <button class="map-btn" @click="$emit('center')" title="定位到路线中心">
        <span>⌖</span>
      </button>
      <button
        class="map-btn"
        :class="{ active: drawMode }"
        @click="$emit('toggleDraw')"
        title="绘制模式"
      >
        <span>✏️</span>
      </button>
      <button
        class="map-btn"
        :class="{ active: showLog }"
        @click="$emit('toggleLog')"
        title="日志面板"
      >
        <span>📋</span>
      </button>
    </div>

    <div v-if="drawMode" class="draw-hint">
      点击地图添加路线点
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  drawMode: boolean
  showLog: boolean
}>()

defineEmits<{
  center: []
  toggleDraw: []
  toggleLog: []
}>()
</script>

<style scoped>
.map-wrapper {
  position: relative;
  flex: 1;
  min-height: 0;
}

#map {
  width: 100%;
  height: 100%;
  z-index: 1;
}

.map-controls {
  position: absolute;
  top: 16px;
  right: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 1000;
}

.map-btn {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border: none;
  border-radius: var(--radius);
  box-shadow: var(--shadow);
  cursor: pointer;
  transition: all var(--transition);
  font-size: 18px;
}

.map-btn:hover {
  background: #fff;
  transform: scale(1.05);
}

.map-btn.active {
  background: var(--primary);
  color: #fff;
}

.draw-hint {
  position: absolute;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  padding: 8px 16px;
  background: rgba(0, 0, 0, 0.75);
  color: #fff;
  font-size: 13px;
  border-radius: var(--radius);
  z-index: 1000;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateX(-50%) translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateX(-50%) translateY(0);
  }
}
</style>
