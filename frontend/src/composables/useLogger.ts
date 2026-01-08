import { ref } from 'vue'
import type { LogEntry } from '@/types'

const go = window.go

// 单例状态
const logs = ref<LogEntry[]>([])
const showLogPanel = ref(false)

export function useLogger() {

  async function loadLogs(): Promise<void> {
    try {
      if (go?.logger?.Service) {
        const result = await go.logger.Service.GetLogs()
        logs.value = result || []
      }
    } catch (error) {
      console.error('加载日志失败:', error)
    }
  }

  function addLog(entry: LogEntry): void {
    logs.value.push(entry)
    if (logs.value.length > 500) {
      logs.value.shift()
    }
  }

  async function clearLogs(): Promise<void> {
    try {
      if (go?.logger?.Service) {
        await go.logger.Service.ClearLogs()
        logs.value = []
      }
    } catch (error) {
      console.error('清除日志失败:', error)
    }
  }

  function toggleLogPanel(): void {
    showLogPanel.value = !showLogPanel.value
  }

  return {
    logs,
    showLogPanel,
    loadLogs,
    addLog,
    clearLogs,
    toggleLogPanel,
  }
}
