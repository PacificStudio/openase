<script lang="ts">
  import { MarkdownContent } from '$lib/features/markdown'

  let {
    source,
    class: className,
  }: {
    source: string
    class?: string
  } = $props()
</script>

<MarkdownContent {source} class={className} />
