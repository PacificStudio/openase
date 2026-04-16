import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterAll, afterEach, describe, expect, it } from 'vitest'
import { ProjectUpdateMarkdownContent } from '$lib/features/project-updates'
import { TicketMarkdownContent } from '$lib/features/ticket-detail'
import { Streamdown } from 'streamdown-svelte'
import type { DiagramPlugin, MermaidInstance } from 'streamdown-svelte'
import MarkdownContent from './markdown-content.svelte'

class MockIntersectionObserver implements IntersectionObserver {
  readonly root = null
  readonly rootMargin = '0px'
  readonly thresholds = [0]

  constructor(private readonly callback: IntersectionObserverCallback) {}

  disconnect(): void {}

  observe(target: Element): void {
    this.callback(
      [
        {
          boundingClientRect: target.getBoundingClientRect(),
          intersectionRatio: 1,
          intersectionRect: target.getBoundingClientRect(),
          isIntersecting: true,
          rootBounds: null,
          target,
          time: 0,
        },
      ],
      this,
    )
  }

  takeRecords(): IntersectionObserverEntry[] {
    return []
  }

  unobserve(): void {}
}

function createAdvisoryMermaidPlugin(): DiagramPlugin {
  return {
    language: 'mermaid',
    name: 'mermaid',
    type: 'diagram',
    getMermaid() {
      const mermaid: MermaidInstance = {
        initialize() {},
        async render(_id: string, _source: string) {
          const { default: createDOMPurify } =
            await import('../../../../node_modules/.pnpm/dompurify@3.4.0/node_modules/dompurify/dist/purify.es.mjs')
          const domPurify = createDOMPurify(window)
          const svg = domPurify.sanitize(
            [
              '<svg xmlns="http://www.w3.org/2000/svg">',
              '  <foreignObject>',
              '    <div xmlns="http://www.w3.org/1999/xhtml">',
              '      <style>body{background:red}</style>',
              '      <span>safe label</span>',
              '    </div>',
              '  </foreignObject>',
              '</svg>',
            ].join('\n'),
            {
              ADD_TAGS: (tagName: string) => tagName === 'foreignobject',
              FORBID_TAGS: ['style'],
              HTML_INTEGRATION_POINTS: { foreignobject: true },
            },
          )

          return { svg }
        },
      }

      return mermaid
    },
  }
}

const originalIntersectionObserver = globalThis.IntersectionObserver
type SVGElementWithBBox = SVGElement & { getBBox: () => DOMRect }

const svgElementPrototype = globalThis.SVGElement?.prototype as SVGElementWithBBox | undefined
const originalGetBBox = svgElementPrototype?.getBBox

globalThis.IntersectionObserver = MockIntersectionObserver

if (svgElementPrototype && typeof svgElementPrototype.getBBox !== 'function') {
  svgElementPrototype.getBBox = () => DOMRect.fromRect({ x: 0, y: 0, width: 120, height: 24 })
}

afterAll(() => {
  globalThis.IntersectionObserver = originalIntersectionObserver

  if (svgElementPrototype) {
    if (originalGetBBox) {
      svgElementPrototype.getBBox = originalGetBBox
    } else {
      delete (svgElementPrototype as Partial<SVGElementWithBBox>).getBBox
    }
  }
})

