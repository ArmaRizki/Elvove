import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    allowedHosts: [
      'elvove.site',
      // The local nginx proxies with the browser's Host header, so the dev
      // server has to accept it too or every page answers 502.
      'localhost:8090',
      'localhost',
      'frontend',
      'frontend:5173',
    ],
  },
})
