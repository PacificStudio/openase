import { defineConfig, devices } from '@playwright/test'

const nodePath = process.env.PLAYWRIGHT_NODE_PATH ?? process.execPath
const host = process.env.PLAYWRIGHT_WEB_HOST ?? process.env.PLAYWRIGHT_HOST ?? '127.0.0.1'
const port = Number(process.env.PLAYWRIGHT_WEB_PORT ?? process.env.PLAYWRIGHT_PORT ?? '4173')
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? `http://${host}:${port}`

export default defineConfig({
  testDir: './tests/e2e',
  globalSetup: './tests/e2e/global-setup.ts',
  fullyParallel: false,
  workers: 1,
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  retries: 0,
  reporter: [['list']],
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    headless: true,
    reducedMotion: 'reduce',
  },
  webServer: {
    command: `${nodePath} ./node_modules/vite/bin/vite.js dev --host ${host} --port ${port}`,
    port,
    timeout: 120_000,
    reuseExistingServer: false,
    env: {
      ...process.env,
      OPENASE_E2E_MOCK: '1',
      CHOKIDAR_USEPOLLING: '1',
    },
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
      },
    },
  ],
})
