import { createRequire } from 'node:module'
import { describe, expect, it } from 'vitest'

const require = createRequire(import.meta.url)

function loadVitePostcss() {
  const viteEntry = require.resolve('vite')
  const postcssEntry = require.resolve('postcss', { paths: [viteEntry] })
  const postcssPackageJsonEntry = require.resolve('postcss/package.json', { paths: [viteEntry] })
  const postcss = require(postcssEntry) as {
    parse: (source: string) => { toString: () => string }
  }
  const { version } = require(postcssPackageJsonEntry) as { version: string }

  return { postcss, version }
}

describe('Vite PostCSS stringifier security', () => {
  it('resolves the patched postcss release from the Vite dependency graph', () => {
    const { version } = loadVitePostcss()

    expect(version).toBe('8.5.10')
  })

  it.each([
    '.x{content:"</style><script>alert(1)</script>"}',
    '@font-face{font-family:"</style><script>alert(1)</script>"}',
    '/* </style><script>alert(1)</script> */ .x{color:red}',
  ])('escapes raw closing style tags when stringifying %s', (source) => {
    const { postcss } = loadVitePostcss()

    const serialized = postcss.parse(source).toString()

    expect(serialized).not.toContain('</style>')
    expect(serialized).toContain('\\3c /style>')
  })
})
