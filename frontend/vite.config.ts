import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    // React plugin with fast refresh support
    react(),
  ],

  server: {
    port: 5173,
    proxy: {
      // Proxy all /api requests to the Go backend running on :8080.
      // This allows the frontend dev server to avoid CORS issues during development.
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        // Keep cookies intact so the httpOnly session cookie is forwarded correctly.
        secure: false,
      },
    },
  },

  build: {
    // Enable source maps in production build for easier debugging
    sourcemap: true,
    // Target modern browsers that support ES modules natively
    target: 'esnext',
  },

  // Enable source maps in the dev server as well
  css: {
    devSourcemap: true,
  },
});
