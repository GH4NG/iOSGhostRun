import { computed, ref, watch } from 'vue'
import { createSharedComposable } from '@vueuse/core'
import { SettingsService } from '../../bindings/iOSGhostRun/services'
import type { MapAppearance } from '../../bindings/iOSGhostRun/services/models'

// 已在高德发布的公开样式标识；API Key 和安全密钥仍由用户在设置中填写。
export const appleInspiredStyleId = '10146c2104d182971b5311278a987024'
export const darkMapStyles = [
    { id: 'darkblue', name: '极夜蓝', styleId: 'darkblue' },
    { id: 'blue', name: '靛青蓝', styleId: 'blue' },
    { id: 'dark', name: '幻影黑', styleId: 'dark' }
]

// 视觉近似的设计参考，并非 Apple 官方色值，也不是高德可直接导入的样式文件。
export const appleInspiredMapPalette = [
    { name: '陆地 / 界面背景', color: '#242628' },
    { name: '建筑物（专业版参考）', color: '#2E3135' },
    { name: '水域', color: '#142A3D' },
    { name: '公园 / 绿地', color: '#263D32' },
    { name: '普通道路', color: '#454B54' },
    { name: '主干道路', color: '#62656B' },
    { name: '高速道路', color: '#9A8055' },
    { name: '文字', color: '#DDE3EB' },
    { name: '文字描边', color: '#202327' },
    { name: '水域文字', color: '#7097BD' },
    { name: '公园文字', color: '#88B69A' }
]

