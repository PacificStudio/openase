const { app, Menu, BrowserWindow, ipcMain, shell } = require('electron')
const path = require('node:path')

const { DesktopAppController } = require('./app-controller')
const { createFileLogger } = require('./runtime/logger')
const { resolveDesktopPaths } = require('./runtime/paths')
const { OpenASEServiceProcess } = require('./runtime/service-process')
const { enforceSingleInstanceLock } = require('./runtime/single-instance')

const isHeadless = process.env.OPENASE_DESKTOP_HEADLESS === '1'

if (isHeadless) {
  app.disableHardwareAcceleration()
  app.commandLine.appendSwitch('headless')
  app.commandLine.appendSwitch('disable-gpu')
  app.commandLine.appendSwitch('ozone-platform', 'headless')
}

const desktopUserDataDir = process.env.OPENASE_DESKTOP_USER_DATA_DIR || app.getPath('userData')
const desktopLogsDir = process.env.OPENASE_DESKTOP_LOGS_DIR || app.getPath('logs')
const paths = resolveDesktopPaths({
  desktopUserDataDir,
  desktopLogsDir,
  env: process.env,
})
const logger = createFileLogger(paths.desktopHostLogPath)
const service = new OpenASEServiceProcess({
  env: process.env,
  paths,
  logger,
})
const controller = new DesktopAppController({
  app,
  Menu,
  ipcMain,
  shell,
  paths,
  service,
  logger,
  version: app.getVersion(),
  devServerURL: process.env.OPENASE_DESKTOP_DEV_SERVER_URL,
  headless: isHeadless,
  desktopGuidePath: app.isPackaged
    ? path.join(process.resourcesPath, 'docs', 'desktop-v1.md')
    : path.resolve(__dirname, '..', '..', 'docs', 'en', 'desktop-v1.md'),
  browserWindowFactory: (options) => new BrowserWindow(options),
})

function focusExistingWindow() {
  const window = controller.getWindow()
  if (!window) {
    return
  }
  if (window.isMinimized()) {
    window.restore()
  }
  window.focus()
}

if (!enforceSingleInstanceLock(app, focusExistingWindow)) {
  process.exit(0)
}

app.whenReady().then(async () => {
  logger.info('desktop shell ready', {
    version: app.getVersion(),
    packaged: app.isPackaged,
  })
  await controller.start()
})

app.on('activate', async () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    await controller.start()
    return
  }
  focusExistingWindow()
})

app.on('before-quit', async () => {
  await controller.shutdown()
})

app.on('window-all-closed', () => {
  app.quit()
})
