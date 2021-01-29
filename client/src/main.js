import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import 'element-plus/lib/theme-chalk/index.css'
import routes from 'vite-plugin-pages/client'
import App from './App.vue'

console.log(routes)

const router = createRouter({
  history: createWebHistory(),
  routes,
})

const app = createApp(App)

app.use(router)
app.use(ElementPlus)

app.mount('#app')