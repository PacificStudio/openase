<script lang="ts">
  import { X } from '@lucide/svelte'
  import { Badge } from '$ui/badge'

  let {
    label,
    title,
    detail = '',
    actionLabel = '',
    actionDisabled = false,
    onAction,
    onDismiss,
  }: {
    label: string
    title: string
    detail?: string
    actionLabel?: string
    actionDisabled?: boolean
    onAction?: () => void
    onDismiss: () => void
  } = $props()
</script>

<div
  class="bg-muted/30 mb-1.5 flex items-center gap-1.5 rounded-md px-2 py-1 text-[11px] leading-tight"
>
  <Badge variant="secondary" class="shrink-0 px-1.5 py-0 text-[10px] font-medium">{label}</Badge>
  <span class="text-foreground min-w-0 truncate font-medium">{title}</span>
  {#if detail}
    <span class="text-muted-foreground hidden shrink-0 sm:inline">·</span>
    <span class="text-muted-foreground hidden min-w-0 truncate sm:inline">{detail}</span>
  {/if}
  {#if actionLabel}
    <button
      type="button"
      class="border-border bg-background text-foreground hover:bg-muted rounded border px-1.5 py-0.5 text-[10px] font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50"
      disabled={actionDisabled}
      onclick={onAction}
    >
      {actionLabel}
    </button>
  {/if}
  <button
    type="button"
    class="text-muted-foreground hover:text-foreground -mr-0.5 ml-auto shrink-0 rounded p-0.5 transition-colors"
    aria-label="Remove focus for this send"
    onclick={onDismiss}
  >
    <X class="size-3" />
  </button>
</div>
