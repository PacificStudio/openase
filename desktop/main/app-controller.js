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
    this.setupRuntime = options.setupRuntime
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
      loadingMessage: '',
      paths: {
        desktopLogsDir: this.paths.desktopLogsDir,
        openaseHomeDir: this.paths.openaseHomeDir,
        openaseConfigPath: this.paths.openaseConfigPath,
        openaseLogsDir: this.paths.openaseLogsDir,
      },
      databaseStrategy: {
        mode: 'manual-or-docker',
        summary: 'Desktop v1 prepares PostgreSQL through an existing database connection or a Docker-backed local PostgreSQL flow.',
      },
      setup: {
        bootstrap: null,
        preflight: null,
        lastApply: null,
      },
    }
  }

  async start() {
    try {
      this.ensureDirectories()
      this.createWindow()
      this.installMenu()
      this.disposeIpc()
      this.disposeIpc = registerDesktopIpc({
        ipcMain: this.ipcMain,
        controller: this,
        shell: this.shell,
        paths: this.paths,
      })
      this.bindServiceEvents()
      await this.resumeStartup()
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
    ]) {
      fs.mkdirSync(directory, { recursive: true })
    }
  }

  async resumeStartup() {
    const preflight = await this.setupRuntime.preflight()
    this.runtimeState = {
      ...this.runtimeState,
      setup: {
        ...this.runtimeState.setup,
        preflight,
      },
    }

    if (!preflight.ready) {
      const bootstrap = await this.setupRuntime.bootstrap()
      await this.showSetup({ preflight, bootstrap })
      return this.getRuntimeState()
    }

    return this.launchMainSurface('Starting the local OpenASE service and checking readiness...')
  }

  async launchMainSurface(message) {
    await this.showLoading(message)
    try {
      const started = await this.service.start()
      await this.loadMainSurface(started.baseURL)
      return this.getRuntimeState()
    } catch (error) {
      await this.showError({
        code: 'startup_failed',
        message: error.message,
      })
      return this.getRuntimeState()
    }
  }

  async loadMainSurface(baseURL) {
    this.runtimeState = {
      ...this.runtimeState,
      status: 'ready',
      baseURL,
      lastError: null,
      loadingMessage: '',
    }

    const targetURL = this.devServerURL || baseURL
    this.logger.info('loading desktop surface', { targetURL, baseURL, devServerURL: this.devServerURL })
    await this.window.loadURL(targetURL)
  }

  async showLoading(message) {
    this.runtimeState = {
      ...this.runtimeState,
      status: 'starting',
      baseURL: '',
      lastError: null,
      loadingMessage: message,
    }
    await this.window.loadFile(path.join(__dirname, '..', 'renderer-shell', 'loading.html'))
  }

  async showSetup({ preflight, bootstrap, applyResult }) {
    const setupState = {
      bootstrap: bootstrap ?? this.runtimeState.setup.bootstrap,
      preflight: preflight ?? this.runtimeState.setup.preflight,
      lastApply: applyResult ?? this.runtimeState.setup.lastApply,
    }
    const primaryIssue = (setupState.preflight?.issues ?? [])[0] ?? (setupState.lastApply?.issues ?? [])[0] ?? null

    this.runtimeState = {
      ...this.runtimeState,
      status: 'setup-required',
      baseURL: '',
      loadingMessage: '',
      lastError: primaryIssue
        ? {
            code: primaryIssue.code,
            message: primaryIssue.message,
          }
        : null,
      setup: setupState,
    }

    await this.window.loadFile(path.join(__dirname, '..', 'renderer-shell', 'setup.html'))
  }

  async showError(error) {
    this.runtimeState = {
      ...this.runtimeState,
      status: 'error',
      baseURL: '',
      loadingMessage: '',
      lastError: {
        code: error.code,
        message: error.message,
      },
    }
    this.logger.error('desktop shell error page', error)
    await this.window.loadFile(path.join(__dirname, '..', 'renderer-shell', 'error.html'))
  }

  async restartService() {
    await this.service.stop()
    return this.resumeStartup()
  }

  async recheckSetup() {
    await this.service.stop()
    return this.resumeStartup()
  }

  async applySetup(request) {
    const mode = request?.database?.type === 'docker' ? 'Docker PostgreSQL' : 'existing PostgreSQL'
    await this.showLoading(`Preparing ${mode} and writing the OpenASE config...`)

    try {
      const applyResult = await this.setupRuntime.apply({
        ...request,
        allow_overwrite: true,
      })
      if (!applyResult.ready) {
        const bootstrap = await this.setupRuntime.bootstrap()
        const preflight = {
          ready: false,
          config_path: applyResult.config_path,
          openase_home_dir: applyResult.openase_home_dir,
          issues: applyResult.issues ?? [],
        }
        await this.showSetup({ bootstrap, preflight, applyResult })
        return this.getRuntimeState()
      }
      return this.resumeStartup()
    } catch (error) {
      this.logger.error('desktop setup apply failed', { error: error.message })
      const bootstrap = await this.setupRuntime.bootstrap()
      const preflight = {
        ready: false,
        config_path: this.paths.openaseConfigPath,
        openase_home_dir: this.paths.openaseHomeDir,
        issues: [this.mapSetupRuntimeError(error)],
      }
      await this.showSetup({ bootstrap, preflight })
      return this.getRuntimeState()
    }
  }

  mapSetupRuntimeError(error) {
    if (error?.code === 'setup_timeout') {
      return {
        code: 'setup_timeout',
        title: 'Setup timed out',
        message: error.message,
        action: 'Retry the setup and open the logs if the timeout repeats.',
      }
    }
    return {
      code: 'setup_failed',
      title: 'Setup failed',
      message: error?.message ?? 'Unknown desktop setup error.',
      action: 'Retry the setup or inspect the logs for more detail.',
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
