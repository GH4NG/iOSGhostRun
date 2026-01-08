<template>
  <div class="log-panel">
    <div class="log-header">
      <span class="log-title">运行日志</span>
      <div class="log-actions">
        <button class="btn btn-sm btn-secondary" @click="$emit('clear')">清除</button>
        <button class="btn btn-sm btn-secondary" @click="$emit('close')">关闭</button>
      </div>
    </div>
    <div class="log-content" ref="contentRef">
      <div
        v-for="(log, index) in logs"
        :key="index"
        :class="['log-entry', 'log-' + log.level]"
      >
        <span class="log-time">{{ log.time }}</span>
        <span class="log-level">{{ log.level.toUpperCase() }}</span>
        <span class="log-module">[{{ log.module }}]</span>
        <span class="log-message">{{ log.message }}</span>
      </div>
      <div v-if="logs.length === 0" class="log-empty">
        暂无日志
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import type { LogEntry } from '@/types'

const props = defineProps<{
  logs: LogEntry[]
}>()

defineEmits<{
  clear: []
  close: []
}>()

const contentRef = ref<HTMLDivElement | null>(null)

watch(() => props.logs.length, () => {
  nextTick(() => {
    if (contentRef.value) {
      contentRef.value.scrollTop = contentRef.value.scrollHeight
    }
  })
})
</script>

<style scoped>
.log-panel {
  height: 200px;
  background: #0f172a;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  display: flex;
  flex-direction: column;
}

.log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: #1e293b;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.log-title {
  font-size: 13px;
  font-weight: 600;
  color: #e5e7eb;
}

.log-actions {
  display: flex;
  gap: 8px;
}

.log-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px 16px;
  font-family: 'JetBrains Mono', 'SF Mono', 'Monaco', 'Inconsolata', monospace;
  font-size: 12px;
  line-height: 1.6;
}

.log-entry {
  display: flex;
  gap: 8px;
  padding: 2px 0;
}

.log-time {
  color: #6b7280;
  flex-shrink: 0;
}

.log-level {
  width: 50px;
  flex-shrink: 0;
  font-weight: 600;
}

.log-module {
  color: #9ca3af;
  flex-shrink: 0;
}

.log-message {
  color: #e5e7eb;
  word-break: break-all;
}

/* 日志级别颜色 */
.log-debug .log-level {
  color: #9ca3af;
}

.log-info .log-level {
  color: #60a5fa;
}

.log-warn .log-level {
  color: #fbbf24;
}

.log-error .log-level {
  color: #f87171;
}

.log-empty {
  color: #6b7280;
  text-align: center;
  padding: 20px;
}
</style>
