<template>
    <Tooltip>
        <TooltipTrigger as-child>
            <TitlebarButton
                class="rounded-md text-muted-foreground hover:text-foreground hover:bg-secondary/60"
                :class="compact ? 'h-7 w-7' : 'h-7 w-9'"
                :aria-label="label"
                :aria-pressed="isDark"
                @click="toggleTheme"
            >
                <SunIcon v-if="isDark" class="h-4 w-4" />
                <MoonIcon v-else class="h-4 w-4" />
            </TitlebarButton>
        </TooltipTrigger>
        <TooltipContent side="bottom">{{ label }}</TooltipContent>
    </Tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { MoonIcon, SunIcon } from '@radix-icons/vue'
import { useTheme } from '@/composables/useTheme'
import { TitlebarButton } from '@/components/ui/titlebar-button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

defineProps<{ compact?: boolean }>()

const { isDark, toggleTheme } = useTheme()
const label = computed(() => (isDark.value ? '切换到亮色模式' : '切换到暗色模式'))
</script>
