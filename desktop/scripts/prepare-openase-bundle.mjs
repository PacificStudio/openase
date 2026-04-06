import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const desktopRoot = path.resolve(fileURLToPath(new URL('..', import.meta.url)))
const repoRoot = path.resolve(desktopRoot, '..')
const packageJson = JSON.parse(await fs.readFile(path.join(desktopRoot, 'package.json'), 'utf8'))
const bundleRoot = path.join(desktopRoot, '.bundle')
const bundleBinDir = path.join(bundleRoot, 'bin')
const bundleConfigDir = path.join(bundleRoot, 'config')
const bundleDocsDir = path.join(bundleRoot, 'docs')
const bundleBinaryPath = path.join(bundleBinDir, process.platform === 'win32' ? 'openase.exe' : 'openase')
const skipBuild = process.argv.includes('--skip-build')

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd ?? repoRoot,
    stdio: 'inherit',
    env: { ...process.env, ...(options.env ?? {}) },
  })

  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(' ')} failed with exit code ${result.status}`)
  }
}

await fs.mkdir(bundleBinDir, { recursive: true })
await fs.mkdir(bundleConfigDir, { recursive: true })
await fs.mkdir(bundleDocsDir, { recursive: true })

if (!skipBuild) {
  run('make', ['build-web', 'build', `VERSION=${packageJson.version}`, `OPENASE_BIN=${bundleBinaryPath}`], {
    cwd: repoRoot,
  })
}

await fs.copyFile(path.join(repoRoot, 'config.example.yaml'), path.join(bundleConfigDir, 'config.example.yaml'))
await fs.copyFile(path.join(repoRoot, 'docs', 'en', 'desktop-v1.md'), path.join(bundleDocsDir, 'desktop-v1.md'))
await fs.writeFile(
  path.join(bundleConfigDir, 'desktop-manifest.json'),
  JSON.stringify(
    {
      version: packageJson.version,
      built_at: new Date().toISOString(),
      openase_binary: path.basename(bundleBinaryPath),
    },
    null,
    2,
  ),
)
