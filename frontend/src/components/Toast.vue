<template>
  <Transition name="toast">
    <div v-if="show" :class="['toast', type]">
      <span class="toast-icon">{{ icon }}</span>
      <span class="toast-message">{{ message }}</span>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  show: boolean
  message: string
  type: 'info' | 'success' | 'warning' | 'error'
}>()

const icon = computed(() => {
  switch (props.type) {
    case 'success': return '✓'
    case 'warning': return '⚠'
    case 'error': return '✗'
    default: return 'ℹ'
  }
})
</script>

<style scoped>
.toast {
  position: fixed;
  top: 20px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  background: rgba(30, 41, 59, 0.95);
  backdrop-filter: blur(10px);
  color: #fff;
  font-size: 14px;
  border-radius: var(--radius);
  box-shadow: var(--shadow-lg);
  z-index: 10000;
}

.toast-icon {
  font-size: 16px;
}

.toast.success {
  border-left: 3px solid var(--success);
}

.toast.success .toast-icon {
  color: var(--success);
}

.toast.warning {
  border-left: 3px solid var(--warning);
}

.toast.warning .toast-icon {
  color: var(--warning);
}

.toast.error {
  border-left: 3px solid var(--danger);
}

.toast.error .toast-icon {
  color: var(--danger);
}

.toast.info {
  border-left: 3px solid var(--primary);
}

.toast.info .toast-icon {
  color: var(--primary);
}

/* 动画 */
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-20px);
}
</style>
