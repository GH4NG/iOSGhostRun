<template>
  <div class="log-panel flex flex-col h-full bg-card overflow-hidden text-foreground">
    <div class="log-toolbar flex shrink-0 flex-wrap justify-between items-center gap-2 p-2 border-b border-border/30 bg-secondary/5">
      <div class="log-filters flex flex-wrap gap-1">
        <Tooltip v-for="lv in levels" :key="lv">
          <TooltipTrigger asChild>
            <button
              class="log-filter h-7 min-w-10 px-2 items-center justify-center rounded-lg text-xs font-semibold uppercase transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 active:scale-95 disabled:opacity-50"
              :class="[
                lv === filterLevel
                  ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20 scale-105'
                  : 'bg-secondary text-muted-foreground hover:bg-secondary/80 hover:text-foreground'
              ]" @click="filterLevel = lv">
              {{ lv }}
            </button>
          </TooltipTrigger>
          <TooltipContent side="bottom">筛选 {{ lv.toUpperCase() }} 级别日志</TooltipContent>
        </Tooltip>
      </div>
      <div class="ml-auto flex shrink-0 gap-1">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="sm" aria-label="导出日志"
              class="h-7 gap-1.5 text-xs font-semibold uppercase text-muted-foreground hover:text-primary hover:bg-primary/10 transition-all border border-transparent hover:border-primary/20 rounded-lg px-2"
              @click="exportLogs">
              <CopyIcon class="w-3.5 h-3.5" />
              <span class="log-action-label">Export</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">{{ copySuccess ? '已复制!' : '导出日志到粘贴板' }}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="sm" aria-label="清空日志"
              class="h-7 gap-1.5 text-xs font-semibold uppercase text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-all border border-transparent hover:border-destructive/20 rounded-lg px-2"
              @click="clearLogs">
              <TrashIcon class="w-3.5 h-3.5" />
              <span class="log-action-label">Clear</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">清除当前显示的所有日志</TooltipContent>
        </Tooltip>
      </div>
    </div>

    <div ref="logContainer"
      class="log-list flex-1 min-h-0 overflow-y-auto p-2 font-sans text-xs leading-relaxed select-text bg-background/30 selection:bg-primary/20">
      <div v-for="(log, index) in filteredLogs" :key="index"
        class="group mb-1 py-1 rounded-lg px-2 transition-all border border-transparent hover:bg-secondary/20 flex gap-3 items-baseline"
        :class="getLogClass(log.level)">
        <span
          class="text-muted-foreground font-medium w-16 shrink-0 select-none">
          {{ log.time.split(' ')[1] || log.time }}
        </span>
        <div class="flex-1 min-w-0">
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1 mb-0.5">
            <span
              class="font-semibold uppercase text-[11px]">{{
              log.level }}</span>
            <span
              class="text-blue-700 dark:text-blue-300 font-semibold uppercase text-[11px] bg-blue-500/10 px-1.5 rounded border border-blue-500/20">{{
                log.module }}</span>
          </div>
          <p class="text-foreground break-all whitespace-pre-wrap leading-relaxed">{{ log.message }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { TrashIcon, CopyIcon } from '@radix-icons/vue'
import { Events } from '@wailsio/runtime'
import { LoggerService } from '../../bindings/iOSGhostRun/services'
import type { LogEntry } from '../../bindings/iOSGhostRun/services/models'
import { useNotification } from '../composables/useNotification'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const logs = ref<LogEntry[]>([])
const filterLevel = ref('all')
const levels = ['all', 'info', 'warn', 'error', 'debug']
const logContainer = ref<HTMLDivElement | null>(null)
const copySuccess = ref(false)
let offLog: (() => void) | null = null
let disposed = false
const { showSuccess, showError, showInfo } = useNotification()

const filteredLogs = computed(() => {
  if (filterLevel.value === 'all') return logs.value
  return logs.value.filter(l => l.level === filterLevel.value)
})

function scrollToBottom() {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

function clearLogs() {
  logs.value = []
}

async function exportLogs() {
  const logsToExport = filterLevel.value === 'all' ? logs.value : filteredLogs.value

  if (logsToExport.length === 0) {
    showInfo('没有日志可导出')
    return
  }

  // 格式化日志
  const formattedLogs = logsToExport
    .map(log => `[${log.time}] [${log.level.toUpperCase()}] [${log.module}] ${log.message}`)
    .join('\n')

  try {
    await navigator.clipboard.writeText(formattedLogs)
    copySuccess.value = true
    showSuccess(`已导出 ${logsToExport.length} 条日志到粘贴板`)
    setTimeout(() => {
      copySuccess.value = false
    }, 2000)
  } catch (err) {
    showError(`导出失败: ${err instanceof Error ? err.message : '未知错误'}`)
  }
}

function getLogClass(level: string) {
  switch (level) {
    case 'error':
      return 'bg-destructive/10 text-destructive border-destructive/50'
    case 'warn':
      return 'bg-amber-500/10 text-amber-700 dark:text-amber-300 border-amber-500/50'
    case 'info':
      return 'text-emerald-700 dark:text-emerald-300 border-emerald-500/30'
    case 'debug':
      return 'text-purple-700 dark:text-purple-300 border-purple-400/30'
    default:
      return ''
  }
}

onMounted(async () => {
  // 加载现有日志
  const existingLogs = await LoggerService.GetLogs()
  if (disposed) return
  logs.value = existingLogs || []

  // 监听新日志
  offLog = Events.On('log-event', ev => {
    const data = ev.data
    logs.value.push(data)
    if (logs.value.length > 1000) {
      logs.value.shift()
    }
    nextTick(scrollToBottom)
  })

  nextTick(scrollToBottom)
})

onUnmounted(() => {
  disposed = true
  offLog?.()
})
</script>

<style scoped>
.log-panel {
  container-type: inline-size;
}

@container (max-width: 480px) {
  .log-action-label {
    display: none;
  }

  .log-filter {
    min-width: 32px;
    padding-inline: 6px;
    font-size: 9px;
    letter-spacing: 0.05em;
  }

  .log-list {
    padding: 4px;
  }
}

.no-scrollbar::-webkit-scrollbar {
  display: none;
}

.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
