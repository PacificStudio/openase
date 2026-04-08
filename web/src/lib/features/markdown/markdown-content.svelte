<script lang="ts">
  import { cn } from '$lib/utils'
  import { Streamdown } from 'streamdown-svelte'
  import {
    MARKDOWN_ALLOWED_IMAGE_PREFIXES,
    MARKDOWN_ALLOWED_LINK_PREFIXES,
    MARKDOWN_DISALLOWED_ELEMENTS,
    markdownUrlTransform,
  } from './policy'

  let {
    source,
    class: className,
    streaming = false,
  }: {
    source: string
    class?: string
    streaming?: boolean
  } = $props()
</script>

<Streamdown
  content={source}
  mode={streaming ? 'streaming' : 'static'}
  parseIncompleteMarkdown={streaming}
  baseTheme="shadcn"
  allowedLinkPrefixes={MARKDOWN_ALLOWED_LINK_PREFIXES}
  allowedImagePrefixes={MARKDOWN_ALLOWED_IMAGE_PREFIXES}
  linkSafety={{ enabled: false }}
  disallowedElements={MARKDOWN_DISALLOWED_ELEMENTS}
  skipHtml={true}
  urlTransform={markdownUrlTransform}
  class={cn(
    'min-w-0 text-sm leading-relaxed break-words [&_[data-streamdown="code-block"]]:my-3 [&_[data-streamdown="table-wrapper"]]:my-3 [&_a]:break-words [&_p:first-child]:mt-0 [&_p:last-child]:mb-0',
    className,
  )}
/>
