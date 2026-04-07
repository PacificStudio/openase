const { spawn } = require('node:child_process')

const { buildOpenASELaunchSpec, resolveOpenASEBinary } = require('./service-process')

class OpenASESetupRuntime {
  constructor(options) {
    this.env = options.env ?? process.env
    this.paths = options.paths
    this.logger = options.logger
    this.spawnImpl = options.spawnImpl ?? spawn
    this.commandTimeoutMs = options.commandTimeoutMs ?? 120_000
  }

  async preflight() {
    return this.runJSON(['setup', 'desktop', 'preflight'])
  }

  async bootstrap() {
    return this.runJSON(['setup', 'desktop', 'bootstrap'])
  }

  async apply(request) {
    return this.runJSON(['setup', 'desktop', 'apply', '--input', '-'], JSON.stringify(request))
  }

  async runJSON(args, stdinBody = '') {
    const binaryPath = resolveOpenASEBinary({ env: this.env })
    const launchSpec = buildOpenASELaunchSpec({
      env: this.env,
      binaryPath,
      port: 0,
      configPath: this.paths.openaseConfigPath,
      args,
    })
    const env = {
      ...this.env,
      OPENASE_SETUP_HOME: this.paths.openaseHomeDir,
      OPENASE_SETUP_CONFIG_PATH: this.paths.openaseConfigPath,
    }

    this.logger.info('running desktop setup command', {
      executablePath: launchSpec.executablePath,
      args: launchSpec.args,
      configPath: this.paths.openaseConfigPath,
      openaseHomeDir: this.paths.openaseHomeDir,
    })

    const child = this.spawnImpl(launchSpec.executablePath, launchSpec.args, {
      env,
      stdio: ['pipe', 'pipe', 'pipe'],
    })

    let stdout = ''
    let stderr = ''
    child.stdout?.on('data', (chunk) => {
      stdout += chunk.toString()
    })
    child.stderr?.on('data', (chunk) => {
      stderr += chunk.toString()
    })

    const timeout = setTimeout(() => {
      child.kill('SIGKILL')
    }, this.commandTimeoutMs)

    return new Promise((resolve, reject) => {
      child.once('error', (error) => {
        clearTimeout(timeout)
        reject(error)
      })

      child.once('close', (code, signal) => {
        clearTimeout(timeout)
        if (signal === 'SIGKILL') {
          const error = new Error(`desktop setup command timed out after ${this.commandTimeoutMs}ms`)
          error.code = 'setup_timeout'
          reject(error)
          return
        }
        if (code !== 0) {
          reject(new Error((stderr || stdout || `desktop setup command exited with code ${code}`).trim()))
          return
        }
        try {
          resolve(JSON.parse(stdout))
        } catch (error) {
          reject(new Error(`decode desktop setup response: ${error.message}`))
        }
      })

      if (stdinBody) {
        child.stdin?.write(stdinBody)
      }
      child.stdin?.end()
    })
  }
}

module.exports = {
  OpenASESetupRuntime,
}
