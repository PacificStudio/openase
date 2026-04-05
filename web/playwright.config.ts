import { existsSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import { defineConfig, devices } from '@playwright/test'

const nodePath = process.env.PLAYWRIGHT_NODE_PATH ?? process.execPath
const host = process.env.PLAYWRIGHT_HOST ?? '127.0.0.1'
const port = parsePlaywrightPort(process.env.PLAYWRIGHT_PORT)
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? `http://${host}:${port}`
const serverMode = parsePlaywrightServerMode(process.env.PLAYWRIGHT_SERVER_MODE)
const builtIndexPath = fileURLToPath(
  new URL('../internal/webui/static/index.html', import.meta.url),
)
const buildCommand = `${nodePath} ./node_modules/vite/bin/vite.js build --logLevel warn`
const previewCommand = `${nodePath} ./node_modules/vite/bin/vite.js preview --host ${host} --port ${port} --strictPort`
const webServerCommand = existsSync(builtIndexPath)
  ? previewCommand
  : `${buildCommand} && ${previewCommand}`

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
    return raw ?? 'preview'
  }

  throw new Error(`OPENASE_PLAYWRIGHT_SERVER_MODE must be dev or preview, got ${raw}`)
}

function buildPlaywrightWebServerCommand(): string {
  if (serverMode === 'preview') {
    return webServerCommand
  }

  return `${nodePath} ./node_modules/vite/bin/vite.js dev --host ${host} --port ${port} --strictPort`
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
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    headless: true,
    reducedMotion: 'reduce',
  },
  webServer: {
    command: buildPlaywrightWebServerCommand(),
    port,
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
