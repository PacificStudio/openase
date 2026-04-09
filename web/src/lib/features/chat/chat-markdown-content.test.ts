import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'
import ChatMarkdownContent from './chat-markdown-content.svelte'

describe('ChatMarkdownContent', () => {
  afterEach(() => {
    cleanup()
  })

  it('rerenders when the markdown source changes on the same component instance', async () => {
    const { findByText, rerender } = render(ChatMarkdownContent, {
      props: {
        source: 'Initial assistant reply.',
      },
    })

    expect(await findByText('Initial assistant reply.')).toBeTruthy()

    await rerender({
      source: 'Updated assistant reply.',
    })

    expect(await findByText('Updated assistant reply.')).toBeTruthy()
  })

  it('keeps incomplete fenced code renderable while streaming', async () => {
    const { container, rerender } = render(ChatMarkdownContent, {
      props: {
        source: '```ts\nconsole.log(42)',
      },
    })

    await waitFor(() => {
      expect(container.querySelector('[data-streamdown="code-block"]')).toBeTruthy()
      expect(container.textContent).toContain('console.log(42)')
    })

    await rerender({
      source: '```ts\nconsole.log(42)\n```\n\nStreaming finished.',
    })

    await waitFor(() => {
      expect(container.textContent).toContain('Streaming finished.')
    })
  })
})
