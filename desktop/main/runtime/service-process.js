const { EventEmitter } = require('node:events')
const fs = require('node:fs')
const path = require('node:path')
const { spawn } = require('node:child_process')
const { setTimeout: delay } = require('node:timers/promises')

const { waitForHealthy } = require('./health-check')
const { findAvailablePort } = require('./port')

function isJavaScriptEntrypoint(targetPath) {
  return /\.(cjs|mjs|js)$/i.test(targetPath)
}

function resolveOpenASEBinary(options) {
  const env = options?.env ?? process.env
  const existsSync = options?.existsSync ?? fs.existsSync
  const resourcesPath = options?.resourcesPath ?? process.resourcesPath ?? path.resolve(__dirname, '..', '..', '..', '.bundle')
  const packagedCandidate = path.join(resourcesPath, 'bin', process.platform === 'win32' ? 'openase.exe' : 'openase')
  const repoCandidate = path.resolve(__dirname, '..', '..', '..', 'bin', process.platform === 'win32' ? 'openase.exe' : 'openase')
  const candidates = [
    env.OPENASE_DESKTOP_OPENASE_BIN ? path.resolve(env.OPENASE_DESKTOP_OPENASE_BIN) : '',
    packagedCandidate,
    repoCandidate,
  ].filter(Boolean)

  for (const candidate of candidates) {
    if (existsSync(candidate)) {
      return candidate
    }
  }

  throw new Error(
    `unable to locate the OpenASE binary; checked ${candidates.join(', ')}. Run \`make build\` or set OPENASE_DESKTOP_OPENASE_BIN.`,
  )
}

function buildOpenASELaunchSpec(options) {
  const env = options?.env ?? process.env
  const binaryPath = options.binaryPath
  const port = options.port
  const configPath = options.configPath
  const commandArgs = Array.isArray(options.args) && options.args.length > 0
    ? [...options.args]
    : ['all-in-one', '--host', '127.0.0.1', '--port', String(port), '--config', configPath]
  let executablePath = binaryPath
  let args = commandArgs

  if (isJavaScriptEntrypoint(binaryPath)) {
    const nodeBinary = env.OPENASE_DESKTOP_NODE_BINARY
    if (!nodeBinary) {
      throw new Error('OPENASE_DESKTOP_NODE_BINARY is required when OPENASE_DESKTOP_OPENASE_BIN points to a JavaScript fixture')
    }
    executablePath = nodeBinary
    args = [binaryPath, ...args]
  }

  return {
    executablePath,
    args,
  }
}

function buildOpenASEServiceEnvironment(options) {
  const env = options?.env ?? process.env
  const port = options?.port

  return {
    ...env,
    OPENASE_SERVER_HOST: '127.0.0.1',
    OPENASE_SERVER_PORT: String(port),
    // Desktop owns the local loopback runtime, so skip the browser bootstrap/login gate.
    OPENASE_DESKTOP_DISABLE_AUTH: '1',
  }
}

class OpenASEServiceProcess extends EventEmitter {
  constructor(options) {
    super()
    this.env = options.env ?? process.env
    this.paths = options.paths
    this.logger = options.logger
    this.fetchImpl = options.fetchImpl
    this.spawnImpl = options.spawnImpl ?? spawn
    this.findPort = options.findPort ?? findAvailablePort
    this.waitForHealthy = options.waitForHealthy ?? waitForHealthy
    this.maxAttempts = options.maxAttempts ?? 2
    this.healthTimeoutMs = options.healthTimeoutMs ?? 45_000
    this.child = null
    this.baseURL = null
    this.currentPort = null
    this.ready = false
    this.stdoutStream = null
    this.stderrStream = null
    this.stopping = false
  }

  async start() {
    let lastError = null

    for (let attempt = 1; attempt <= this.maxAttempts; attempt += 1) {
      try {
        return await this.startAttempt(attempt)
      } catch (error) {
        lastError = error
        this.logger.error('desktop service start failed', {
          attempt,
          error: error.message,
        })
        await this.stop()
        if (attempt < this.maxAttempts) {
          await delay(500)
        }
      }
    }

    throw lastError ?? new Error('desktop service failed without an explicit error')
  }

  async startAttempt(attempt) {
    const port = await this.findPort('127.0.0.1')
    const binaryPath = resolveOpenASEBinary({ env: this.env })
    const launchSpec = buildOpenASELaunchSpec({
      env: this.env,
      binaryPath,
      port,
      configPath: this.paths.openaseConfigPath,
    })

    fs.mkdirSync(this.paths.openaseLogsDir, { recursive: true })
    this.stdoutStream = fs.createWriteStream(this.paths.desktopServiceLogPath, { flags: 'a' })
    this.stderrStream = fs.createWriteStream(this.paths.desktopServiceStderrPath, { flags: 'a' })

    this.logger.info('starting desktop service', {
      attempt,
      executablePath: launchSpec.executablePath,
      args: launchSpec.args,
      port,
      configPath: this.paths.openaseConfigPath,
    })

    const child = this.spawnImpl(launchSpec.executablePath, launchSpec.args, {
      env: buildOpenASEServiceEnvironment({
        env: this.env,
        port,
      }),
      stdio: ['ignore', 'pipe', 'pipe'],
    })

    child.stdout?.pipe(this.stdoutStream)
    child.stderr?.pipe(this.stderrStream)

    const childErrorPromise = new Promise((_, reject) => {
      child.once('error', (error) => {
        this.logger.error('desktop service spawn error', {
          error: error.message,
          executablePath: launchSpec.executablePath,
        })
        reject(error)
      })
    })

    child.once('exit', (code, signal) => {
      const metadata = { code, signal, ready: this.ready, stopping: this.stopping }
      this.logger.info('desktop service exited', metadata)
      this.child = null
      this.ready = false
      this.baseURL = null
      if (!this.stopping) {
        this.emit('exit', metadata)
      }
    })

    this.child = child
    this.currentPort = port
    this.baseURL = `http://127.0.0.1:${port}`
    this.ready = false
    this.stopping = false

    await Promise.race([
      this.waitForHealthy({
        baseURL: this.baseURL,
        timeoutMs: this.healthTimeoutMs,
        fetchImpl: this.fetchImpl,
      }),
      childErrorPromise,
    ])

    this.ready = true
    this.emit('ready', {
      baseURL: this.baseURL,
      port,
      logPath: this.paths.desktopServiceLogPath,
    })

    return {
      baseURL: this.baseURL,
      port,
      logPath: this.paths.desktopServiceLogPath,
    }
  }

  async restart() {
    await this.stop()
    return this.start()
  }

  async stop() {
    this.stopping = true

    if (!this.child) {
      this.closeStreams()
      return
    }

    const child = this.child

    await new Promise((resolve) => {
      let settled = false
      const finalize = () => {
        if (settled) {
          return
        }
        settled = true
        resolve()
      }

      const timeout = setTimeout(() => {
        child.kill('SIGKILL')
      }, 10_000)

      child.once('exit', () => {
        clearTimeout(timeout)
        finalize()
      })

      child.kill('SIGTERM')
    })

    this.child = null
    this.ready = false
    this.baseURL = null
    this.currentPort = null
    this.closeStreams()
  }

  closeStreams() {
    if (this.stdoutStream) {
      this.stdoutStream.end()
      this.stdoutStream = null
    }
    if (this.stderrStream) {
      this.stderrStream.end()
      this.stderrStream = null
    }
  }
}

module.exports = {
  OpenASEServiceProcess,
  buildOpenASEServiceEnvironment,
  buildOpenASELaunchSpec,
  resolveOpenASEBinary,
}
