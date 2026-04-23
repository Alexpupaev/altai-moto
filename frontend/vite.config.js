import { defineConfig } from 'vite';

export default defineConfig({
  server: {
    port: 3000,
    host: true,            // expose outside Docker container
    watch: {
      usePolling: true,    // needed for HMR inside Docker volumes
    },
  },
  preview: {
    port: 3000,
    host: true,
  },
});
