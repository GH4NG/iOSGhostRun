import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersist from 'pinia-plugin-persistedstate'
import App from './App.vue'
import './styles/main.css'

const app = createApp(App)
const pinia = createPinia()

pinia.use(piniaPluginPersist)

app.use(pinia)
app.mount('#app')
