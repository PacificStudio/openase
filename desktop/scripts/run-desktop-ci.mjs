import { spawnSync } from 'node:child_process'

run('corepack', ['pnpm', 'run', 'test:unit'])
runPlaywright()
run('corepack', ['pnpm', 'run', 'package:smoke'])

function runPlaywright() {
  if (process.platform === 'linux' && !process.env.DISPLAY) {
    const xvfbCheck = spawnSync('sh', ['-lc', 'command -v xvfb-run >/dev/null 2>&1'])
    if (xvfbCheck.status === 0) {
      run('xvfb-run', ['-a', 'corepack', 'pnpm', 'run', 'test:e2e'])
      return
    }
  }

  run('corepack', ['pnpm', 'run', 'test:e2e'])
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
