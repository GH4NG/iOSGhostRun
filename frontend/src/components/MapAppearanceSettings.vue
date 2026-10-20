<template>
    <section
        class="mt-5 space-y-3 border-t border-border pt-5"
        aria-labelledby="map-appearance-title"
    >
        <div class="space-y-1">
            <h3 id="map-appearance-title" class="text-sm font-semibold">深色地图外观</h3>
            <p class="text-xs leading-5 text-muted-foreground">
                新安装默认使用高德「极夜蓝」。自定义样式须与当前 JS API Key 属于同一高德账号；
                若不可用，地图会回退到「极夜蓝」。
            </p>
            <p class="text-xs leading-5 text-muted-foreground">
                「冷灰夜色」的配色可供参考；跨账号使用时，先在高德共享并复制样式到自己的账号，再填写新的样式
                ID。
            </p>
        </div>
        <details class="rounded-xl border border-border/70 bg-background/60 p-3">
            <summary class="cursor-pointer text-xs font-medium">查看配色参考</summary>
            <p class="mt-2 text-xs leading-5 text-muted-foreground">
                以下为视觉近似配色，并非 Apple 官方色值。按对应地图元素设置颜色后发布。
            </p>
            <dl class="mt-3 grid grid-cols-1 gap-2 min-[420px]:grid-cols-2">
                <div
                    v-for="item in appleInspiredMapPalette"
                    :key="item.name"
                    class="flex items-center gap-2 text-xs"
                >
                    <span
                        class="h-6 w-6 shrink-0 rounded-md border border-border"
                        :style="{ backgroundColor: item.color }"
                        aria-hidden="true"
                    />
                    <div class="min-w-0">
                        <dt class="text-muted-foreground">{{ item.name }}</dt>
                        <dd class="font-mono">{{ item.color }}</dd>
                    </div>
                </div>
            </dl>
        </details>
        <form class="space-y-3" @submit.prevent="save">
            <div class="space-y-2">
                <label for="amap-dark-style-id" class="text-sm font-medium"
                    >自定义深色样式 ID</label
                >
                <Input
                    id="amap-dark-style-id"
                    v-model="draftStyleId"
                    type="text"
                    autocomplete="off"
                    spellcheck="false"
                    placeholder="样式 ID 或 amap://styles/…"
                    :disabled="locked || busy || !ready"
                    :aria-invalid="!!error"
                    aria-describedby="amap-dark-style-help"
                    @input="clearFeedback"
                />
                <p id="amap-dark-style-help" class="text-xs leading-5 text-muted-foreground">
                    保存后在暗色模式下验证并应用；亮色模式使用标准地图。留空保存可恢复内置样式。
                </p>
            </div>
            <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
            <p v-if="storageError" role="alert" class="text-sm text-destructive">
                {{ storageError }}
            </p>
            <p v-if="styleWarning" role="status" class="text-xs text-amber-500">
                {{ styleWarning }}
            </p>
            <p v-if="checkingStyle" role="status" class="text-xs text-muted-foreground">
                正在验证自定义样式…
            </p>
            <div class="flex flex-wrap items-center justify-between gap-2">
                <Button type="button" variant="outline" @click="openEditor"
                    >打开高德样式编辑器</Button
                >
                <Button type="submit" :disabled="locked || busy || !ready">保存地图外观</Button>
            </div>
            <p v-if="saved" role="status" class="text-xs text-primary">{{ saved }}</p>
        </form>
        <div class="space-y-2 rounded-xl border border-border/70 bg-background/60 p-3">
            <p class="text-xs leading-5 text-muted-foreground">
                外观偏好保存在程序数据目录的 map-appearance.json。导出的档案不含高德 Key
                或安全密钥；换电脑后需单独填写地图凭据。
            </p>
            <div class="flex flex-wrap gap-2">
                <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    :disabled="locked || busy || !ready"
                    @click="importAppearance"
                    >导入外观档案</Button
                >
                <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    :disabled="locked || busy || !ready"
                    @click="exportAppearance"
                    >导出外观档案</Button
                >
                <Button
                    v-if="styleWarning"
                    type="button"
                    size="sm"
                    variant="ghost"
                    :disabled="busy || checkingStyle"
                    @click="verifyCustomStyle"
                    >重试样式验证</Button
                >
            </div>
        </div>
    </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Browser } from '@wailsio/runtime'
import { appleInspiredMapPalette, useMapAppearance } from '@/composables/useMapAppearance'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const props = defineProps<{ open: boolean; locked?: boolean }>()
const {
    savedCustomStyleId,
    saveCustomDarkStyle,
    importProfile,
    exportProfile,
    ready,
    storageError,
    styleWarning,
    checkingStyle,
    verifyCustomStyle
} = useMapAppearance()
const draftStyleId = ref('')
const error = ref('')
const saved = ref('')
const busy = ref(false)

watch(
    [() => props.open, ready],
    ([open, loaded]) => {
        if (!open || !loaded) return
        draftStyleId.value = savedCustomStyleId.value
        error.value = ''
        saved.value = ''
    },
    { immediate: true }
)

function clearFeedback() {
    error.value = ''
    saved.value = ''
}

async function save() {
    if (props.locked || busy.value) return
    error.value = ''
    saved.value = ''
    busy.value = true
    try {
        await saveCustomDarkStyle(draftStyleId.value)
        draftStyleId.value = savedCustomStyleId.value
        saved.value = savedCustomStyleId.value
            ? '已保存；验证通过后暗色模式使用此自定义样式。'
            : '已恢复使用内置地图样式。'
    } catch (value) {
        error.value = value instanceof Error ? value.message : String(value)
    } finally {
        busy.value = false
    }
}

async function importAppearance() {
    if (props.locked || busy.value) return
    clearFeedback()
    busy.value = true
    try {
        if (await importProfile()) {
            draftStyleId.value = savedCustomStyleId.value
            saved.value = '外观档案已导入，地图样式正在验证。'
        }
    } catch (value) {
        error.value = value instanceof Error ? value.message : String(value)
    } finally {
        busy.value = false
    }
}

async function exportAppearance() {
    if (props.locked || busy.value) return
    clearFeedback()
    busy.value = true
    try {
        const path = await exportProfile()
        if (path) saved.value = `外观档案已导出：${path}`
    } catch (value) {
        error.value = value instanceof Error ? value.message : String(value)
    } finally {
        busy.value = false
    }
}

async function openEditor() {
    try {
        await Browser.OpenURL('https://lbs.amap.com/dev/mapstyle/index')
    } catch {
        error.value = '无法打开浏览器，请访问 https://lbs.amap.com/dev/mapstyle/index'
    }
}
</script>
