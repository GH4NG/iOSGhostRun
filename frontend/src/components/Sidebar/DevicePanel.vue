<template>
  <div class="card device-panel">
    <div class="card-header">
      <span class="card-title">设备连接</span>
      <button class="btn btn-sm btn-secondary" @click="$emit('refresh')" :disabled="loading">
        {{ loading ? '刷新中...' : '刷新' }}
      </button>
    </div>

    <div v-if="devices.length === 0" class="empty-state">
      <div class="empty-icon">📱</div>
      <p>未检测到设备</p>
      <p class="empty-hint">请连接iOS设备并信任此电脑</p>
    </div>

    <div v-else class="device-list">
      <div
        v-for="device in devices"
        :key="device.udid"
        class="device-item"
        :class="{ selected: selectedDevice === device.udid }"
        @click="$emit('select', device.udid)"
      >
        <div class="device-icon">📱</div>
        <div class="device-info">
          <h3 class="device-name">{{ device.name }}</h3>
          <p class="device-detail">
            {{ device.productType }} · iOS {{ device.productVersion }}
            <span v-if="device.supportsRsd" class="ios17-badge">17+</span>
          </p>
        </div>
        <div v-if="selectedDevice === device.udid" class="device-check">✓</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Device } from '@/types'

defineProps<{
  devices: Device[]
  selectedDevice: string | null
  loading: boolean
}>()

defineEmits<{
  refresh: []
  select: [udid: string]
}>()
</script>

<style scoped>
.device-panel {
  margin-bottom: 16px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-sidebar);
}

.empty-state {
  padding: 32px 16px;
  text-align: center;
}

.empty-icon {
  font-size: 32px;
  margin-bottom: 12px;
  opacity: 0.5;
}

.empty-hint {
  font-size: 12px;
  margin-top: 8px;
  opacity: 0.6;
}

.device-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.device-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--bg-card);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition);
}

.device-item:hover {
  background: var(--bg-card-hover);
}

.device-item.selected {
  background: var(--primary);
}

.device-icon {
  font-size: 24px;
  opacity: 0.8;
}

.device-info {
  flex: 1;
  min-width: 0;
}

.device-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-sidebar);
  margin: 0 0 4px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.device-detail {
  font-size: 12px;
  color: var(--text-sidebar-secondary);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.ios17-badge {
  display: inline-block;
  padding: 1px 5px;
  font-size: 10px;
  font-weight: 600;
  background: var(--warning);
  color: #000;
  border-radius: 4px;
}

.device-check {
  color: #fff;
  font-weight: bold;
}
</style>
