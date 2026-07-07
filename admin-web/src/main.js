import { createApp } from 'vue'
import { createPinia } from 'pinia'
import 'element-plus/dist/index.css'
import './styles/global.css'
import App from './App.vue'
import router from './router'
import { registerElementPlusComponents } from './plugins/elementPlus'

const app = createApp(App)

app.use(createPinia())
app.use(router)
registerElementPlusComponents(app)
app.mount('#app')
