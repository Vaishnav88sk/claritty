import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  base: './',
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8822',
      '/healthz': 'http://localhost:8822'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});

