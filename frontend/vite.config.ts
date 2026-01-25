import path from 'path'
import tailwindcss from '@tailwindcss/vite'
import wails from '@wailsio/runtime/plugins/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [vue(), wails('./bindings'), tailwindcss()],
    resolve: {
        alias: {
            '@': path.resolve(__dirname, './src')
        }
    },
    build: {
        rollupOptions: {
            output: {
                manualChunks(id) {
                    // 分离 OpenLayers
                    if (id.includes('node_modules/ol')) {
                        return 'openlayers'
                    }
                    // 分离 UI 组件
                    if (id.includes('src/components/ui')) {
                        return 'ui-components'
                    }
                    // 分离其他第三方库
                    if (id.includes('node_modules')) {
                        return 'vendor'
                    }
                }
            }
        }
    }
})
