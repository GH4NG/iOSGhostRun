import { ref } from 'vue'

type ToastType = 'info' | 'success' | 'warning' | 'error'

// 单例状态
const show = ref(false)
const message = ref('')
const type = ref<ToastType>('info')
let timeoutId: number | null = null

export function useToast() {

  function showToast(msg: string, toastType: ToastType = 'info', duration = 3000): void {
    if (timeoutId) {
      clearTimeout(timeoutId)
    }

    message.value = msg
    type.value = toastType
    show.value = true

    timeoutId = window.setTimeout(() => {
      show.value = false
      timeoutId = null
    }, duration)
  }

  function hideToast(): void {
    show.value = false
    if (timeoutId) {
      clearTimeout(timeoutId)
      timeoutId = null
    }
  }

  return {
    show,
    message,
    type,
    showToast,
    hideToast,
  }
}
