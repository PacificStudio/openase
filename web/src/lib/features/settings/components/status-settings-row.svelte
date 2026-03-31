<script lang="ts">
  import {
    parseStatusDraft,
    type EditableStatus,
    type ParsedStatusDraft,
    type StatusDraft,
  } from '$lib/features/statuses/public'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { Input } from '$ui/input'
  import { ArrowDown, ArrowUp, CircleDot, Ellipsis, Save, Trash2 } from '@lucide/svelte'

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
    onSetDefault,
  }: {
    status: EditableStatus
    order: number
    busy?: boolean
    canMoveUp?: boolean
    canMoveDown?: boolean
    onSave: (statusId: string, draft: ParsedStatusDraft) => Promise<void> | void
    onDelete: (status: EditableStatus) => Promise<void> | void
    onMoveUp: (statusId: string) => Promise<void> | void
    onMoveDown: (statusId: string) => Promise<void> | void
    onSetDefault: (statusId: string) => Promise<void> | void
  } = $props()

  let draft = $state<StatusDraft>({
    name: '',
    color: '#94a3b8',
    isDefault: false,
    maxActiveRuns: '',
  })
  let validationError = $state('')

  const dirty = $derived(
    draft.name.trim() !== status.name ||
      draft.color.toLowerCase() !== status.color.toLowerCase() ||
      draft.maxActiveRuns !== (status.maxActiveRuns?.toString() ?? ''),
  )

  $effect(() => {
    draft = {
      name: status.name,
      color: status.color,
      isDefault: status.isDefault,
      maxActiveRuns: status.maxActiveRuns?.toString() ?? '',
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
  <div class="flex items-center gap-3">
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
    <Input
      bind:value={draft.maxActiveRuns}
      type="number"
      min="1"
      step="1"
      disabled={busy}
      class="h-9 w-32 text-sm"
      placeholder="Unlimited"
    />

    {#if status.isDefault}
      <Badge variant="secondary" class="shrink-0 text-[10px]">Default</Badge>
    {/if}

    {#if status.maxActiveRuns}
      <Badge variant="outline" class="shrink-0 text-[10px]">
        {status.activeRuns} / {status.maxActiveRuns} active
      </Badge>
    {/if}

    <Button
      variant="ghost"
      size="sm"
      class="h-8 w-8 shrink-0 p-0"
      disabled={busy || !dirty}
      onclick={handleSave}
    >
      <Save class="size-3.5" />
      <span class="sr-only">Save</span>
    </Button>

    <DropdownMenu.Root>
      <DropdownMenu.Trigger>
        {#snippet child({ props })}
          <Button variant="ghost" size="sm" class="h-8 w-8 shrink-0 p-0" {...props}>
            <Ellipsis class="size-3.5" />
            <span class="sr-only">More actions</span>
          </Button>
        {/snippet}
      </DropdownMenu.Trigger>
      <DropdownMenu.Content align="end" class="w-44">
        {#if !status.isDefault}
          <DropdownMenu.Item disabled={busy} onclick={() => onSetDefault(status.id)}>
            <CircleDot class="mr-2 size-3.5" />
            Set as default
          </DropdownMenu.Item>
        {/if}
        <DropdownMenu.Item disabled={busy || !canMoveUp} onclick={() => onMoveUp(status.id)}>
          <ArrowUp class="mr-2 size-3.5" />
          Move up
        </DropdownMenu.Item>
        <DropdownMenu.Item disabled={busy || !canMoveDown} onclick={() => onMoveDown(status.id)}>
          <ArrowDown class="mr-2 size-3.5" />
          Move down
        </DropdownMenu.Item>
        <DropdownMenu.Separator />
        <DropdownMenu.Item
          class="text-destructive focus:text-destructive"
          disabled={busy}
          onclick={() => onDelete(status)}
        >
          <Trash2 class="mr-2 size-3.5" />
          Delete
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu.Root>
  </div>

  {#if validationError}
    <p class="text-destructive mt-2 text-xs">{validationError}</p>
  {/if}
</div>
