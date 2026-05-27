import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import './style/global.css'

async function start () {
  if (import.meta.env.VITE_STUB) {
    const { mock } = await import('./stub/api.js')
    mock()
  }
  createApp(App).use(router).mount('#app')
}

start()