export function normalizeMapStyleId(value: string): string {
    return value.trim().replace(/^amap:\/\/styles\//i, '')
}

export function isValidMapStyleId(value: string): boolean {
    return /^[a-zA-Z0-9_-]{1,128}$/.test(value)
}

export const useMapAppearance = createSharedComposable(() => {
    const darkMapStyle = ref('darkblue')
    const customDarkStyleId = ref('')
    const themeMode = ref<'system' | 'light' | 'dark'>('system')
    const ready = ref(false)
    const storageError = ref('')
    const styleWarning = ref('')
    const checkingStyle = ref(false)
    const verifiedStyleId = ref('')
    let verificationRevision = 0
    let saveQueue: Promise<unknown> = Promise.resolve()
    const savedCustomStyleId = computed(() => {
        const id = normalizeMapStyleId(customDarkStyleId.value)
        return isValidMapStyleId(id) ? id : ''
    })
    const availableDarkMapStyles = computed(() => [
        ...(savedCustomStyleId.value ? [{ id: 'custom', name: '自定义深色' }] : []),
        ...darkMapStyles
    ])
    const darkMapStyleURL = computed(() => {
        const id =
            darkMapStyle.value === 'custom' && savedCustomStyleId.value
                ? verifiedStyleId.value || 'darkblue'
                : (darkMapStyles.find(style => style.id === darkMapStyle.value)?.styleId ??
                  'darkblue')
        return `amap://styles/${id}`
    })

    function currentProfile(): MapAppearance {
        return {
            version: 1,
            darkStyle: darkMapStyle.value,
            customStyleId: savedCustomStyleId.value,
            themeMode: themeMode.value
        } as MapAppearance
    }

    function applyProfile(profile: MapAppearance) {
        darkMapStyle.value = profile.darkStyle
        customDarkStyleId.value = profile.customStyleId
        themeMode.value = profile.themeMode as 'system' | 'light' | 'dark'
    }

    function legacyProfile(): MapAppearance {
        const oldStyle = localStorage.getItem('iosghostrun-dark-map-style')
        const oldCustom = normalizeMapStyleId(
            localStorage.getItem('iosghostrun-custom-dark-map-style-id') || ''
        )
        const customId =
            oldStyle === 'apple'
                ? appleInspiredStyleId
                : isValidMapStyleId(oldCustom)
                  ? oldCustom
                  : ''
        const selected =
            oldStyle === 'apple' || (oldStyle === 'custom' && customId)
                ? 'custom'
                : darkMapStyles.some(style => style.id === oldStyle)
                  ? oldStyle!
                  : 'darkblue'
        const oldTheme = localStorage.getItem('iosghostrun-theme')
        return {
            version: 1,
            darkStyle: selected,
            customStyleId: customId,
            themeMode: oldTheme === 'light' || oldTheme === 'dark' ? oldTheme : 'system'
        } as MapAppearance
    }

    async function initialize() {
        try {
            const stored = await SettingsService.GetMapAppearance()
            if (stored.version === 0) {
                const migrated = legacyProfile()
                applyProfile(await SettingsService.SaveMapAppearance(migrated))
            } else {
                applyProfile(stored)
            }
        } catch (error) {
            storageError.value = error instanceof Error ? error.message : String(error)
        } finally {
            ready.value = true
        }
    }
    const initialization = initialize()

    async function persist() {
        await initialization
        saveQueue = saveQueue
            .catch(() => {})
            .then(() => SettingsService.SaveMapAppearance(currentProfile()))
        try {
            await saveQueue
            storageError.value = ''
        } catch (error) {
            storageError.value = error instanceof Error ? error.message : String(error)
            throw error
        }
    }

    async function verifyCustomStyle() {
        await initialization
        const revision = ++verificationRevision
        const id = darkMapStyle.value === 'custom' ? savedCustomStyleId.value : ''
        verifiedStyleId.value = ''
        styleWarning.value = ''
        if (!id) {
            checkingStyle.value = false
            return
        }
        checkingStyle.value = true
        try {
            const available = await SettingsService.CheckMapStyle(id)
            if (revision !== verificationRevision) return
            if (available) verifiedStyleId.value = id
            else styleWarning.value = '此样式与当前高德 Key 不匹配或尚未发布，已改用「极夜蓝」。'
        } catch (error) {
            if (revision !== verificationRevision) return
            styleWarning.value = `无法验证自定义样式，已改用「极夜蓝」：${error instanceof Error ? error.message : String(error)}`
        } finally {
            if (revision === verificationRevision) checkingStyle.value = false
        }
    }

    watch([ready, darkMapStyle, savedCustomStyleId], () => {
        if (ready.value) void verifyCustomStyle()
    })

    async function selectDarkStyle(value: string) {
        if (value !== 'custom' && !darkMapStyles.some(style => style.id === value)) return
        if (value === 'custom' && !savedCustomStyleId.value) return
        darkMapStyle.value = value
        await persist()
    }

    async function saveCustomDarkStyle(value: string) {
        const id = normalizeMapStyleId(value)
        if (id && !isValidMapStyleId(id))
            throw new Error('请填写高德发布的样式 ID，或 amap://styles/… 格式的样式地址')
        customDarkStyleId.value = id
        if (id) darkMapStyle.value = 'custom'
        else if (darkMapStyle.value === 'custom') darkMapStyle.value = 'darkblue'
        await persist()
    }

    async function setThemeMode(value: 'system' | 'light' | 'dark') {
        themeMode.value = value
        await persist()
    }

    async function importProfile() {
        await initialization
        await saveQueue.catch(() => {})
        const profile = await SettingsService.ImportMapAppearance()
        if (profile.version === 1) {
            applyProfile(profile)
            storageError.value = ''
            return true
        }
        return false
    }

    async function exportProfile() {
        await initialization
        await saveQueue
        return SettingsService.ExportMapAppearance()
    }

    return {
        darkMapStyle,
        themeMode,
        ready,
        storageError,
        styleWarning,
        checkingStyle,
        savedCustomStyleId,
        availableDarkMapStyles,
        darkMapStyleURL,
        selectDarkStyle,
        saveCustomDarkStyle,
        setThemeMode,
        verifyCustomStyle,
        importProfile,
        exportProfile
    }
})
