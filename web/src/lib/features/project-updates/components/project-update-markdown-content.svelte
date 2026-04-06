<script lang="ts">
  import { cn } from '$lib/utils'
  import { renderMarkdown } from '../markdown'

  let {
    source,
    class: className,
  }: {
    source: string
    class?: string
  } = $props()

  const rendered = $derived(renderMarkdown(source))
</script>

<div
  class={cn(
    '[&_a]:text-primary [&_blockquote]:border-border [&_blockquote]:text-muted-foreground [&_code]:bg-muted [&_pre]:bg-muted [&_th]:bg-muted/60 overflow-hidden text-sm leading-relaxed break-words [&_a]:underline [&_blockquote]:my-3 [&_blockquote]:border-l-2 [&_blockquote]:pl-3 [&_code]:rounded-sm [&_code]:px-1 [&_code]:py-0.5 [&_code]:break-all [&_h1]:text-base [&_h1]:font-semibold [&_h2]:text-base [&_h2]:font-semibold [&_h3]:font-medium [&_li]:ml-5 [&_li]:list-disc [&_ol]:my-3 [&_ol]:space-y-1 [&_p]:my-3 [&_p:first-child]:mt-0 [&_p:last-child]:mb-0 [&_pre]:max-w-full [&_pre]:overflow-x-auto [&_pre]:rounded-md [&_pre]:p-3 [&_pre]:text-xs [&_table]:my-3 [&_table]:block [&_table]:max-w-full [&_table]:border-collapse [&_table]:overflow-x-auto [&_td]:border [&_td]:p-2 [&_td]:whitespace-nowrap [&_th]:border [&_th]:p-2 [&_th]:whitespace-nowrap [&_ul]:my-3 [&_ul]:space-y-1',
    className,
  )}
>
  <!-- eslint-disable-next-line svelte/no-at-html-tags -->
  {@html rendered}
</div>
