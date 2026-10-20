<template>
    <DialogRoot :open="open" @update:open="emit('update:open', $event)">
        <DialogPortal>
            <DialogOverlay class="fixed inset-0 z-[10000] bg-black/45 backdrop-blur-[2px]" />
            <DialogContent
                class="fixed left-1/2 top-1/2 z-[10001] max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-2xl border border-border bg-card p-5 text-foreground shadow-2xl focus:outline-none"
                style="--wails-draggable: no-drag"
            >
                <div class="mb-5 flex items-start justify-between gap-3">
                    <div class="space-y-1">
                        <DialogTitle class="text-lg font-bold">设置</DialogTitle>
                        <DialogDescription class="text-sm text-muted-foreground">
                            配置地图与 iOS 17+ 连接所需的运行库。
                        </DialogDescription>
                    </div>
                    <DialogClose as-child>
                        <Button
                            variant="ghost"
                            size="icon"
                            aria-label="关闭设置"
                            class="h-8 w-8 shrink-0"
                        >
                            <Cross1Icon class="h-4 w-4" />
                        </Button>
                    </DialogClose>
                </div>

                <p
                    v-if="locked"
                    role="status"
                    class="mb-4 rounded-xl bg-secondary/50 p-3 text-sm text-muted-foreground"
                >
                    请等待设备操作完成，并停止跑步或手动定位后再修改设置。
                </p>

                <form class="space-y-4" @submit.prevent="saveMap">
                    <div class="space-y-1">
                        <h3 class="text-sm font-semibold">高德地图</h3>
                        <p class="text-xs leading-5 text-muted-foreground">
                            保存后自动重新加载地图，下次启动继续使用。
                        </p>
                    </div>
                    <div class="space-y-2">
                        <label for="amap-key" class="text-sm font-medium">JS API Key</label>
                        <Input
                            id="amap-key"
                            v-model="key"
                            type="text"
                            autocomplete="off"
                            spellcheck="false"
                            placeholder="填写高德 Web 端（JS API）Key"
                            :disabled="loadingMap || savingMap || locked"
                        />
                    </div>
                    <div class="space-y-2">
                        <label for="amap-security-code" class="text-sm font-medium"
                            >安全密钥（securityJsCode）</label
                        >
                        <Input
                            id="amap-security-code"
                            v-model="securityCode"
                            type="password"
                            autocomplete="new-password"
                            spellcheck="false"
                            :placeholder="
                                securityCodeConfigured && key.trim() === savedKey
                                    ? '已保存，留空则保留原密钥'
                                    : '填写与 Key 配套的高德安全密钥'
                            "
                            :disabled="loadingMap || savingMap || locked"
                        />
                        <p class="text-xs leading-5 text-muted-foreground">
                            使用 Web 端（JS API）Key
                            对应的安全密钥。程序会自动启动本地代理，无需另行部署服务器。
                        </p>
                    </div>
                    <p v-if="mapError" role="alert" class="text-sm text-destructive">
                        {{ mapError }}
                    </p>
                    <div class="flex justify-end">
                        <Button type="submit" :disabled="loadingMap || savingMap || locked">
                            {{
                                loadingMap ? '正在读取…' : savingMap ? '正在保存…' : '保存地图设置'
                            }}
                        </Button>
                    </div>
                </form>

                <MapProxyPanel
                    class="mt-4"
                    allow-restart
                    :locked="locked || savingMap"
                    @restarted="emit('map-saved')"
                />

                <MapAppearanceSettings :open="open" :locked="locked" />

                <section
                    class="mt-5 space-y-3 border-t border-border pt-5"
                    aria-labelledby="wintun-title"
                >
                    <div class="flex items-center justify-between gap-3">
                        <h3 id="wintun-title" class="text-sm font-semibold">
                            iOS 17+ 运行库 · Wintun
                        </h3>
                        <span
                            v-if="runtimeStatus"
                            class="text-xs font-medium"
                            :class="
                                runtimeStatus.installed ? 'text-primary' : 'text-muted-foreground'
                            "
                        >
                            {{
                                runtimeStatus.supported
                                    ? runtimeStatus.installed
                                        ? '已安装'
                                        : '未就绪'
                                    : '无需安装'
                            }}
                        </span>
                    </div>
                    <p class="text-xs leading-5 text-muted-foreground">
                        Windows 连接 iOS 17 及以上设备时使用。开发者镜像会在选择设备时自动下载。
                    </p>
                    <div
                        v-if="runtimeStatus"
                        class="space-y-1 rounded-xl border border-border/70 bg-background/60 p-3"
                    >
                        <p class="text-sm" role="status">{{ runtimeStatus.message }}</p>
                        <p v-if="runtimeStatus.supported" class="text-xs text-muted-foreground">
                            程序架构：{{ runtimeStatus.arch }}
                        </p>
                        <p
                            v-if="runtimeStatus.path"
                            class="break-all text-xs text-muted-foreground"
                        >
                            {{ runtimeStatus.path }}
                        </p>
                    </div>
                    <p v-else class="text-sm text-muted-foreground">正在检查运行库…</p>
                    <template v-if="runtimeStatus?.supported">
                        <p class="text-xs leading-5 text-muted-foreground">
                            从官网下载 Wintun 0.14.1，校验 SHA-256
                            后自动选择匹配架构，安装到程序同级目录。 本地导入支持官方 ZIP
                            安装包或对应架构的 DLL。
                        </p>
                        <input
                            ref="fileInput"
                            type="file"
                            accept=".zip,.dll"
                            class="hidden"
                            @change="importRuntime"
                        />
                        <div class="flex flex-wrap gap-2">
                            <Button
                                type="button"
                                :disabled="installingRuntime || locked || runtimeStatus.installed"
                                @click="downloadRuntime"
                            >
                                <DownloadIcon
                                    class="mr-2 h-4 w-4"
                                    :class="{ 'animate-pulse': installingRuntime }"
                                />
                                {{
                                    installingRuntime
                                        ? '正在校验并安装…'
                                        : runtimeStatus.installed
                                          ? '运行库已就绪'
                                          : '一键下载安装'
                                }}
                            </Button>
                            <Button
                                type="button"
                                variant="outline"
                                :disabled="installingRuntime || locked"
                                @click="fileInput?.click()"
                            >
                                导入本地文件
                            </Button>
                        </div>
                    </template>
                    <p v-if="runtimeError" role="alert" class="text-sm text-destructive">
                        {{ runtimeError }}
                    </p>
                </section>
            </DialogContent>
        </DialogPortal>
    </DialogRoot>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Cross1Icon, DownloadIcon } from '@radix-icons/vue'
