import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles/main.css'
import { wsManager } from './api/ws'

const app = createApp(App)
app.use(createPinia())
app.use(router)

wsManager.connect()

app.mount('#app')