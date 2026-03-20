<script lang="ts">
  import type { Snippet } from 'svelte'
  import { cn } from '$lib/utils'

  let {
    main,
    detail,
    class: className = '',
    mainClass = '',
    detailClass = '',
    detailWidthClass = 'xl:grid-cols-[minmax(0,1fr)_24rem]',
  }: {
    main?: Snippet
    detail?: Snippet
    class?: string
    mainClass?: string
    detailClass?: string
    detailWidthClass?: string
  } = $props()
</script>

<div class={cn('grid min-h-0 gap-4', detail ? detailWidthClass : '', className)}>
  <div class={cn('min-h-0 space-y-4', mainClass)}>
    {@render main?.()}
  </div>

  {#if detail}
    <aside class={cn('min-h-0 space-y-4', detailClass)}>
      {@render detail()}
    </aside>
  {/if}
</div>
