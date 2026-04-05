import { defineConfig, devices } from '@playwright/test'

const nodePath = process.env.PLAYWRIGHT_NODE_PATH ?? process.execPath
const host = process.env.PLAYWRIGHT_HOST ?? '127.0.0.1'
const playwrightPort = Number(process.env.PLAYWRIGHT_PORT ?? '4173')
const playwrightBaseURL = process.env.PLAYWRIGHT_BASE_URL ?? `http://${host}:${playwrightPort}`
const playwrightWebServerMode = process.env.PLAYWRIGHT_WEB_SERVER_MODE ?? 'dev'

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
    baseURL: playwrightBaseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    headless: true,
    reducedMotion: 'reduce',
  },
  webServer: {
    command: `${nodePath} ./node_modules/vite/bin/vite.js ${playwrightWebServerMode} --host ${host} --port ${playwrightPort}`,
    port: playwrightPort,
    timeout: 120_000,
    reuseExistingServer: false,
    env: {
      ...process.env,
      CHOKIDAR_USEPOLLING: '1',
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
