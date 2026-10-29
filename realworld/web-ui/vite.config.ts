import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const target = env.PDFSIGN_API_ORIGIN || 'http://127.0.0.1:3000';
  return { plugins: [react()], server: { host: '127.0.0.1', port: 5173, strictPort: true, proxy: { '/api/v1': target, '/healthz': target } }, preview: { host: '127.0.0.1', port: 5173, strictPort: true, proxy: { '/api/v1': target, '/healthz': target } } };
});
