import { defineConfig, type Plugin } from 'vite';
import vue from '@vitejs/plugin-vue';
import path from 'path';
import { readFileSync } from 'node:fs';

/**
 * The one place a release version is read from. Both the status bar and the About panel render
 * `__APP_VERSION__`, so a bump is a single edit to package.json and the UI cannot drift from the
 * manifest. Wails reads the same version for executable and installer metadata.
 */
const { version: appVersion } = JSON.parse(
  readFileSync(new URL('./package.json', import.meta.url), 'utf-8')
) as { version: string };

function patchQpCode(code: string): string {
  return code
    .replace('return factory(root, root.document)', 'return factory.call(root, root, root.document)')
    .replace('var SVG = this.SVG =', 'var SVG = (typeof window !== "undefined" ? window : this).SVG =')
    .replace('var n=this.SVG=', 'var n=(typeof window !== "undefined" ? window : this).SVG=')
    .replace(
      /window\.clearTimeout\(timeoutId\);[\r\n\s]+timeoutId\s*=\s*null;[\r\n\s]+hideTooltip\(\);/,
      'window.clearTimeout(timeoutId);\n    timeoutId = null;'
    )
    .replace(
      /function showTooltip\(node,\s*tooltip\)\s*\{[\r\n\s]+hideTooltip\(\);/,
      'function showTooltip(node, tooltip) {\n    hideTooltip();\n    timeoutId = null;'
    )
    .replace(
      'function hideTooltip() {',
      'function hideTooltip() {\n    if (typeof window !== "undefined") { window.QP_hideTooltip = hideTooltip; }'
    )
    .replace(
      'document.body.removeChild(currentTooltip);',
      'if (currentTooltip && currentTooltip.parentNode === document.body) { document.body.removeChild(currentTooltip); }'
    );
}

function patchHtmlQueryPlanPlugin(): Plugin {
  return {
    name: 'patch-html-query-plan',
    enforce: 'pre',
    transform(code, id) {
      if (id.includes('html-query-plan')) {
        return {
          code: patchQpCode(code),
          map: null,
        };
      }
    },
  };
}

export default defineConfig({
  plugins: [vue(), patchHtmlQueryPlanPlugin()],
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
  },
  optimizeDeps: {
    esbuildOptions: {
      plugins: [
        {
          name: 'patch-html-query-plan-esbuild',
          setup(build) {
            build.onLoad({ filter: /html-query-plan[\\/]dist[\\/]qp/ }, async (args) => {
              const fs = await import('node:fs/promises');
              const contents = await fs.readFile(args.path, 'utf8');
              return {
                contents: patchQpCode(contents),
                loader: 'js',
              };
            });
          },
        },
      ],
    },

  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  // Keep a predictable endpoint for the Wails development watcher.
  server: {
    port: 1420,
    strictPort: true,
    host: false,
  },
  envPrefix: ['VITE_', 'WAILS_'],
  build: {
    target: 'es2020',
    minify: process.env.WAILS_DEBUG ? false : 'esbuild',
    sourcemap: !!process.env.WAILS_DEBUG,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/monaco-editor')) {
            return 'vendor-monaco';
          }
          if (id.includes('node_modules/tabulator-tables')) {
            return 'vendor-grid';
          }
          if (id.includes('node_modules/sql-formatter')) {
            return 'vendor-sql-formatter';
          }
          if (id.includes('node_modules/html-query-plan')) {
            return 'vendor-html-query-plan';
          }
          if (id.includes('node_modules/vue') || id.includes('node_modules/pinia') || id.includes('node_modules/@vue')) {
            return 'vendor-vue';
          }
        },
      },
    },
  },
});
