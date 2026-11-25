import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom'],
          xterm: ['@xterm/xterm', '@xterm/addon-fit'],
          icons: ['lucide-react'],
        },
      },
    },
  },
})
