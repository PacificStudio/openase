const os = require('node:os')
const path = require('node:path')

const { resolveDesktopPaths, resolveOpenASEHome } = require('../../main/runtime/paths')

describe('resolveOpenASEHome', () => {
  it('defaults to ~/.openase', () => {
    expect(resolveOpenASEHome({}, '/tmp/home')).toBe(path.join('/tmp/home', '.openase'))
  })

  it('allows desktop-specific home overrides', () => {
    expect(resolveOpenASEHome({ OPENASE_DESKTOP_OPENASE_HOME: '/tmp/custom-home' }, os.homedir())).toBe(
      '/tmp/custom-home',
    )
  })
})

describe('resolveDesktopPaths', () => {
  it('keeps desktop and openase directories separate', () => {
    const paths = resolveDesktopPaths({
      env: {},
      homeDir: '/tmp/home',
      desktopUserDataDir: '/tmp/app-data',
      desktopLogsDir: '/tmp/app-logs',
    })

    expect(paths.desktopUserDataDir).toBe('/tmp/app-data')
    expect(paths.desktopLogsDir).toBe('/tmp/app-logs')
    expect(paths.openaseHomeDir).toBe('/tmp/home/.openase')
    expect(paths.openaseConfigPath).toBe('/tmp/home/.openase/config.yaml')
    expect(paths.desktopServiceLogPath).toBe('/tmp/home/.openase/logs/desktop-service.log')
  })
})
