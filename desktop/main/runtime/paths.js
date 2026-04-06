const os = require('node:os')
const path = require('node:path')

function resolveOpenASEHome(env = process.env, homeDir = os.homedir()) {
  if (env.OPENASE_DESKTOP_OPENASE_HOME) {
    return path.resolve(env.OPENASE_DESKTOP_OPENASE_HOME)
  }

  return path.join(homeDir, '.openase')
}

function resolveDesktopPaths(options) {
  const env = options?.env ?? process.env
  const homeDir = options?.homeDir ?? os.homedir()
  const desktopUserDataDir = path.resolve(options.desktopUserDataDir)
  const desktopLogsDir = path.resolve(options.desktopLogsDir ?? path.join(desktopUserDataDir, 'logs'))
  const openaseHomeDir = resolveOpenASEHome(env, homeDir)
  const openaseConfigPath = env.OPENASE_DESKTOP_OPENASE_CONFIG
    ? path.resolve(env.OPENASE_DESKTOP_OPENASE_CONFIG)
    : path.join(openaseHomeDir, 'config.yaml')

  return {
    desktopUserDataDir,
    desktopLogsDir,
    desktopRuntimeDir: path.join(desktopUserDataDir, 'runtime'),
    desktopStatePath: path.join(desktopUserDataDir, 'window-state.json'),
    desktopHostLogPath: path.join(desktopLogsDir, 'desktop-host.log'),
    openaseHomeDir,
    openaseConfigPath,
    openaseLogsDir: path.join(openaseHomeDir, 'logs'),
    desktopServiceLogPath: path.join(openaseHomeDir, 'logs', 'desktop-service.log'),
    desktopServiceStderrPath: path.join(openaseHomeDir, 'logs', 'desktop-service.stderr.log'),
  }
}

module.exports = {
  resolveDesktopPaths,
  resolveOpenASEHome,
}
