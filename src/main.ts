import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import './styles/main.css'
import { installCachedCjkFont } from '@/services/font-cache'

createApp(App).use(router).mount('#app')
void installCachedCjkFont(document, false).catch(() => {})
