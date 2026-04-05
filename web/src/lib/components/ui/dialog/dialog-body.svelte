<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements'
  import { cn, type WithElementRef } from '$lib/utils.js'

  let {
    ref = $bindable(null),
    class: className,
    children,
    ...restProps
  }: WithElementRef<HTMLAttributes<HTMLDivElement>> = $props()
</script>

<div
  bind:this={ref}
  data-slot="dialog-body"
  class={cn('min-h-0 flex-1 overflow-y-auto', className)}
  {...restProps}
>
  {@render children?.()}
</div>
