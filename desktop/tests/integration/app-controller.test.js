const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')

const { DesktopAppController } = require('../../main/app-controller')
const { resolveDesktopPaths } = require('../../main/runtime/paths')

function createHarness(options = {}) {
  const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'openase-desktop-controller-'))
  const configPath = path.join(tempRoot, 'config.yaml')
  if (options.withConfig !== false) {
    fs.writeFileSync(configPath, 'server:\n  mode: all-in-one\n')
  }

  const paths = resolveDesktopPaths({
    env: { OPENASE_DESKTOP_OPENASE_CONFIG: configPath },
    homeDir: tempRoot,
    desktopUserDataDir: path.join(tempRoot, 'desktop-user-data'),
    desktopLogsDir: path.join(tempRoot, 'desktop-logs'),
  })
  const window = {
    show: vi.fn(),
    once: vi.fn((event, handler) => {
      if (event === 'ready-to-show') {
        handler()
      }
    }),
    on: vi.fn(),
    loadURL: vi.fn(async () => {}),
    loadFile: vi.fn(async () => {}),
    getBounds: vi.fn(() => ({ width: 1200, height: 800 })),
  }
  const service = {
    on: vi.fn(),
    start: vi.fn(async () => ({ baseURL: 'http://127.0.0.1:43127' })),
    restart: vi.fn(async () => ({ baseURL: 'http://127.0.0.1:43127' })),
    stop: vi.fn(async () => {}),
  }
  const controller = new DesktopAppController({
    app: { quit: vi.fn(), isPackaged: false },
    Menu: { setApplicationMenu: vi.fn(), buildFromTemplate: vi.fn(() => ({})) },
    ipcMain: { handle: vi.fn(), removeHandler: vi.fn() },
    shell: { openPath: vi.fn(async () => '') },
    service,
    logger: { info: vi.fn(), error: vi.fn() },
    paths,
    version: '0.1.0',
    devServerURL: options.devServerURL,
    browserWindowFactory: () => window,
  })

  return { controller, service, window, paths }
}

describe('DesktopAppController', () => {
  it('loads the local service URL after readiness in production mode', async () => {
    const { controller, window } = createHarness()

    await controller.start()

    expect(window.loadFile).toHaveBeenCalledWith(expect.stringMatching(/loading\.html$/))
    expect(window.loadURL).toHaveBeenCalledWith('http://127.0.0.1:43127')
  })

  it('loads the Vite dev server when configured', async () => {
    const { controller, window } = createHarness({ devServerURL: 'http://127.0.0.1:4174' })

    await controller.start()

    expect(window.loadURL).toHaveBeenCalledWith('http://127.0.0.1:4174')
  })

  it('shows the error page when the config file is missing', async () => {
    const { controller, window } = createHarness({ withConfig: false })

    await controller.start()

    expect(window.loadFile).toHaveBeenLastCalledWith(expect.stringMatching(/error\.html$/))
    expect(controller.getRuntimeState().lastError.code).toBe('config_missing')
  })

  it('restarts the service from the error surface actions', async () => {
    const { controller, service, window } = createHarness()

    await controller.start()
    await controller.restartService()

    expect(service.restart).toHaveBeenCalledTimes(1)
    expect(window.loadURL).toHaveBeenCalledWith('http://127.0.0.1:43127')
  })
})
