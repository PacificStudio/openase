const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')

const { _electron: electron } = require('playwright')
const { test, expect } = require('@playwright/test')

const desktopRoot = path.resolve(__dirname, '..', '..')
const fixtureBinary = path.resolve(__dirname, '..', 'fixtures', 'fake-openase.cjs')
const hasDisplay = Boolean(process.env.DISPLAY || process.env.WAYLAND_DISPLAY)

test.skip(
  process.platform === 'linux' && !hasDisplay,
  'Electron E2E requires an X11 or Wayland display server on Linux.',
)

test('launches the desktop shell and reaches the hosted OpenASE surface', async () => {
  const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'openase-desktop-e2e-'))
  const configPath = path.join(tempRoot, 'config.yaml')
  fs.writeFileSync(configPath, 'server:\n  mode: all-in-one\n')

  const electronApp = await electron.launch({
    args: ['.'],
    cwd: desktopRoot,
    env: {
      ...process.env,
      OPENASE_DESKTOP_OPENASE_BIN: fixtureBinary,
      OPENASE_DESKTOP_NODE_BINARY: process.execPath,
      OPENASE_DESKTOP_OPENASE_CONFIG: configPath,
      OPENASE_DESKTOP_HEADLESS: '1',
      OPENASE_DESKTOP_USER_DATA_DIR: path.join(tempRoot, 'user-data'),
      OPENASE_DESKTOP_LOGS_DIR: path.join(tempRoot, 'logs'),
    },
  })

  try {
    const page = await electronApp.firstWindow()
    await expect(page).toHaveTitle(/OpenASE Desktop Fixture/)
    await expect(page.getByText('OpenASE workspace is running.')).toBeVisible()
    await expect(page.getByText('API connected: fixture-ok')).toBeVisible()
  } finally {
    await electronApp.close()
  }
})

test('shows the desktop error page when the config is missing', async () => {
  const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'openase-desktop-e2e-missing-config-'))

  const electronApp = await electron.launch({
    args: ['.'],
    cwd: desktopRoot,
    env: {
      ...process.env,
      OPENASE_DESKTOP_OPENASE_BIN: fixtureBinary,
      OPENASE_DESKTOP_NODE_BINARY: process.execPath,
      OPENASE_DESKTOP_OPENASE_CONFIG: path.join(tempRoot, 'missing-config.yaml'),
      OPENASE_DESKTOP_HEADLESS: '1',
      OPENASE_DESKTOP_USER_DATA_DIR: path.join(tempRoot, 'user-data'),
      OPENASE_DESKTOP_LOGS_DIR: path.join(tempRoot, 'logs'),
    },
  })

  try {
    const page = await electronApp.firstWindow()
    await expect(page.getByRole('heading', { name: 'OpenASE could not start' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Retry service' })).toBeVisible()
  } finally {
    await electronApp.close()
  }
})
