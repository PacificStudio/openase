<script lang="ts">
  import {
    parseStatusDraft,
    type EditableStatus,
    type StatusDraft,
  } from '$lib/features/statuses/public'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import ArrowDown from '@lucide/svelte/icons/arrow-down'
  import ArrowUp from '@lucide/svelte/icons/arrow-up'
  import Save from '@lucide/svelte/icons/save'
  import Trash2 from '@lucide/svelte/icons/trash-2'

  let {
    status,
    order,
    busy = false,
    canMoveUp = false,
    canMoveDown = false,
    onSave,
    onDelete,
    onMoveUp,
    onMoveDown,
  }: {
    status: EditableStatus
    order: number
    busy?: boolean
    canMoveUp?: boolean
    canMoveDown?: boolean
    onSave: (statusId: string, draft: StatusDraft) => Promise<void> | void
    onDelete: (status: EditableStatus) => Promise<void> | void
    onMoveUp: (statusId: string) => Promise<void> | void
    onMoveDown: (statusId: string) => Promise<void> | void
  } = $props()

  let draft = $state<StatusDraft>({
    name: '',
    color: '#94a3b8',
    isDefault: false,
  })
  let validationError = $state('')

  const dirty = $derived(
    draft.name.trim() !== status.name ||
      draft.color.toLowerCase() !== status.color.toLowerCase() ||
      draft.isDefault !== status.isDefault,
  )

  $effect(() => {
    draft = {
      name: status.name,
      color: status.color,
      isDefault: status.isDefault,
    }
    validationError = ''
  })

  async function handleSave() {
    const parsed = parseStatusDraft(draft)
    if (!parsed.ok) {
      validationError = parsed.error
      return
    }

    validationError = ''
    await onSave(status.id, parsed.value)
  }
</script>

<div class="border-border rounded-md border px-3 py-3">
  <div class="flex flex-col gap-3 lg:flex-row lg:items-center">
    <div class="flex min-w-0 flex-1 items-center gap-3">
      <span class="text-muted-foreground w-7 shrink-0 text-xs font-medium">
        {order + 1}.
      </span>
      <input
        type="color"
        bind:value={draft.color}
        disabled={busy}
        class="size-8 shrink-0 cursor-pointer rounded border-0 bg-transparent p-0 disabled:cursor-not-allowed"
      />
      <Input
        bind:value={draft.name}
        disabled={busy}
        class="h-9 flex-1 text-sm"
        placeholder="Status name"
      />
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <label class="text-muted-foreground flex items-center gap-2 text-xs font-medium">
        <input type="checkbox" bind:checked={draft.isDefault} disabled={busy} />
        Default
      </label>

      <Button
        variant="outline"
        size="sm"
        disabled={busy || !canMoveUp}
        onclick={() => onMoveUp(status.id)}
      >
        <ArrowUp class="size-3.5" />
        Up
      </Button>
      <Button
        variant="outline"
        size="sm"
        disabled={busy || !canMoveDown}
        onclick={() => onMoveDown(status.id)}
      >
        <ArrowDown class="size-3.5" />
        Down
      </Button>
      <Button variant="outline" size="sm" disabled={busy || !dirty} onclick={handleSave}>
        <Save class="size-3.5" />
        Save
      </Button>
      <Button variant="destructive" size="sm" disabled={busy} onclick={() => onDelete(status)}>
        <Trash2 class="size-3.5" />
        Delete
      </Button>
    </div>
  </div>

  {#if validationError}
    <p class="text-destructive mt-2 text-xs">{validationError}</p>
  {/if}
</div>
