import { defineConfig, devices } from '@playwright/test'

const nodePath = process.env.PLAYWRIGHT_NODE_PATH ?? process.execPath
const playwrightHost = process.env.OPENASE_PLAYWRIGHT_HOST ?? '127.0.0.1'
const playwrightPort = parsePlaywrightPort(process.env.OPENASE_PLAYWRIGHT_PORT)
const playwrightBaseURL =
  process.env.OPENASE_PLAYWRIGHT_BASE_URL ?? `http://${playwrightHost}:${playwrightPort}`
const playwrightServerMode = parsePlaywrightServerMode(process.env.OPENASE_PLAYWRIGHT_SERVER_MODE)

function parsePlaywrightPort(raw: string | undefined): number {
  if (!raw) {
    return 4173
  }

  const parsed = Number(raw)
  if (!Number.isInteger(parsed) || parsed < 1 || parsed > 65_535) {
    throw new Error(`OPENASE_PLAYWRIGHT_PORT must be a valid TCP port, got ${raw}`)
  }

  return parsed
}

function parsePlaywrightServerMode(raw: string | undefined): 'dev' | 'preview' {
  if (!raw || raw === 'dev' || raw === 'preview') {
    return raw ?? 'dev'
  }

  throw new Error(`OPENASE_PLAYWRIGHT_SERVER_MODE must be dev or preview, got ${raw}`)
}

function buildPlaywrightWebServerCommand(): string {
  if (playwrightServerMode === 'preview') {
    return `${nodePath} ./node_modules/vite/bin/vite.js preview --host ${playwrightHost} --port ${playwrightPort} --strictPort`
  }

  return `${nodePath} ./node_modules/vite/bin/vite.js dev --host ${playwrightHost} --port ${playwrightPort}`
}

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
    command: buildPlaywrightWebServerCommand(),
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
