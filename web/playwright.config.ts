import { defineConfig, devices } from '@playwright/test'

const nodePath = process.env.PLAYWRIGHT_NODE_PATH ?? process.execPath
const playwrightPort = readPortFromEnv('OPENASE_PLAYWRIGHT_PORT', 4173)
const playwrightBaseURL = `http://127.0.0.1:${playwrightPort}`

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
    command: `${nodePath} ./node_modules/vite/bin/vite.js dev --host 127.0.0.1 --port ${playwrightPort}`,
    port: playwrightPort,
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

function readPortFromEnv(name: string, fallback: number): number {
  const raw = process.env[name]?.trim()
  if (!raw) {
    return fallback
  }

  const parsed = Number.parseInt(raw, 10)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    return fallback
  }

  return parsed
}
