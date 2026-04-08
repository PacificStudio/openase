import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'
import { ProjectUpdateMarkdownContent } from '$lib/features/project-updates'
import { TicketMarkdownContent } from '$lib/features/ticket-detail'
import MarkdownContent from './markdown-content.svelte'

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
