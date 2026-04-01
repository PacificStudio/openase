import tailwindcss from '@tailwindcss/vite'
import { svelteTesting } from '@testing-library/svelte/vite'
import { sveltekit } from '@sveltejs/kit/vite'
import { loadEnv } from 'vite'
import { configDefaults, defineConfig } from 'vitest/config'

const defaultDevHost = '127.0.0.1'
const defaultDevPort = 4173

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.OPENASE_DEV_PROXY_TARGET?.trim()
  const devHost = env.OPENASE_DEV_HOST?.trim() || defaultDevHost
  const devPort = Number(env.OPENASE_DEV_PORT || defaultDevPort)

  return {
    plugins: [tailwindcss(), sveltekit(), svelteTesting()],
    server: proxyTarget
      ? {
          host: devHost,
          port: devPort,
          proxy: {
            '/api': {
              target: proxyTarget,
              changeOrigin: true,
            },
          },
        }
      : {
          host: devHost,
          port: devPort,
        },
    test: {
      environment: 'jsdom',
      exclude: [...configDefaults.exclude, 'tests/e2e/**'],
    },
  }
})
