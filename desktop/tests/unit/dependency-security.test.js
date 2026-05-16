const fs = require('node:fs')
const path = require('node:path')

const desktopRoot = path.resolve(__dirname, '../..')
const runtimeRoots = ['main', 'preload', 'scripts']

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

describe('desktop dependency security guardrails', () => {
  it('pins redirect-following dependencies to patched versions', () => {
    const pkg = JSON.parse(fs.readFileSync(path.join(desktopRoot, 'package.json'), 'utf8'))
    expect(pkg.pnpm?.overrides?.['@xmldom/xmldom']).toBe('^0.8.13')
    expect(pkg.pnpm?.overrides?.axios).toBe('1.15.1')
    expect(pkg.pnpm?.overrides?.['follow-redirects']).toBe('^1.16.0')
    expect(pkg.pnpm?.overrides?.['ip-address']).toBe('10.2.0')
    expect(pkg.pnpm?.overrides?.postcss).toBe('^8.5.10')

    const lockfile = fs.readFileSync(path.join(desktopRoot, 'pnpm-lock.yaml'), 'utf8')
    expect(lockfile).not.toContain('@xmldom/xmldom@0.8.12')
    expect(lockfile).toContain('@xmldom/xmldom@0.8.13')
    expect(lockfile).not.toContain('axios@1.15.0')
    expect(lockfile).toContain('axios@1.15.1')
    expect(lockfile).not.toContain('follow-redirects@1.15.')
    expect(lockfile).toMatch(/follow-redirects@1\.16\./)
    expect(lockfile).not.toContain('ip-address@10.1.0')
    expect(lockfile).toContain('ip-address@10.2.0')
    expect(lockfile).not.toContain('postcss@8.5.9')
    expect(lockfile).toMatch(/postcss@8\.5\.(?:1\d|[2-9]\d)/)
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
        expect(source).not.toMatch(/\b(?:require\(|from\s+)['"]ip-address['"]/)
      }
    }
  })
})
