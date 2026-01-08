<template>
  <div class="card run-controls">
    <div class="card-title">运行控制</div>

    <div class="btn-group" v-if="status === 'idle'">
      <button
        class="btn btn-success btn-block"
        @click="$emit('start')"
        :disabled="!canStart"
      >
        ▶ 开始跑步
      </button>
    </div>

    <div class="btn-group" v-else-if="status === 'running'">
      <button class="btn btn-warning" @click="$emit('pause')">
        ⏸ 暂停
      </button>
      <button class="btn btn-danger" @click="$emit('stop')">
        ⏹ 停止
      </button>
    </div>

    <div class="btn-group" v-else-if="status === 'paused'">
      <button class="btn btn-success" @click="$emit('resume')">
        ▶ 继续
      </button>
      <button class="btn btn-danger" @click="$emit('stop')">
        ⏹ 停止
      </button>
    </div>

    <button
      class="btn btn-secondary btn-block reset-btn"
      @click="$emit('reset')"
      :disabled="status === 'running'"
    >
      🔄 重置真实位置
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  status: 'idle' | 'running' | 'paused'
  canStart: boolean
}>()

defineEmits<{
  start: []
  pause: []
  resume: []
  stop: []
  reset: []
}>()
</script>

<style scoped>
.run-controls {
  margin-bottom: 16px;
}

.card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-sidebar);
  margin-bottom: 12px;
}

.btn-group {
  display: flex;
  gap: 10px;
}

.btn-group .btn {
  flex: 1;
}

.reset-btn {
  margin-top: 10px;
}
</style>
