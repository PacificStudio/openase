import { readdir } from 'node:fs/promises'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
import { JSDOM } from 'jsdom'

const timeoutMs = Number(process.env.MERMAID_RENDER_TIMEOUT_MS ?? '1500')
const mermaidPath = await resolveMermaidEntrypoint()
const { default: mermaid } = await import(pathToFileURL(mermaidPath).href)

const dom = new JSDOM('<div id="root"></div>', { pretendToBeVisual: true })
const { window } = dom

installDomGlobals(window)

mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: 'base' })

const hostileGanttChart = [
  'gantt',
  'dateFormat YYYY-MM-DD',
  'excludes weekends monday,tuesday,wednesday,thursday,friday',
  'section Demo',
  'Task :2026-05-12, 1d',
].join('\n')

const timeout = new Promise((_, reject) => {
  setTimeout(() => {
    reject(new Error(`mermaid.render timed out after ${timeoutMs}ms`))
  }, timeoutMs)
})

const outcome = await Promise.race([
  mermaid
    .render('test-graph', hostileGanttChart)
    .then((result) => {
      if (!result?.svg || !result.svg.includes('<svg')) {
        throw new Error('Expected mermaid to return SVG markup for the hostile gantt chart.')
      }
      return {
        status: 'rendered',
        svgLength: result.svg.length,
      }
    })
    .catch((error) => {
      return {
        status: 'rejected',
        errorMessage: error instanceof Error ? error.message : String(error),
      }
    }),
  timeout,
])

process.stdout.write(`${JSON.stringify({ mermaidPath, timeoutMs, ...outcome })}\n`)

function installDomGlobals(window) {
  class MockCSSStyleSheet {
    constructor() {
      this.cssRules = []
    }

    insertRule(rule, index = this.cssRules.length) {
      this.cssRules.splice(index, 0, { cssText: rule })
      return index
    }

    replaceSync(text) {
      this.cssRules = [{ cssText: text }]
    }
  }

  globalThis.window = window
  globalThis.document = window.document
  globalThis.HTMLElement = window.HTMLElement
  globalThis.SVGElement = window.SVGElement
  globalThis.Element = window.Element
  globalThis.DOMParser = window.DOMParser
  globalThis.CSSStyleSheet = MockCSSStyleSheet
  globalThis.getComputedStyle = window.getComputedStyle.bind(window)
  globalThis.requestAnimationFrame = window.requestAnimationFrame.bind(window)
  globalThis.cancelAnimationFrame = window.cancelAnimationFrame.bind(window)
  window.CSSStyleSheet = MockCSSStyleSheet

  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: window.navigator,
  })

  if (typeof SVGElement.prototype.getBBox !== 'function') {
    SVGElement.prototype.getBBox = () => ({ x: 0, y: 0, width: 100, height: 100 })
  }
}

async function resolveMermaidEntrypoint() {
  const pnpmStoreDir = path.join(process.cwd(), 'node_modules', '.pnpm')
  const storeEntries = await readdir(pnpmStoreDir)
  const streamdownStoreDir = storeEntries.find((entry) => entry.startsWith('streamdown-svelte@'))

  if (!streamdownStoreDir) {
    throw new Error(`Could not locate streamdown-svelte in ${pnpmStoreDir}.`)
  }

  return path.join(
    pnpmStoreDir,
    streamdownStoreDir,
    'node_modules',
    'mermaid',
    'dist',
    'mermaid.core.mjs',
  )
}
