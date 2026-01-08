<template>
  <div class="card route-config">
    <div class="card-header">
      <span class="card-title">路线设置</span>
      <span class="route-count">{{ routePoints.length }} 个点</span>
    </div>

    <button
      class="btn btn-secondary btn-block"
      @click="$emit('clear')"
      :disabled="disabled || routePoints.length === 0"
    >
      清除路线
    </button>

    <div v-if="routePoints.length > 0" class="route-points">
      <div
        v-for="(point, index) in displayPoints"
        :key="index"
        class="route-point"
      >
        <span class="route-point-index" :class="pointClass(index)">{{ index + 1 }}</span>
        <span class="route-point-coord">
          {{ point.latitude.toFixed(5) }}, {{ point.longitude.toFixed(5) }}
        </span>
      </div>
      <div v-if="routePoints.length > maxDisplay" class="route-point more">
        <span>... 还有 {{ routePoints.length - maxDisplay }} 个点</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Coordinate } from '@/types'

const props = defineProps<{
  routePoints: Coordinate[]
  disabled: boolean
}>()

defineEmits<{
  clear: []
}>()

const maxDisplay = 5

const displayPoints = computed(() => props.routePoints.slice(0, maxDisplay))

function pointClass(index: number): string {
  if (index === 0) return 'start'
  if (index === props.routePoints.length - 1 && index < maxDisplay) return 'end'
  return ''
}
</script>

<style scoped>
.route-config {
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

.route-count {
  font-size: 12px;
  color: var(--text-sidebar-secondary);
}

.route-points {
  margin-top: 12px;
}

.route-point {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid var(--border-color);
  font-size: 12px;
}

.route-point:last-child {
  border-bottom: none;
}

.route-point.more {
  color: var(--text-sidebar-secondary);
  justify-content: center;
}

.route-point-index {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  font-size: 11px;
  font-weight: 600;
  background: var(--bg-card);
  color: var(--text-sidebar);
  border-radius: 50%;
}

.route-point-index.start {
  background: var(--success);
  color: #fff;
}

.route-point-index.end {
  background: var(--danger);
  color: #fff;
}

.route-point-coord {
  color: var(--text-sidebar-secondary);
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace;
}
</style>
