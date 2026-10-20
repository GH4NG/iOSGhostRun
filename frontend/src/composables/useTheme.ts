import { computed, watch } from 'vue'
import { createSharedComposable, useColorMode } from '@vueuse/core'
import { useMapAppearance } from './useMapAppearance'

export const useTheme = createSharedComposable(() => {
    // 首次使用跟随系统；手动切换后保存明确的亮色或暗色选择。
    const mode = useColorMode({ storageKey: 'iosghostrun-theme' })
    const isDark = computed(() => mode.value === 'dark')
    const { ready, themeMode, setThemeMode } = useMapAppearance()
    watch(
        [ready, themeMode],
        () => {
            if (ready.value) mode.value = themeMode.value === 'system' ? 'auto' : themeMode.value
        },
        { immediate: true }
    )

    function toggleTheme() {
        const selected = isDark.value ? 'light' : 'dark'
        mode.value = selected
        void setThemeMode(selected)
    }

    return { isDark, toggleTheme }
})
