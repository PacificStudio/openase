<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    title,
    description = '',
    actions,
  }: {
    title: string
    description?: string
    actions?: Snippet
  } = $props()
</script>

<div class="flex items-center justify-between border-b border-border px-6 py-4">
  <div class="min-w-0">
    <h1 class="truncate text-lg font-semibold text-foreground">{title}</h1>
    {#if description}
      <p class="mt-0.5 text-sm text-muted-foreground">{description}</p>
    {/if}
  </div>
  {#if actions}
    <div class="flex shrink-0 items-center gap-2">
      {@render actions()}
    </div>
  {/if}
</div>
