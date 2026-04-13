const fs = require('node:fs')
const path = require('node:path')

const desktopRoot = path.resolve(__dirname, '../..')

describe('desktop dependency security guardrails', () => {
  it('pins transitive axios to a patched version', () => {
    const pkg = JSON.parse(fs.readFileSync(path.join(desktopRoot, 'package.json'), 'utf8'))
    expect(pkg.pnpm?.overrides?.axios).toBe('^1.15.0')

    const lockfile = fs.readFileSync(path.join(desktopRoot, 'pnpm-lock.yaml'), 'utf8')
    expect(lockfile).not.toContain('axios@1.14.0')
    expect(lockfile).toMatch(/axios@1\.15\./)
  })
})
