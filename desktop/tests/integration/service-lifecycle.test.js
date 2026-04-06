const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')

const { createFileLogger } = require('../../main/runtime/logger')
const { resolveDesktopPaths } = require('../../main/runtime/paths')
const { OpenASEServiceProcess } = require('../../main/runtime/service-process')

const fixtureBinary = path.resolve(__dirname, '..', 'fixtures', 'fake-openase.cjs')
const processes = []

afterEach(async () => {
  while (processes.length > 0) {
    const service = processes.pop()
    await service.stop()
  }
})

describe('OpenASEServiceProcess', () => {
  it('starts the local service, waits for readiness, and writes logs', async () => {
    const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'openase-desktop-service-'))
    const configPath = path.join(tempRoot, 'config.yaml')
    fs.writeFileSync(configPath, 'server:\n  mode: all-in-one\n')

    const paths = resolveDesktopPaths({
      env: {
        OPENASE_DESKTOP_OPENASE_BIN: fixtureBinary,
        OPENASE_DESKTOP_NODE_BINARY: process.execPath,
        OPENASE_DESKTOP_OPENASE_CONFIG: configPath,
        OPENASE_DESKTOP_OPENASE_HOME: path.join(tempRoot, '.openase-home'),
      },
      homeDir: tempRoot,
      desktopUserDataDir: path.join(tempRoot, 'desktop-user-data'),
      desktopLogsDir: path.join(tempRoot, 'desktop-logs'),
    })
    const logger = createFileLogger(path.join(tempRoot, 'host.log'))
    const service = new OpenASEServiceProcess({
      env: {
        OPENASE_DESKTOP_OPENASE_BIN: fixtureBinary,
        OPENASE_DESKTOP_NODE_BINARY: process.execPath,
        OPENASE_DESKTOP_OPENASE_CONFIG: configPath,
      },
      paths,
      logger,
      healthTimeoutMs: 5_000,
    })
    processes.push(service)

    const started = await service.start()
    const healthResponse = await fetch(`${started.baseURL}/healthz`)
    const payload = await healthResponse.json()

    expect(payload.status).toBe('ok')
    expect(fs.existsSync(paths.desktopServiceLogPath)).toBe(true)
  })

  it('emits an exit event when the child terminates after readiness', async () => {
    const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'openase-desktop-service-exit-'))
    const configPath = path.join(tempRoot, 'config.yaml')
    fs.writeFileSync(configPath, 'server:\n  mode: all-in-one\n')

    const env = {
      OPENASE_DESKTOP_OPENASE_BIN: fixtureBinary,
      OPENASE_DESKTOP_NODE_BINARY: process.execPath,
      OPENASE_DESKTOP_OPENASE_CONFIG: configPath,
      FAKE_OPENASE_EXIT_AFTER_READY_MS: '500',
    }
    const paths = resolveDesktopPaths({
      env,
      homeDir: tempRoot,
      desktopUserDataDir: path.join(tempRoot, 'desktop-user-data'),
      desktopLogsDir: path.join(tempRoot, 'desktop-logs'),
    })
    const logger = createFileLogger(path.join(tempRoot, 'host.log'))
    const service = new OpenASEServiceProcess({
      env,
      paths,
      logger,
      healthTimeoutMs: 5_000,
    })
    processes.push(service)

    const exitEvent = new Promise((resolve) => {
      service.once('exit', resolve)
    })

    await service.start()
    const metadata = await exitEvent

    expect(metadata.ready).toBe(true)
  })
})
