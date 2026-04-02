<script lang="ts">
  import { X } from '@lucide/svelte'
  import { Button } from '$ui/button'

  let {
    label,
    title,
    detail = '',
    onDismiss,
  }: {
    label: string
    title: string
    detail?: string
    onDismiss: () => void
  } = $props()
</script>

<div class="bg-muted/40 mb-2 flex items-start justify-between gap-3 rounded-lg border px-3 py-2">
  <div class="min-w-0">
    <div class="text-muted-foreground text-[11px] font-medium tracking-[0.16em] uppercase">
      Current focus
    </div>
    <div class="truncate text-sm font-medium">{label}: {title}</div>
    {#if detail}
      <div class="text-muted-foreground truncate text-xs">{detail}</div>
    {/if}
  </div>
  <Button
    variant="ghost"
    size="sm"
    class="text-muted-foreground hover:text-foreground size-7 shrink-0 p-0"
    aria-label="Remove focus for this send"
    onclick={onDismiss}
  >
    <X class="size-4" />
  </Button>
</div>
