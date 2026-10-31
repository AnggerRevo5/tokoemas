import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    // source map tidak dibuat untuk produksi agar kode sumber tidak ikut tersaji ke publik
    sourcemap: false,
    // bundel tunggal ±500 kB (±150 kB gzip) setelah library animasi motion ditambahkan
    chunkSizeWarningLimit: 650,
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})