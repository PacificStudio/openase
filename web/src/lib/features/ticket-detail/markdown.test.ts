import { describe, expect, it } from 'vitest'
import { renderTicketCommentMarkdown } from './markdown'

describe('renderTicketCommentMarkdown', () => {
  it('renders paragraphs, lists, code fences, and links', () => {
    const html = renderTicketCommentMarkdown(`Review notes:

- keep comments separate
- support markdown

\`\`\`
go test ./...
\`\`\`

[OpenASE](https://github.com/BetterAndBetterII/openase)`)

    expect(html).toContain('<p>Review notes:</p>')
    expect(html).toContain('<ul><li>keep comments separate</li><li>support markdown</li></ul>')
    expect(html).toContain('<pre><code>go test ./...</code></pre>')
    expect(html).toContain('href="https://github.com/BetterAndBetterII/openase"')
  })

  it('escapes unsafe html and strips unsafe links', () => {
    const html = renderTicketCommentMarkdown('<script>alert(1)</script> [bad](javascript:alert(1))')

    expect(html).toContain('&lt;script&gt;alert(1)&lt;/script&gt;')
    expect(html).not.toContain('javascript:alert(1)')
  })
})
