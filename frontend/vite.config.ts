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
    }
})
