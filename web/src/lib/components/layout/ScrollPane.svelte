<script lang="ts">
  import type { Snippet } from 'svelte'
  import { cn } from '$lib/utils'

  let {
    children,
    class: className = '',
  }: {
    children?: Snippet
    class?: string
  } = $props()
</script>

<div class={cn('min-h-0 overflow-y-auto overscroll-contain pr-1', className)}>
  {@render children?.()}
</div>
