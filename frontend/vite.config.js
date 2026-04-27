import { defineConfig } from 'vite';
import fs from 'fs';
import path from 'path';

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
  plugins: [
    {
      name: 'serve-404',
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          const url = (req.url ?? '/').split('?')[0];
          // Allow: root, files with extension, Vite internals
          if (url === '/' || url.includes('.') || url.startsWith('/@')) {
            return next();
          }
          const html = fs.readFileSync(path.resolve(__dirname, 'public/404.html'), 'utf-8');
          res.statusCode = 404;
          res.setHeader('Content-Type', 'text/html; charset=utf-8');
          res.end(html);
        });
      },
    },
  ],
});
