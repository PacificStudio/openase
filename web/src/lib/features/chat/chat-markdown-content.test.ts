import { cleanup, render } from '@testing-library/svelte'
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
})
