<template>
  <div class="flex flex-col h-full bg-card overflow-hidden text-foreground">
    <div class="flex justify-between items-center p-3 px-6 border-b border-border/30 bg-secondary/5">
      <div class="flex gap-2">
        <Tooltip v-for="lv in levels" :key="lv">
          <TooltipTrigger asChild>
            <button
              class="h-8 min-w-14 items-center justify-center rounded-lg text-[10px] font-black uppercase tracking-widest transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 active:scale-95 disabled:opacity-50"
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
      <div class="flex gap-2">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="sm"
              class="h-8 gap-2 text-[10px] font-black uppercase tracking-widest text-muted-foreground/60 hover:text-primary hover:bg-primary/10 transition-all border border-transparent hover:border-primary/20 rounded-lg px-3"
              @click="exportLogs">
              <CopyIcon class="w-3.5 h-3.5" />
              <span>Export</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">{{ copySuccess ? '已复制!' : '导出日志到粘贴板' }}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="sm"
              class="h-8 gap-2 text-[10px] font-black uppercase tracking-widest text-muted-foreground/60 hover:text-destructive hover:bg-destructive/10 transition-all border border-transparent hover:border-destructive/20 rounded-lg px-3"
              @click="clearLogs">
              <TrashIcon class="w-3.5 h-3.5" />
              <span>Clear</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">清除当前显示的所有日志</TooltipContent>
        </Tooltip>
      </div>
    </div>

    <div ref="logContainer"
      class="flex-1 overflow-y-auto p-5 font-mono text-[11px] leading-relaxed no-scrollbar select-text bg-background/30 selection:bg-primary/20">
      <div v-for="(log, index) in filteredLogs" :key="index"
        class="group mb-1.5 py-1.5 rounded-lg px-4 transition-all border border-transparent hover:bg-secondary/20 flex gap-4 items-baseline"
        :class="getLogClass(log.level)">
        <span
          class="text-muted-foreground/30 font-black tracking-tighter w-20 shrink-0 select-none group-hover:text-muted-foreground/60 transition-colors">
          {{ log.time.split(' ')[1] || log.time }}
        </span>
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-3 mb-1">
            <span
              class="font-black uppercase tracking-[0.1em] text-[9px] opacity-80 decoration-2 decoration-primary/20">{{
              log.level }}</span>
            <span
              class="text-blue-500/50 font-bold uppercase tracking-widest text-[9px] bg-blue-500/5 px-1.5 rounded border border-blue-500/10">{{
                log.module }}</span>
          </div>
          <p class="text-foreground/80 break-all whitespace-pre-wrap leading-relaxed">{{ log.message }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { TrashIcon, CopyIcon } from '@radix-icons/vue'
import { Events } from '@wailsio/runtime'
import { LoggerService } from '../../bindings/iOSGhostRun/services'
import { useNotification } from '../composables/useNotification'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

interface LogEntry {
  level: string
  module: string
  message: string
  time: string
}

const logs = ref<LogEntry[]>([])
const filterLevel = ref('all')
const levels = ['all', 'info', 'warn', 'error', 'debug']
const logContainer = ref<HTMLDivElement | null>(null)
const copySuccess = ref(false)
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
      return 'bg-amber-500/10 text-amber-500 border-amber-500/50'
    case 'info':
      return 'text-emerald-500 border-emerald-500/30'
    case 'debug':
      return 'text-purple-400 border-purple-400/30'
    default:
      return ''
  }
}

onMounted(async () => {
  // 加载现有日志
  const existingLogs = await LoggerService.GetLogs()
  logs.value = existingLogs || []

  // 监听新日志
  Events.On('log-event', (ev: any) => {
    const data = ev.data as LogEntry
    logs.value.push(data)
    if (logs.value.length > 1000) {
      logs.value.shift()
    }
    nextTick(scrollToBottom)
  })

  nextTick(scrollToBottom)
})
</script>

<style scoped>
.no-scrollbar::-webkit-scrollbar {
  display: none;
}

.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
