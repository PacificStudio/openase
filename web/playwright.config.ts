import { existsSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import { defineConfig, devices } from '@playwright/test'

const nodePath = process.env.PLAYWRIGHT_NODE_PATH ?? process.execPath
const previewPort = process.env.PLAYWRIGHT_PORT ?? '4173'
const builtIndexPath = fileURLToPath(
  new URL('../internal/webui/static/index.html', import.meta.url),
)
const buildCommand = `${nodePath} ./node_modules/vite/bin/vite.js build --logLevel warn`
const previewCommand = `${nodePath} ./node_modules/vite/bin/vite.js preview --host 127.0.0.1 --port ${previewPort}`
const webServerCommand = existsSync(builtIndexPath)
  ? previewCommand
  : `${buildCommand} && ${previewCommand}`

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
    baseURL: `http://127.0.0.1:${previewPort}`,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    headless: true,
    reducedMotion: 'reduce',
  },
  webServer: {
    command: webServerCommand,
    port: Number(previewPort),
    timeout: 120_000,
    reuseExistingServer: false,
    env: {
      ...process.env,
      OPENASE_E2E_MOCK: '1',
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
