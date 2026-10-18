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
            '@': path.resolve(import.meta.dirname, './src')
        }
    },
    server: {
        host: '127.0.0.1'
    },
    build: {
        rolldownOptions: {
            output: {
                codeSplitting: {
                    groups: [
                        {
                            name: 'ui-components',
                            test: /[/\\]src[/\\]components[/\\]ui[/\\]/,
                            priority: 20
                        },
                        {
                            name: 'vendor',
                            test: /[/\\]node_modules[/\\]/,
                            priority: 10
                        }
                    ]
                }
            }
        }
    }
})
