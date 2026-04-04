<script lang="ts">
  import { ScrollArea as ScrollAreaPrimitive } from 'bits-ui'
  import { cn, type WithoutChild } from '$lib/utils.js'

  let {
    ref = $bindable(null),
    class: className,
    orientation = 'vertical',
    children,
    ...restProps
  }: WithoutChild<ScrollAreaPrimitive.ScrollbarProps> = $props()
</script>

<ScrollAreaPrimitive.Scrollbar
  bind:ref
  data-slot="scroll-area-scrollbar"
  data-orientation={orientation}
  {orientation}
  class={cn(
    'group/scrollbar flex touch-none p-[1px] transition-colors select-none data-horizontal:h-2 data-horizontal:flex-col data-horizontal:border-t data-horizontal:border-t-transparent data-vertical:h-full data-vertical:w-2 data-vertical:border-l data-vertical:border-l-transparent',
    className,
  )}
  {...restProps}
>
  {@render children?.()}
  <ScrollAreaPrimitive.Thumb
    data-slot="scroll-area-thumb"
    class="bg-foreground/14 group-hover/scrollbar:bg-foreground/22 relative flex-1 rounded-full transition-colors duration-150"
  />
</ScrollAreaPrimitive.Scrollbar>
