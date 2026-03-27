<script lang="ts" module>
  export type Side = 'top' | 'right' | 'bottom' | 'left'
</script>

<script lang="ts">
  import { Dialog as SheetPrimitive } from 'bits-ui'
  import type { Snippet } from 'svelte'
  import SheetPortal from './sheet-portal.svelte'
  import SheetOverlay from './sheet-overlay.svelte'
  import { Button } from '$lib/components/ui/button/index.js'
  import XIcon from '@lucide/svelte/icons/x'
  import { cn, type WithoutChildrenOrChild } from '$lib/utils.js'
  import type { ComponentProps } from 'svelte'

  let {
    ref = $bindable(null),
    class: className,
    side = 'right',
    showCloseButton = true,
    portalProps,
    children,
    ...restProps
  }: WithoutChildrenOrChild<SheetPrimitive.ContentProps> & {
    portalProps?: WithoutChildrenOrChild<ComponentProps<typeof SheetPortal>>
    side?: Side
    showCloseButton?: boolean
    children: Snippet
  } = $props()

  function getSideLayoutClass(currentSide: typeof side) {
    switch (currentSide) {
      case 'bottom':
        return 'inset-x-0 bottom-0 h-auto border-t'
      case 'left':
        return 'inset-y-0 left-0 h-full w-3/4 border-r sm:max-w-sm'
      case 'top':
        return 'inset-x-0 top-0 h-auto border-b'
      case 'right':
      default:
        return 'inset-y-0 right-0 h-full w-3/4 border-l sm:max-w-sm'
    }
  }
</script>

<SheetPortal {...portalProps}>
  <SheetOverlay />
  <SheetPrimitive.Content
    bind:ref
    data-slot="sheet-content"
    data-side={side}
    class={cn(
      'bg-background data-open:animate-in data-open:fade-in-0 data-[side=bottom]:data-open:slide-in-from-bottom-10 data-[side=left]:data-open:slide-in-from-left-10 data-[side=right]:data-open:slide-in-from-right-10 data-[side=top]:data-open:slide-in-from-top-10 data-closed:animate-out data-closed:fade-out-0 data-[side=bottom]:data-closed:slide-out-to-bottom-10 data-[side=left]:data-closed:slide-out-to-left-10 data-[side=right]:data-closed:slide-out-to-right-10 data-[side=top]:data-closed:slide-out-to-top-10 fixed z-50 flex flex-col gap-4 bg-clip-padding text-sm shadow-lg transition duration-200 ease-in-out',
      getSideLayoutClass(side),
      className,
    )}
    {...restProps}
  >
    {@render children?.()}
    {#if showCloseButton}
      <SheetPrimitive.Close data-slot="sheet-close">
        {#snippet child({ props })}
          <Button variant="ghost" class="absolute top-4 right-4" size="icon-sm" {...props}>
            <XIcon />
            <span class="sr-only">Close</span>
          </Button>
        {/snippet}
      </SheetPrimitive.Close>
    {/if}
  </SheetPrimitive.Content>
</SheetPortal>
