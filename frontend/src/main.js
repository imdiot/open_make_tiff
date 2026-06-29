import {createApp} from 'vue'
import App from './App.vue'
import './style.css'
import {WindowShow} from '../wailsjs/runtime/runtime.js'

const app = createApp(App)
app.config.errorHandler = () => WindowShow()
try {
  app.mount('#app')
  WindowShow()
} catch (e) {
  console.error(e)
  WindowShow()
}
