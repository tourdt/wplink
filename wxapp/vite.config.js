import { defineConfig } from 'vite'
import * as uniModule from '@dcloudio/vite-plugin-uni'

const uni = uniModule.default?.default || uniModule.default || uniModule['module.exports']

export default defineConfig({
  plugins: [uni()],
})
