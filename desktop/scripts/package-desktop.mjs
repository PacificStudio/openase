import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const desktopRoot = path.resolve(fileURLToPath(new URL('..', import.meta.url)))
const builderArgs = ['pnpm', 'exec', 'electron-builder', '--config', 'packaging/electron-builder.yml']

if (process.argv.includes('--dir')) {
  builderArgs.push('--dir')
}

const prepare = spawnSync(process.execPath, ['scripts/prepare-openase-bundle.mjs'], {
  cwd: desktopRoot,
  stdio: 'inherit',
  env: process.env,
})

if (prepare.status !== 0) {
  throw new Error('desktop bundle preparation failed')
}

const result = spawnSync('corepack', builderArgs, {
  cwd: desktopRoot,
  stdio: 'inherit',
  env: process.env,
})

if (result.status !== 0) {
  throw new Error(`electron-builder failed with exit code ${result.status}`)
}
