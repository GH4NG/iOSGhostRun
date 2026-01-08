<template>
  <div class="status-bar">
    <div class="status-section">
      <span :class="['status-dot', connected ? 'connected' : '']"></span>
      <span>{{ connected ? '设备已连接' : '未连接设备' }}</span>
    </div>

    <div class="status-section" v-if="currentLat && currentLon">
      <span>📍 {{ currentLat.toFixed(5) }}, {{ currentLon.toFixed(5) }}</span>
    </div>

    <div class="status-section">
      <span>📏 路线距离: {{ routeDistance }}m</span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  connected: boolean
  currentLat: number
  currentLon: number
  routeDistance: number
}>()
</script>

<style scoped>
.status-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--status-bar-height);
  padding: 0 16px;
  background: #1e293b;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  font-size: 12px;
  color: #9ca3af;
}

.status-section {
  display: flex;
  align-items: center;
  gap: 6px;
}
</style>
