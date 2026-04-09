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
    stop: vi.fn(async () => {}),
  }
  const setupRuntime = {
    preflight: vi.fn(async () => ({
      ready: fs.existsSync(configPath),
      config_path: configPath,
      openase_home_dir: paths.openaseHomeDir,
      issues: fs.existsSync(configPath)
        ? []
        : [
            {
              code: 'config_missing',
              title: 'OpenASE config is missing',
              message: `No config file was found at ${configPath}.`,
            },
          ],
    })),
    bootstrap: vi.fn(async () => ({
      config_path: configPath,
      defaults: {
        manual_database: {
          host: '127.0.0.1',
          port: 5432,
          name: 'openase',
          user: 'openase',
          ssl_mode: 'disable',
        },
        docker_database: {
          container_name: 'openase-local-postgres',
          database_name: 'openase',
          user: 'openase',
          port: 15432,
          volume_name: 'openase-local-postgres-data',
          image: 'postgres:16-alpine',
        },
      },
      cli: [{ id: 'docker', name: 'Docker', command: 'docker', status: 'ready' }],
    })),
    apply: vi.fn(async () => {
      fs.writeFileSync(configPath, 'server:\n  mode: all-in-one\n')
      return {
        ready: true,
        config_path: configPath,
        openase_home_dir: paths.openaseHomeDir,
        env_path: path.join(paths.openaseHomeDir, '.env'),
        database_source: 'manual',
        issues: [],
      }
    }),
  }
  const controller = new DesktopAppController({
    app: { quit: vi.fn(), isPackaged: false },
    Menu: { setApplicationMenu: vi.fn(), buildFromTemplate: vi.fn(() => ({})) },
    ipcMain: { handle: vi.fn(), removeHandler: vi.fn() },
    shell: { openPath: vi.fn(async () => '') },
    service,
    setupRuntime,
    logger: { info: vi.fn(), error: vi.fn() },
    paths,
    version: '0.1.0',
    devServerURL: options.devServerURL,
    browserWindowFactory: () => window,
  })

  return { controller, service, setupRuntime, window, paths }
}

describe('DesktopAppController', () => {
  it('loads the local service URL after readiness in production mode', async () => {
    const { controller, window, setupRuntime } = createHarness()

    await controller.start()

    expect(setupRuntime.preflight).toHaveBeenCalledTimes(1)
    expect(window.loadFile).toHaveBeenCalledWith(expect.stringMatching(/loading\.html$/))
    expect(window.loadURL).toHaveBeenCalledWith('http://127.0.0.1:43127')
  })

  it('loads the Vite dev server when configured', async () => {
    const { controller, window } = createHarness({ devServerURL: 'http://127.0.0.1:4174' })

    await controller.start()

    expect(window.loadURL).toHaveBeenCalledWith('http://127.0.0.1:4174')
  })

  it('shows the setup page when the config file is missing', async () => {
    const { controller, window } = createHarness({ withConfig: false })

    await controller.start()

    expect(window.loadFile).toHaveBeenLastCalledWith(expect.stringMatching(/setup\.html$/))
    expect(controller.getRuntimeState().status).toBe('setup-required')
    expect(controller.getRuntimeState().lastError.code).toBe('config_missing')
  })

  it('applies setup and then loads the desktop surface', async () => {
    const { controller, service, setupRuntime, window } = createHarness({ withConfig: false })

    await controller.start()
    await controller.applySetup({
      database: {
        type: 'manual',
        manual: {
          host: '127.0.0.1',
          port: 5432,
          name: 'openase',
          user: 'openase',
          password: 'secret',
          ssl_mode: 'disable',
        },
      },
    })

    expect(setupRuntime.apply).toHaveBeenCalledWith(
      expect.objectContaining({
        allow_overwrite: true,
        database: expect.objectContaining({ type: 'manual' }),
      }),
    )
    expect(service.start).toHaveBeenCalledTimes(1)
    expect(window.loadURL).toHaveBeenCalledWith('http://127.0.0.1:43127')
  })

  it('rechecks setup through the startup pipeline', async () => {
    const { controller, setupRuntime } = createHarness({ withConfig: false })

    await controller.start()
    await controller.recheckSetup()

    expect(setupRuntime.preflight).toHaveBeenCalledTimes(2)
  })
})
