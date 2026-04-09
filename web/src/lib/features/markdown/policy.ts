import type { UrlTransform } from 'streamdown-svelte'

export const MARKDOWN_ALLOWED_LINK_PREFIXES = ['http://', 'https://', 'mailto:', '/', '#']
export const MARKDOWN_ALLOWED_IMAGE_PREFIXES: string[] = []
export const MARKDOWN_DISALLOWED_ELEMENTS = ['img']

// Images stay disabled until the product defines an explicit safe media policy.
export const markdownUrlTransform: UrlTransform = (url, key, node) => {
  if (node.tagName === 'img' && key === 'src') {
    return null
  }

  return url
}
