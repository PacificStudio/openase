const path = require('node:path')

const {
  buildOpenASELaunchSpec,
  resolveOpenASEBinary,
} = require('../../main/runtime/service-process')

describe('buildOpenASELaunchSpec', () => {
  it('assembles the all-in-one command with localhost binding', () => {
    const spec = buildOpenASELaunchSpec({
      env: {},
      binaryPath: '/tmp/openase',
      port: 43127,
      configPath: '/tmp/config.yaml',
    })

    expect(spec).toEqual({
      executablePath: '/tmp/openase',
      args: ['all-in-one', '--host', '127.0.0.1', '--port', '43127', '--config', '/tmp/config.yaml'],
    })
  })

  it('wraps JavaScript fixtures with the provided node binary', () => {
    const fixturePath = '/tmp/fake-openase.cjs'
    const spec = buildOpenASELaunchSpec({
      env: { OPENASE_DESKTOP_NODE_BINARY: '/usr/bin/node' },
      binaryPath: fixturePath,
      port: 43127,
      configPath: '/tmp/config.yaml',
    })

    expect(spec).toEqual({
      executablePath: '/usr/bin/node',
      args: [fixturePath, 'all-in-one', '--host', '127.0.0.1', '--port', '43127', '--config', '/tmp/config.yaml'],
    })
  })
})

describe('resolveOpenASEBinary', () => {
  it('prefers explicit environment overrides', () => {
    const target = path.resolve('/tmp/openase')

    expect(
      resolveOpenASEBinary({
        env: { OPENASE_DESKTOP_OPENASE_BIN: target },
        existsSync: (candidate) => candidate === target,
        resourcesPath: '/opt/resources',
      }),
    ).toBe(target)
  })
})
