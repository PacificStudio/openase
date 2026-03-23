function escapeHTML(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

function sanitizeLink(rawHref: string) {
  const href = rawHref.trim()
  if (href.startsWith('http://') || href.startsWith('https://') || href.startsWith('mailto:')) {
    return href
  }
  if (href.startsWith('/') || href.startsWith('#')) {
    return href
  }
  return ''
}

function renderInline(markdown: string) {
  let html = escapeHTML(markdown)
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>')
  html = html.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_match, label: string, href: string) => {
    const safeHref = sanitizeLink(href)
    if (!safeHref) {
      return escapeHTML(label)
    }
    return `<a href="${escapeHTML(safeHref)}" rel="noreferrer" target="_blank">${escapeHTML(label)}</a>`
  })
  return html
}

function flushParagraph(paragraphLines: string[], output: string[]) {
  if (paragraphLines.length === 0) return
  output.push(`<p>${paragraphLines.map((line) => renderInline(line)).join('<br />')}</p>`)
  paragraphLines.length = 0
}

function flushList(listItems: string[], ordered: boolean, output: string[]) {
  if (listItems.length === 0) return
  const tag = ordered ? 'ol' : 'ul'
  output.push(
    `<${tag}>${listItems.map((item) => `<li>${renderInline(item)}</li>`).join('')}</${tag}>`,
  )
  listItems.length = 0
}

export function renderTicketCommentMarkdown(markdown: string) {
  const lines = markdown.replaceAll('\r\n', '\n').split('\n')
  const output: string[] = []
  const paragraphLines: string[] = []
  const listItems: string[] = []
  let listOrdered = false
  let inCodeBlock = false
  let codeLines: string[] = []

  for (const line of lines) {
    if (line.match(/^```/)) {
      flushParagraph(paragraphLines, output)
      flushList(listItems, listOrdered, output)
      if (inCodeBlock) {
        output.push(`<pre><code>${escapeHTML(codeLines.join('\n'))}</code></pre>`)
        codeLines = []
        inCodeBlock = false
      } else {
        inCodeBlock = true
      }
      continue
    }

    if (inCodeBlock) {
      codeLines.push(line)
      continue
    }

    const orderedMatch = line.match(/^\d+\.\s+(.*)$/)
    const unorderedMatch = line.match(/^[-*+]\s+(.*)$/)
    if (orderedMatch || unorderedMatch) {
      flushParagraph(paragraphLines, output)
      const ordered = Boolean(orderedMatch)
      if (listItems.length > 0 && listOrdered !== ordered) {
        flushList(listItems, listOrdered, output)
      }
      listOrdered = ordered
      listItems.push((orderedMatch ?? unorderedMatch)?.[1] ?? '')
      continue
    }

    if (line.trim() === '') {
      flushParagraph(paragraphLines, output)
      flushList(listItems, listOrdered, output)
      continue
    }

    flushList(listItems, listOrdered, output)
    paragraphLines.push(line)
  }

  if (inCodeBlock) {
    output.push(`<pre><code>${escapeHTML(codeLines.join('\n'))}</code></pre>`)
  }
  flushParagraph(paragraphLines, output)
  flushList(listItems, listOrdered, output)

  return output.join('')
}
