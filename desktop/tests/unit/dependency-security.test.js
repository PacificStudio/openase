const fs = require('node:fs')
const path = require('node:path')

const desktopRoot = path.resolve(__dirname, '../..')
const runtimeRoots = ['main', 'preload', 'scripts']
const minPatchedAxios = '1.15.1'

function listFiles(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  return entries.flatMap((entry) => {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      return listFiles(fullPath)
    }
    return [fullPath]
  })
}

function extractLockedVersions(lockfile, packageName) {
  return [...lockfile.matchAll(new RegExp(`^  ${packageName}@(\\d+\\.\\d+\\.\\d+):$`, 'gm'))].map(
    (match) => match[1],
  )
}

function compareSemver(left, right) {
  const leftParts = left.split('.').map(Number)
  const rightParts = right.split('.').map(Number)

  for (let index = 0; index < Math.max(leftParts.length, rightParts.length); index += 1) {
    const delta = (leftParts[index] ?? 0) - (rightParts[index] ?? 0)
    if (delta !== 0) {
      return delta
    }
  }

  return 0
}

describe('desktop dependency security guardrails', () => {
  it('pins redirect-following dependencies to patched versions', () => {
    const pkg = JSON.parse(fs.readFileSync(path.join(desktopRoot, 'package.json'), 'utf8'))
    expect(pkg.pnpm?.overrides?.axios).toBe('^1.15.1')
    expect(pkg.pnpm?.overrides?.['follow-redirects']).toBe('^1.16.0')

    const lockfile = fs.readFileSync(path.join(desktopRoot, 'pnpm-lock.yaml'), 'utf8')
    const axiosVersions = extractLockedVersions(lockfile, 'axios')

    expect(axiosVersions.length).toBeGreaterThan(0)
    expect(axiosVersions.every((version) => compareSemver(version, minPatchedAxios) >= 0)).toBe(true)
    expect(lockfile).not.toContain('follow-redirects@1.15.')
    expect(lockfile).toMatch(/follow-redirects@1\.16\./)
  })

  it('keeps desktop-owned code paths off custom axios redirect flows', () => {
    const packageJson = JSON.parse(fs.readFileSync(path.join(desktopRoot, 'package.json'), 'utf8'))
    expect(packageJson.scripts?.dev).toContain('wait-on http://127.0.0.1:4174')
    expect(packageJson.scripts?.dev).not.toMatch(/--header|--headers|authorization/i)

    for (const relativeRoot of runtimeRoots) {
      const files = listFiles(path.join(desktopRoot, relativeRoot))
      for (const file of files) {
        const source = fs.readFileSync(file, 'utf8')
        expect(source).not.toMatch(/\b(?:require\(|from\s+)['"]axios['"]/)
        expect(source).not.toMatch(/\b(?:require\(|from\s+)['"]follow-redirects['"]/)
      }
    }
  })
})
