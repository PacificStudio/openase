import { beforeEach, describe, expect, it } from 'vitest'
import { renderMarkdown } from './markdown'

describe('renderMarkdown', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders paragraphs, lists, code fences, and links', () => {
    const html = renderMarkdown(`Review notes:

- keep comments separate
- support markdown

\`\`\`
go test ./...
\`\`\`

[OpenASE](https://github.com/PacificStudio/openase)`)

    expect(html).toContain('<p>Review notes:</p>')
    expect(html).toContain('<ul>')
    expect(html).toContain('<li>keep comments separate</li>')
    expect(html).toContain('<pre><code>go test ./...')
    expect(html).toContain('href="https://github.com/PacificStudio/openase"')
  })

  it('escapes unsafe html and strips unsafe links', () => {
    const html = renderMarkdown('<script>alert(1)</script> [bad](javascript:alert(1))')

    expect(html).toContain('alert(1)')
    expect(html).not.toContain('<script>')
    expect(html).not.toContain('href="javascript:alert(1)"')
  })
})
