<template>
  <div class="card run-stats">
    <div class="card-header">
      <span class="card-title">运行状态</span>
      <span :class="['status-dot', stats.status]"></span>
    </div>

    <div class="stats-grid">
      <div class="stat-item">
        <div class="stat-value">{{ formattedDistance }}</div>
        <div class="stat-label">总距离 (m)</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ formattedSpeed }}</div>
        <div class="stat-label">当前配速</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ formattedTime }}</div>
        <div class="stat-label">已用时间</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ loopDisplay }}</div>
        <div class="stat-label">当前圈数</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { RunStats } from '@/types'

const props = defineProps<{
  stats: RunStats
  loopCount: number
}>()

const formattedDistance = computed(() => Math.round(props.stats.totalDistance))

const formattedSpeed = computed(() => {
  const kmh = props.stats.currentSpeed
  if (kmh <= 0) return "--'--\""
  const minPerKm = 60 / kmh
  const min = Math.floor(minPerKm)
  const sec = Math.round((minPerKm - min) * 60)
  return `${min}'${sec.toString().padStart(2, '0')}"`
})

const formattedTime = computed(() => {
  const seconds = props.stats.elapsedTime
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }
  return `${m}:${s.toString().padStart(2, '0')}`
})

const loopDisplay = computed(() => {
  const total = props.loopCount === 0 ? '∞' : props.loopCount
  return `${props.stats.currentLoop}/${total}`
})
</script>

<style scoped>
.run-stats {
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

.stats-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.stat-item {
  background: var(--bg-card);
  border-radius: var(--radius-sm);
  padding: 12px;
  text-align: center;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-sidebar);
  font-variant-numeric: tabular-nums;
}

.stat-label {
  font-size: 11px;
  color: var(--text-sidebar-secondary);
  margin-top: 4px;
}
</style>
