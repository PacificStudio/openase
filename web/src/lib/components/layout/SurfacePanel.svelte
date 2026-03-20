<script lang="ts">
  import type { Snippet } from 'svelte'
  import { Card } from '$lib/components/ui/card'
  import { cn } from '$lib/utils'

  let {
    children,
    header,
    actions,
    class: className = '',
    bodyClass = '',
  }: {
    children?: Snippet
    header?: Snippet
    actions?: Snippet
    class?: string
    bodyClass?: string
  } = $props()
</script>

<Card
  class={cn(
    'border-border/80 bg-background/88 flex min-h-0 flex-col overflow-hidden border shadow-sm backdrop-blur',
    className,
  )}
>
  {#if header || actions}
    <div class="border-border/70 flex items-start justify-between gap-3 border-b px-4 py-3">
      <div class="min-w-0 flex-1">
        {@render header?.()}
      </div>

      {#if actions}
        <div class="shrink-0">
          {@render actions()}
        </div>
      {/if}
    </div>
  {/if}

  <div class={cn('min-h-0 flex-1', bodyClass)}>
    {@render children?.()}
  </div>
</Card>
