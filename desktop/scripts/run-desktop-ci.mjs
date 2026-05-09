import { spawnSync } from 'node:child_process'

runPnpm(['run', 'test:unit'])
runPlaywright()
runPnpm(['run', 'package:smoke'])

function runPlaywright() {
  if (process.platform === 'linux' && !process.env.DISPLAY) {
    const xvfbCheck = spawnSync('sh', ['-lc', 'command -v xvfb-run >/dev/null 2>&1'])
    if (xvfbCheck.status === 0) {
      const { command, args } = resolvePnpm(['run', 'test:e2e'])
      run('xvfb-run', ['-a', command, ...args])
      return
    }
  }

  runPnpm(['run', 'test:e2e'])
}

function runPnpm(args) {
  const { command, args: resolvedArgs } = resolvePnpm(args)
  run(command, resolvedArgs)
}

function resolvePnpm(args) {
  const pnpmCommand = process.platform === 'win32' ? 'pnpm.cmd' : 'pnpm'
  const pnpmCheck = spawnSync(pnpmCommand, ['--version'], { stdio: 'ignore', env: process.env })
  if (pnpmCheck.status === 0) {
    return { command: pnpmCommand, args }
  }

  const corepackCommand = process.platform === 'win32' ? 'corepack.cmd' : 'corepack'
  return { command: corepackCommand, args: ['pnpm', ...args] }
}

function run(command, args) {
  const result = spawnSync(command, args, {
    stdio: 'inherit',
    env: process.env,
  })

  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(' ')} failed with exit code ${result.status}`)
  }
}
