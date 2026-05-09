const fs = require('node:fs')
const path = require('node:path')

const desktopRoot = path.resolve(__dirname, '../..')
const runtimeRoots = ['main', 'preload', 'scripts']
const PATCHED_MINIMUMS = {
  axios: [1, 15, 1],
  'follow-redirects': [1, 16, 0],
}

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

function parseVersionFloor(range) {
  const match = range.match(/(\d+)\.(\d+)\.(\d+)/)
  if (!match) {
    throw new Error(`Unable to parse package range: ${range}`)
  }
  return match.slice(1).map((part) => Number(part))
}

function versionAtLeast(version, minimum) {
  for (let index = 0; index < minimum.length; index += 1) {
    const current = version[index] ?? 0
    const required = minimum[index]
    if (current > required) {
      return true
    }
    if (current < required) {
      return false
    }
  }
  return true
}

function extractPackageVersions(lockfile, packageName) {
  const pattern = new RegExp(`${packageName}@(\\d+\\.\\d+\\.\\d+)`, 'g')
  const versions = new Set()
  for (const match of lockfile.matchAll(pattern)) {
    versions.add(match[1])
  }
  return [...versions].map((version) => version.split('.').map((part) => Number(part)))
}

describe('desktop dependency security guardrails', () => {
  it('pins redirect-following dependencies to patched versions', () => {
    const pkg = JSON.parse(fs.readFileSync(path.join(desktopRoot, 'package.json'), 'utf8'))
    expect(versionAtLeast(parseVersionFloor(pkg.pnpm?.overrides?.axios ?? ''), PATCHED_MINIMUMS.axios)).toBe(true)
    expect(versionAtLeast(parseVersionFloor(pkg.pnpm?.overrides?.['follow-redirects'] ?? ''), PATCHED_MINIMUMS['follow-redirects'])).toBe(true)

    const lockfile = fs.readFileSync(path.join(desktopRoot, 'pnpm-lock.yaml'), 'utf8')
    const axiosVersions = extractPackageVersions(lockfile, 'axios')
    const followRedirectVersions = extractPackageVersions(lockfile, 'follow-redirects')

    expect(axiosVersions.length).toBeGreaterThan(0)
    expect(followRedirectVersions.length).toBeGreaterThan(0)
    expect(axiosVersions.every((version) => versionAtLeast(version, PATCHED_MINIMUMS.axios))).toBe(true)
    expect(followRedirectVersions.every((version) => versionAtLeast(version, PATCHED_MINIMUMS['follow-redirects']))).toBe(true)
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
