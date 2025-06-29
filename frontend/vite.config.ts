import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  optimizeDeps: {
    include: ['cookie', 'react-router-dom']
  },
  ssr: {
    noExternal: ['react-router-dom']
  },
  server: {
    allowedHosts: ['230a-103-47-133-181.ngrok-free.app']
  }
})
