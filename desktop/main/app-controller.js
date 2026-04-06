const fs = require('node:fs')
const path = require('node:path')

const { buildApplicationMenu } = require('./menu')
const { registerDesktopIpc } = require('./ipc')
const { loadWindowState, persistWindowState } = require('./runtime/window-state')

class DesktopAppController {
  constructor(options) {
    this.app = options.app
    this.Menu = options.Menu
    this.ipcMain = options.ipcMain
    this.shell = options.shell
    this.service = options.service
    this.logger = options.logger
    this.paths = options.paths
    this.browserWindowFactory = options.browserWindowFactory
    this.devServerURL = options.devServerURL ?? ''
    this.headless = options.headless ?? false
    this.version = options.version
    this.desktopGuidePath = options.desktopGuidePath ?? path.resolve(__dirname, '..', '..', 'docs', 'en', 'desktop-v1.md')
    this.window = null
    this.disposeIpc = () => {}
    this.serviceEventsBound = false
    this.runtimeState = {
      status: 'booting',
      baseURL: '',
      version: this.version,
      lastError: null,
      paths: {
        desktopLogsDir: this.paths.desktopLogsDir,
        openaseHomeDir: this.paths.openaseHomeDir,
        openaseConfigPath: this.paths.openaseConfigPath,
        openaseLogsDir: this.paths.openaseLogsDir,
      },
      databaseStrategy: {
        mode: 'manual-or-docker',
        summary: 'Desktop v1 requires PostgreSQL prepared through the existing manual or Docker setup paths.',
      },
    }
  }

  async start() {
    this.ensureDirectories()
    this.createWindow()
    this.installMenu()
    this.disposeIpc = registerDesktopIpc({
      ipcMain: this.ipcMain,
      controller: this,
      shell: this.shell,
      paths: this.paths,
    })
    this.bindServiceEvents()

    if (!fs.existsSync(this.paths.openaseConfigPath)) {
      await this.showError({
        code: 'config_missing',
        message: `OpenASE config was not found at ${this.paths.openaseConfigPath}. Desktop v1 expects the existing manual or Docker PostgreSQL setup flow to create this file first.`,
      })
      return
    }

    await this.showLoading('Starting the local OpenASE service and checking readiness...')

    try {
      const started = await this.service.start()
      await this.loadMainSurface(started.baseURL)
    } catch (error) {
      await this.showError({
        code: 'startup_failed',
        message: error.message,
      })
    }
  }

  createWindow() {
    if (this.window) {
      return this.window
    }
    if (typeof this.browserWindowFactory !== 'function') {
      throw new Error('browserWindowFactory is required')
    }

    const bounds = loadWindowState(this.paths.desktopStatePath)
    this.window = this.browserWindowFactory({
      ...bounds,
      minWidth: 1120,
      minHeight: 720,
      show: !this.headless,
      backgroundColor: '#0c1012',
      title: 'OpenASE',
      autoHideMenuBar: false,
      webPreferences: {
        contextIsolation: true,
        nodeIntegration: false,
        preload: path.join(__dirname, '..', 'preload', 'index.js'),
      },
    })

    this.window.once('ready-to-show', () => {
      if (!this.headless) {
        this.window.show()
      }
    })

    this.window.on('close', () => {
      persistWindowState(this.paths.desktopStatePath, this.window)
    })

    return this.window
  }

  installMenu() {
    const menu = buildApplicationMenu({
      Menu: this.Menu,
      app: this.app,
      controller: this,
      shell: this.shell,
      paths: this.paths,
    })
    this.Menu.setApplicationMenu(menu)
  }

  bindServiceEvents() {
    if (this.serviceEventsBound) {
      return
    }
    this.serviceEventsBound = true
    this.service.on('exit', async (metadata) => {
      await this.showError({
        code: 'service_exited',
        message: `The OpenASE service exited unexpectedly${metadata.code === null ? '' : ` (code ${metadata.code})`}.`,
      })
    })
  }

  ensureDirectories() {
    for (const directory of [
      this.paths.desktopUserDataDir,
      this.paths.desktopLogsDir,
      this.paths.desktopRuntimeDir,
      this.paths.openaseLogsDir,
    ]) {
      fs.mkdirSync(directory, { recursive: true })
    }
  }

  async loadMainSurface(baseURL) {
    this.runtimeState = {
      ...this.runtimeState,
      status: 'ready',
      baseURL,
      lastError: null,
    }

    const targetURL = this.devServerURL || baseURL
    this.logger.info('loading desktop surface', { targetURL, baseURL, devServerURL: this.devServerURL })
    await this.window.loadURL(targetURL)
  }

  async showLoading(message) {
    this.runtimeState = {
      ...this.runtimeState,
      status: 'starting',
      lastError: null,
      loadingMessage: message,
    }
    await this.window.loadFile(path.join(__dirname, '..', 'renderer-shell', 'loading.html'))
  }

  async showError(error) {
    this.runtimeState = {
      ...this.runtimeState,
      status: 'error',
      baseURL: '',
      lastError: {
        code: error.code,
        message: error.message,
      },
    }
    this.logger.error('desktop shell error page', error)
    await this.window.loadFile(path.join(__dirname, '..', 'renderer-shell', 'error.html'))
  }

  async restartService() {
    await this.showLoading('Restarting the local OpenASE service...')
    try {
      const started = await this.service.restart()
      await this.loadMainSurface(started.baseURL)
      return this.getRuntimeState()
    } catch (error) {
      await this.showError({
        code: 'restart_failed',
        message: error.message,
      })
      return this.getRuntimeState()
    }
  }

  getRuntimeState() {
    return {
      ...this.runtimeState,
      desktopGuidePath: this.desktopGuidePath,
    }
  }

  getWindow() {
    return this.window
  }

  getDesktopGuidePath() {
    return this.desktopGuidePath
  }

  async shutdown() {
    this.disposeIpc()
    await this.service.stop()
  }

  async quitApplication() {
    await this.shutdown()
    this.app.quit()
  }
}

module.exports = {
  DesktopAppController,
}
