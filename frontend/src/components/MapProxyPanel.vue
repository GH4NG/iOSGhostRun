<template>
    <Card class="panel-card flex flex-col gap-0 overflow-hidden p-0" aria-label="地图服务状态">
        <div class="panel-header">
            <div class="flex min-w-0 items-center gap-2">
                <div class="panel-icon"><GlobeIcon class="h-4 w-4 text-primary" /></div>
                <div class="min-w-0">
                    <span class="panel-title">地图服务</span>
                    <span class="panel-description">内置高德安全代理</span>
                </div>
            </div>
            <span
                role="status"
                class="ml-auto flex shrink-0 items-center gap-1.5 text-xs font-semibold"
                :class="
                    state === 'running'
                        ? 'text-primary'
                        : state === 'error'
                          ? 'text-destructive'
                          : 'text-muted-foreground'
                "
            >
                <span
                    class="h-1.5 w-1.5 rounded-full bg-current"
                    :class="{ 'animate-pulse': restarting }"
                />
                {{ label }}
            </span>
        </div>
        <div class="space-y-2 px-3 pb-3 text-xs leading-5">
            <p :role="error ? 'alert' : undefined" class="text-muted-foreground">
                {{ error || status?.message || '正在检查本地地图服务…' }}
            </p>
            <div
                v-if="status?.address"
                class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 rounded-lg bg-secondary/25 px-2 py-1.5 text-muted-foreground"
            >
                <span class="break-all">{{ status.address }}</span>
                <span class="shrink-0">已转发 {{ status.requests }} 次</span>
            </div>
            <div class="flex justify-end">
                <Button
                    v-if="allowRestart"
                    type="button"
                    size="sm"
                    variant="outline"
                    :disabled="locked || restarting"
                    @click="restartService"
                >
                    <UpdateIcon
                        class="mr-1.5 h-3.5 w-3.5"
                        :class="{ 'animate-spin': restarting }"
                    />
                    {{ restarting ? '正在重启…' : status?.running ? '重启服务' : '启动服务' }}
                </Button>
                <Button
                    v-else
                    type="button"
                    size="sm"
                    variant="ghost"
                    @click="emit('open-settings')"
                    >设置</Button
                >
            </div>
        </div>
    </Card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { GlobeIcon, UpdateIcon } from '@radix-icons/vue'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useMapProxyStatus } from '../composables/useMapProxyStatus'

defineProps<{ allowRestart?: boolean; locked?: boolean }>()
const emit = defineEmits<{ 'open-settings': []; restarted: [] }>()
const { status, error, restarting, restart } = useMapProxyStatus()
const state = computed(() => (error.value ? 'error' : (status.value?.state ?? 'loading')))
const label = computed(() =>
    restarting.value
        ? '重启中'
        : ({
              running: '运行中',
              unconfigured: '待配置',
              stopped: '已停止',
              error: '异常',
              loading: '检查中'
          }[state.value] ?? '未知')
)

async function restartService() {
    if (await restart()) emit('restarted')
}
</script>
