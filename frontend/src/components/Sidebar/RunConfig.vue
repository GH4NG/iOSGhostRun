<template>
  <div class="card run-config">
    <div class="card-title">跑步设置</div>

    <div class="slider-group">
      <div class="slider-header">
        <span class="slider-label">跑步速度</span>
        <span class="slider-value">{{ modelValue.speed.toFixed(1) }} km/h</span>
      </div>
      <input
        type="range"
        :value="modelValue.speed"
        @input="updateField('speed', Number(($event.target as HTMLInputElement).value))"
        min="3"
        max="20"
        step="0.5"
        :disabled="disabled"
      />
    </div>

    <div class="slider-group">
      <div class="slider-header">
        <span class="slider-label">速度随机波动</span>
        <span class="slider-value">±{{ modelValue.speedVariation.toFixed(1) }} km/h</span>
      </div>
      <input
        type="range"
        :value="modelValue.speedVariation"
        @input="updateField('speedVariation', Number(($event.target as HTMLInputElement).value))"
        min="0"
        max="3"
        step="0.1"
        :disabled="disabled"
      />
    </div>

    <div class="slider-group">
      <div class="slider-header">
        <span class="slider-label">路线随机偏移</span>
        <span class="slider-value">±{{ modelValue.routeVariation.toFixed(0) }} 米</span>
      </div>
      <input
        type="range"
        :value="modelValue.routeVariation"
        @input="updateField('routeVariation', Number(($event.target as HTMLInputElement).value))"
        min="0"
        max="10"
        step="1"
        :disabled="disabled"
      />
    </div>

    <div class="form-group">
      <label>循环次数 (0=无限)</label>
      <input
        type="number"
        class="form-control"
        :value="modelValue.loopCount"
        @input="updateField('loopCount', Number(($event.target as HTMLInputElement).value))"
        min="0"
        max="100"
        :disabled="disabled"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { RunConfig } from '@/types'

const props = defineProps<{
  modelValue: RunConfig
  disabled: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: RunConfig]
}>()

function updateField<K extends keyof RunConfig>(field: K, value: RunConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [field]: value })
}
</script>

<style scoped>
.run-config {
  margin-bottom: 16px;
}

.card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-sidebar);
  margin-bottom: 16px;
}

.slider-group {
  margin-bottom: 16px;
}

.slider-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.slider-label {
  font-size: 12px;
  color: var(--text-sidebar-secondary);
}

.slider-value {
  font-size: 13px;
  font-weight: 600;
  color: var(--primary);
  font-variant-numeric: tabular-nums;
}

.form-group {
  margin-bottom: 0;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-size: 12px;
  color: var(--text-sidebar-secondary);
}
</style>