describe('MarkdownContent policy', () => {
  afterEach(() => {
    cleanup()
  })

  it('allows safe links while blocking unsafe links and images', async () => {
    const { getByRole, queryByRole } = render(MarkdownContent, {
      props: {
        source: [
          '[Docs](https://example.com/docs)',
          '[Mail](mailto:help@example.com)',
          '[Relative](/tickets/ase-101)',
          '[Anchor](#policy)',
          '[Bad](javascript:alert(1))',
          '![Diagram](https://example.com/diagram.png)',
        ].join(' '),
      },
    })

    const docsLink = getByRole('link', { name: 'Docs' })
    const mailLink = getByRole('link', { name: 'Mail' })
    const relativeLink = getByRole('link', { name: 'Relative' })
    const anchorLink = getByRole('link', { name: 'Anchor' })

    expect(docsLink.getAttribute('href')).toBe('https://example.com/docs')
    expect(docsLink.getAttribute('target')).toBe('_blank')
    expect(docsLink.getAttribute('rel')).toBe('noopener noreferrer')
    expect(mailLink.getAttribute('href')).toBe('mailto:help@example.com')
    expect(mailLink.getAttribute('target')).toBe('_blank')
    expect(relativeLink.getAttribute('href')).toBe('/tickets/ase-101')
    expect(relativeLink.getAttribute('target')).toBeNull()
    expect(anchorLink.getAttribute('href')).toBe('#policy')
    expect(anchorLink.getAttribute('target')).toBeNull()
    expect(queryByRole('link', { name: /Bad/i })).toBeNull()
    expect(queryByRole('img', { name: /Diagram/i })).toBeNull()
  })

  it('drops raw html content at the renderer boundary', () => {
    const { container, getByText, queryByText } = render(MarkdownContent, {
      props: {
        source: 'Safe paragraph.\n\n<div>unsafe html</div>\n\n<script>alert(1)</script>',
      },
    })

    expect(getByText('Safe paragraph.')).toBeTruthy()
    expect(queryByText('unsafe html')).toBeNull()
    expect(container.querySelector('script')).toBeNull()
  })

  it('keeps mermaid fences on the non-rendering fallback path', async () => {
    const { container } = render(MarkdownContent, {
      props: {
        source: ['```mermaid', 'graph TD', '  A --> B', '```'].join('\n'),
      },
    })

    await waitFor(() => {
      expect(container.querySelector('[data-streamdown-mermaid]')).toBeTruthy()
    })

    const fallbackMermaid = container.querySelector('[data-streamdown-mermaid]')

    expect(fallbackMermaid?.textContent).toContain('graph TD')
    expect(fallbackMermaid?.querySelector('pre code')).toBeTruthy()
    expect(container.querySelector('[data-mermaid-svg]')).toBeNull()
  })
})

describe('Streamdown mermaid security', () => {
  afterEach(() => {
    cleanup()
  })

  it('preserves FORBID_TAGS when mermaid-style sanitization adds foreignObject tags', async () => {
    const { container } = render(Streamdown, {
      props: {
        content: ['```mermaid', 'graph TD', '  A --> B', '```'].join('\n'),
        mode: 'static',
        plugins: {
          mermaid: createAdvisoryMermaidPlugin(),
        },
      },
    })

    await waitFor(() => {
      expect(container.querySelector('[data-mermaid-svg]')).toBeTruthy()
    })

    const renderedMermaid = container.querySelector('[data-mermaid-svg]')

    expect(renderedMermaid?.querySelector('style')).toBeNull()
    expect(renderedMermaid?.querySelector('foreignObject')).toBeTruthy()
    expect(renderedMermaid?.textContent).toContain('safe label')
  })
})

describe.each([
  ['ticket detail', TicketMarkdownContent],
  ['project updates', ProjectUpdateMarkdownContent],
])('%s markdown surfaces', (_label, Component) => {
  afterEach(() => {
    cleanup()
  })

  it('renders tables, blockquotes, lists, and code blocks with overflow containers', async () => {
    const { container } = render(Component, {
      props: {
        source: [
          '> Ship after the smoke test passes.',
          '',
          '- capture screenshots',
          '- confirm mobile overflow',
          '',
          '| Step | Status |',
          '| --- | --- |',
          '| Build | Green |',
          '',
          '```ts',
          'console.log("ship it")',
          '```',
        ].join('\n'),
      },
    })

    await waitFor(() => {
      expect(container.querySelector('blockquote')).toBeTruthy()
      expect(container.querySelector('ul')).toBeTruthy()
      expect(container.querySelector('[data-streamdown="table"]')).toBeTruthy()
      expect(container.querySelector('[data-streamdown="code-block"]')).toBeTruthy()
    })

    const tableContainer = container.querySelector('[data-streamdown-table]')
    const codeBody = container.querySelector('[data-streamdown="code-block-body"]')

    expect(tableContainer?.className).toContain('overflow-x-auto')
    expect(codeBody?.className).toContain('overflow-x-auto')
  })
})
