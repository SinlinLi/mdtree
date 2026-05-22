import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The backend address the dev server proxies API calls to.
const backend = process.env.MDTREE_BACKEND ?? 'http://127.0.0.1:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': backend,
      '/healthz': backend,
    },
  },
  build: {
    // Output into web/dist, which is embedded into the Go binary.
    outDir: 'dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 1500,
  },
})
