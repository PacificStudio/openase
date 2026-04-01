import { marked } from 'marked'

marked.setOptions({
  breaks: true,
  gfm: true,
})

const allowedTags = new Set([
  'a',
  'blockquote',
  'br',
  'code',
  'em',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'hr',
  'li',
  'ol',
  'p',
  'pre',
  'strong',
  'table',
  'tbody',
  'td',
  'th',
  'thead',
  'tr',
  'ul',
])

export function renderMarkdown(source: string) {
  const trimmed = source.trim()
  if (!trimmed) return ''
  if (typeof window === 'undefined' || typeof DOMParser === 'undefined') {
    return `<p>${escapeHtml(trimmed)}</p>`
  }

  const rawHtml = marked.parse(trimmed, { async: false }) as string
  return sanitizeHtml(rawHtml)
}

function sanitizeHtml(rawHtml: string) {
  const parser = new DOMParser()
  const sourceDocument = parser.parseFromString(rawHtml, 'text/html')
  const cleanDocument = document.implementation.createHTMLDocument('')
  const container = cleanDocument.createElement('div')

  for (const node of Array.from(sourceDocument.body.childNodes)) {
    const sanitized = sanitizeNode(cleanDocument, node)
    if (sanitized) {
      container.appendChild(sanitized)
    }
  }

  return container.innerHTML
}

function sanitizeNode(targetDocument: Document, node: Node): Node | DocumentFragment | null {
  if (node.nodeType === Node.TEXT_NODE) {
    return targetDocument.createTextNode(node.textContent ?? '')
  }
  if (node.nodeType !== Node.ELEMENT_NODE) {
    return null
  }

  const element = node as HTMLElement
  const tagName = element.tagName.toLowerCase()
  if (!allowedTags.has(tagName)) {
    const fragment = targetDocument.createDocumentFragment()
    for (const child of Array.from(element.childNodes)) {
      const sanitizedChild = sanitizeNode(targetDocument, child)
      if (sanitizedChild) {
        fragment.appendChild(sanitizedChild)
      }
    }
    return fragment
  }

  const cleanElement = targetDocument.createElement(tagName)
  if (tagName === 'a') {
    const href = sanitizeHref(element.getAttribute('href'))
    if (href) {
      cleanElement.setAttribute('href', href)
      cleanElement.setAttribute('target', '_blank')
      cleanElement.setAttribute('rel', 'noreferrer noopener')
    }
  }

  for (const child of Array.from(element.childNodes)) {
    const sanitizedChild = sanitizeNode(targetDocument, child)
    if (sanitizedChild) {
      cleanElement.appendChild(sanitizedChild)
    }
  }

  return cleanElement
}

function sanitizeHref(rawHref: string | null) {
  if (!rawHref) return ''

  const href = rawHref.trim()
  if (!href) return ''
  if (href.startsWith('#') || href.startsWith('/')) return href

  try {
    const parsed = new URL(href, window.location.origin)
    if (
      parsed.protocol === 'http:' ||
      parsed.protocol === 'https:' ||
      parsed.protocol === 'mailto:'
    ) {
      return href
    }
  } catch {
    return ''
  }

  return ''
}

function escapeHtml(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}