import {
    DialogRoot,
    DialogPortal,
    DialogOverlay,
    DialogContent,
    DialogTitle,
    DialogDescription,
    DialogClose
} from 'reka-ui'
import { SettingsService } from '../../bindings/iOSGhostRun/services'
import { MapSettings, type WintunStatus } from '../../bindings/iOSGhostRun/services/models'
import MapProxyPanel from './MapProxyPanel.vue'
import MapAppearanceSettings from './MapAppearanceSettings.vue'
import { useMapProxyStatus } from '../composables/useMapProxyStatus'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useNotification } from '../composables/useNotification'

const props = defineProps<{ open: boolean; locked?: boolean }>()
const emit = defineEmits<{ 'update:open': [open: boolean]; 'map-saved': [] }>()
const { showSuccess } = useNotification()
const key = ref('')
const savedKey = ref('')
const securityCode = ref('')
const securityCodeConfigured = ref(false)
const { refresh: refreshProxyStatus } = useMapProxyStatus()
const loadingMap = ref(false)
const savingMap = ref(false)
const mapError = ref('')
const runtimeStatus = ref<WintunStatus | null>(null)
const runtimeError = ref('')
const installingRuntime = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

function errorMessage(error: unknown): string {
    return error instanceof Error ? error.message : String(error)
}

watch(
    () => props.open,
    async open => {
        if (!open) return
        if (!savingMap.value && !loadingMap.value) {
            loadingMap.value = true
            mapError.value = ''
            try {
                const settings = await SettingsService.GetMapSettings()
                key.value = settings.key
                savedKey.value = settings.key
                securityCode.value = ''
                securityCodeConfigured.value = settings.securityCodeConfigured
            } catch (error) {
                mapError.value = errorMessage(error)
            } finally {
                loadingMap.value = false
            }
        }
        try {
            runtimeStatus.value = await SettingsService.GetWintunStatus()
        } catch (error) {
            runtimeError.value = errorMessage(error)
        }
    }
)

async function saveMap() {
    if (props.locked || savingMap.value || loadingMap.value) return
    savingMap.value = true
    mapError.value = ''
    try {
        const settings = await SettingsService.SaveMapSettings(
            new MapSettings({ key: key.value, securityJsCode: securityCode.value })
        )
        key.value = settings.key
        savedKey.value = settings.key
        securityCode.value = ''
        securityCodeConfigured.value = settings.securityCodeConfigured
        await refreshProxyStatus()
        emit('map-saved')
        showSuccess('地图设置已保存，正在重新加载地图')
    } catch (error) {
        mapError.value = errorMessage(error)
    } finally {
        savingMap.value = false
    }
}

async function downloadRuntime() {
    if (props.locked || installingRuntime.value) return
    installingRuntime.value = true
    runtimeError.value = ''
    try {
        runtimeStatus.value = await SettingsService.DownloadAndInstallWintun()
        showSuccess('Wintun 已就绪，请重新选择设备')
    } catch (error) {
        runtimeError.value = errorMessage(error)
    } finally {
        installingRuntime.value = false
    }
}

async function importRuntime(event: Event) {
    const input = event.target as HTMLInputElement
    const file = input.files?.[0]
    input.value = ''
    if (!file || props.locked || installingRuntime.value) return
    installingRuntime.value = true
    runtimeError.value = ''
    try {
        if (file.size > 16 * 1024 * 1024)
            throw new Error('请选择不超过 16 MB 的 Wintun ZIP 或 DLL 文件')
        const encoded = await new Promise<string>((resolve, reject) => {
            const reader = new FileReader()
            reader.onload = () => resolve(String(reader.result).split(',')[1])
            reader.onerror = () => reject(new Error('无法读取文件，请重新选择'))
            reader.readAsDataURL(file)
        })
        runtimeStatus.value = await SettingsService.ImportWintun(file.name, encoded)
        showSuccess('Wintun 已导入，请重新选择设备')
    } catch (error) {
        runtimeError.value = errorMessage(error)
    } finally {
        installingRuntime.value = false
    }
}
</script>
