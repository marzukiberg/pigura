import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// Dev proxy target. Defaults to a local backend (make run); override with VITE_API_URL
// to point at another Pigura server.
const apiTarget = process.env.VITE_API_URL || 'http://127.0.0.1:8080';

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: '../internal/web/dist',
    emptyOutDir: false, // jaga .gitkeep; Makefile membersihkan dist
  },
  server: {
    proxy: {
      '/api': {
        target: apiTarget.startsWith('http') ? apiTarget : `http://${apiTarget}`,
        changeOrigin: true,
        // Server menolak Origin lintas-situs. Origin yang SAMA dengan dev server (UI sendiri)
        // diteruskan sebagai same-origin; origin lain dibiarkan apa adanya (ditolak server).
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq, req) => {
            const origin = proxyReq.getHeader('origin');
            if (!origin) return;
            try {
              if (new URL(String(origin)).host === req.headers.host) {
                proxyReq.setHeader('origin', proxyReq.protocol + '//' + proxyReq.getHeader('host'));
              }
            } catch {
              /* Origin tak valid: biarkan server menolak */
            }
          });
        },
      },
    },
  },
});
