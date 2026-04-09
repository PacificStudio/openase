import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const desktopRoot = path.resolve(fileURLToPath(new URL('..', import.meta.url)))
const outputRoot = path.join(desktopRoot, 'dist', 'packages')

const build = spawnSync(process.execPath, ['scripts/package-desktop.mjs', '--dir'], {
  cwd: desktopRoot,
  stdio: 'inherit',
  env: process.env,
})

if (build.status !== 0) {
  throw new Error('desktop package smoke build failed')
}

const resourcesRoot = findResourceRoot(outputRoot)
const bundledBinary = path.join(resourcesRoot, 'bin', process.platform === 'win32' ? 'openase.exe' : 'openase')
const bundledConfig = path.join(resourcesRoot, 'config', 'config.example.yaml')
const bundledManifest = path.join(resourcesRoot, 'config', 'desktop-manifest.json')
const bundledGuide = path.join(resourcesRoot, 'docs', 'desktop-v1.md')

for (const requiredPath of [bundledBinary, bundledConfig, bundledManifest, bundledGuide]) {
  if (!fs.existsSync(requiredPath)) {
    throw new Error(`desktop package smoke missing ${requiredPath}`)
  }
}

console.log(`desktop package smoke passed: ${resourcesRoot}`)

function findResourceRoot(outputRootPath) {
  const entries = fs.readdirSync(outputRootPath, { withFileTypes: true })

  for (const entry of entries) {
    if (!entry.isDirectory()) {
      continue
    }
    const candidate = path.join(outputRootPath, entry.name)
    const linuxResources = path.join(candidate, 'resources')
    const macResources = path.join(candidate, 'OpenASE.app', 'Contents', 'Resources')
    if (fs.existsSync(linuxResources)) {
      return linuxResources
    }
    if (fs.existsSync(macResources)) {
      return macResources
    }
  }

  throw new Error(`unable to locate desktop package resources under ${outputRootPath}`)
}
