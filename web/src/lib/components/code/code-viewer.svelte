<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { cn } from '$lib/utils'
  import { detectLanguage } from './lang'

  let {
    code = '',
    filePath = '',
    language = '',
    class: className = '',
  }: {
    /** Source code to display */
    code?: string
    /** Used to auto-detect language when `language` is not set */
    filePath?: string
    /** Explicit language override (shiki language id) */
    language?: string
    class?: string
  } = $props()

  const lang = $derived(language || detectLanguage(filePath))
  const shikiTheme = $derived(
    appStore.theme === 'light' ? 'github-light-default' : 'github-dark-default',
  )

  // Lazy-load shiki and cache the highlighter instance
  let html = $state('')
  let lastKey = ''

  $effect(() => {
    const key = `${lang}::${shikiTheme}::${code}`
    if (key === lastKey) return
    lastKey = key

    const currentCode = code
    const currentLang = lang
    const currentTheme = shikiTheme

    if (!currentCode) {
      html = ''
      return
    }

    void highlight(currentCode, currentLang, currentTheme, key)
  })

  async function highlight(source: string, langId: string, theme: string, key: string) {
    try {
      const { codeToHtml } = await import('shiki')
      const result = await codeToHtml(source, {
        lang: langId,
        theme,
      })
      // Only apply if this is still the current request
      if (lastKey === key) {
        html = result
      }
    } catch {
      // Language not supported by shiki — fall back to plain text
      if (lastKey === key) {
        html = ''
      }
    }
  }
</script>

<div
  class={cn('code-viewer min-h-0 min-w-0 overflow-auto font-mono text-[13px] leading-6', className)}
  data-testid="code-viewer-scrollport"
>
  {#if html}
    <div class="w-max min-w-full" data-testid="code-viewer-content">
      <!-- eslint-disable-next-line svelte/no-at-html-tags -->
      {@html html}
    </div>
  {:else}
    <pre class="w-max min-w-full p-4 whitespace-pre" data-testid="code-viewer-content">{code}</pre>
  {/if}
</div>

<style>
  .code-viewer :global(pre) {
    margin: 0;
    padding: 1rem;
    min-width: 100%;
    width: max-content;
    overflow: visible;
    background: transparent !important;
  }
  .code-viewer :global(code) {
    display: block;
    min-width: 100%;
    font-family: inherit;
    font-size: inherit;
    line-height: inherit;
    white-space: normal;
    counter-reset: line;
  }
  .code-viewer :global(code .line) {
    display: block;
    min-width: max-content;
    min-height: 1.5rem;
    white-space: pre;
  }
  .code-viewer :global(code .line::before) {
    counter-increment: line;
    content: counter(line);
    display: inline-block;
    width: 2.5rem;
    margin-right: 1rem;
    text-align: right;
    color: color-mix(in srgb, currentColor 25%, transparent);
    user-select: none;
    font-size: 0.75rem;
  }
</style>
