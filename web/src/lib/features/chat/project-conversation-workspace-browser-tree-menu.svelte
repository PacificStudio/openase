<script lang="ts">
  import { cn } from '$lib/utils'

  export type TreeMenuItem =
    | { kind: 'separator' }
    | {
        kind: 'item'
        label: string
        onSelect: () => void
        danger?: boolean
      }

  let {
    x,
    y,
    items,
    onClose,
  }: {
    x: number
    y: number
    items: TreeMenuItem[]
    onClose: () => void
  } = $props()

  let menuEl = $state<HTMLDivElement | null>(null)

  $effect(() => {
    const handleMouseDown = (event: MouseEvent) => {
      if (menuEl && event.target instanceof Node && menuEl.contains(event.target)) {
        return
      }
      onClose()
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('mousedown', handleMouseDown)
    window.addEventListener('keydown', handleKeyDown)
    return () => {
      window.removeEventListener('mousedown', handleMouseDown)
      window.removeEventListener('keydown', handleKeyDown)
    }
  })

  // Rough viewport clamp: keep the menu on-screen without measuring exact
  // height up front. 240 px matches the tallest menu we currently build.
  const estimatedWidth = 220
  const estimatedHeight = 240
  const clampedX = $derived(
    Math.max(8, Math.min(x, (globalThis.innerWidth ?? 1024) - estimatedWidth - 8)),
  )
  const clampedY = $derived(
    Math.max(8, Math.min(y, (globalThis.innerHeight ?? 768) - estimatedHeight - 8)),
  )
</script>

<div
  bind:this={menuEl}
  class="border-border bg-popover text-popover-foreground fixed z-50 min-w-[13rem] rounded-md border p-1 shadow-md"
  style="left: {clampedX}px; top: {clampedY}px"
  role="menu"
  data-testid="workspace-browser-tree-menu"
>
  {#each items as item, index (index)}
    {#if item.kind === 'separator'}
      <div class="bg-border my-1 h-px"></div>
    {:else}
      <button
        type="button"
        role="menuitem"
        class={cn(
          'flex w-full items-center justify-between gap-4 rounded-sm px-2 py-1.5 text-left text-[12px]',
          item.danger
            ? 'text-destructive hover:bg-destructive/10'
            : 'hover:bg-accent hover:text-accent-foreground',
        )}
        onclick={() => {
          item.onSelect()
          onClose()
        }}
      >
        <span>{item.label}</span>
      </button>
    {/if}
  {/each}
</div>
