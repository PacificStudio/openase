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
  import {
    ArrowDown,
    ArrowUp,
    CircleDot,
    Ellipsis,
    GripVertical,
    Pencil,
    Trash2,
    X,
  } from '@lucide/svelte'

  let {
    status,
    busy = false,
    canMoveUp = false,
    canMoveDown = false,
    onSave,
    onDelete,
    onMoveUp,
    onMoveDown,
    onSetDefault,
    onDragStart,
    onDragEnd,
  }: {
    status: EditableStatus
    busy?: boolean
    canMoveUp?: boolean
    canMoveDown?: boolean
    onSave: (statusId: string, draft: ParsedStatusDraft) => Promise<void> | void
    onDelete: (status: EditableStatus) => Promise<void> | void
    onMoveUp: (statusId: string) => Promise<void> | void
    onMoveDown: (statusId: string) => Promise<void> | void
    onSetDefault: (statusId: string) => Promise<void> | void
    onDragStart: (statusId: string) => void
    onDragEnd: () => void
  } = $props()

  let editing = $state(false)
  let draft = $state<StatusDraft>({
    name: '',
    stage: 'unstarted',
    color: '#94a3b8',
    isDefault: false,
    maxActiveRuns: '',
  })
  let validationError = $state('')

  function enterEdit() {
    draft = {
      name: status.name,
      stage: status.stage,
      color: status.color,
      isDefault: status.isDefault,
      maxActiveRuns: status.maxActiveRuns?.toString() ?? '',
    }
    validationError = ''
    editing = true
  }

  function exitEdit() {
    editing = false
    validationError = ''
  }

  async function handleSave() {
    const parsed = parseStatusDraft({ ...draft, stage: status.stage })
    if (!parsed.ok) {
      validationError = parsed.error
      return
    }
    validationError = ''
    await onSave(status.id, parsed.value)
    editing = false
  }

  function handleDragStart(e: DragEvent) {
    e.dataTransfer?.setData('text/plain', status.id)
    if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
    onDragStart(status.id)
  }
</script>

{#if editing}
  <div class="border-border bg-muted/30 rounded-md border px-3 py-3">
    <div class="flex items-center gap-3">
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
      <Button size="sm" disabled={busy} onclick={handleSave}>Save</Button>
      <Button variant="ghost" size="sm" disabled={busy} onclick={exitEdit}>
        <X class="size-3.5" />
      </Button>
    </div>
    {#if validationError}
      <p class="text-destructive mt-2 text-xs">{validationError}</p>
    {/if}
  </div>
{:else}
  <div
    class="group hover:bg-muted/50 flex items-center gap-3 rounded-md px-2 py-2 transition-colors"
    draggable="true"
    ondragstart={handleDragStart}
    ondragend={onDragEnd}
    role="listitem"
  >
    <GripVertical
      class="text-muted-foreground size-4 shrink-0 cursor-grab opacity-0 transition-opacity group-hover:opacity-100"
    />

    <span class="size-3 shrink-0 rounded-full" style="background-color: {status.color}"></span>

    <span class="text-foreground flex-1 truncate text-sm font-medium">{status.name}</span>

    {#if status.isDefault}
      <Badge variant="secondary" class="shrink-0 text-[10px]">Default</Badge>
    {/if}

    {#if status.maxActiveRuns}
      <Badge variant="outline" class="text-muted-foreground shrink-0 text-[10px]">
        {status.activeRuns}/{status.maxActiveRuns}
      </Badge>
    {/if}

    <DropdownMenu.Root>
      <DropdownMenu.Trigger>
        {#snippet child({ props })}
          <Button
            variant="ghost"
            size="sm"
            class="h-7 w-7 shrink-0 p-0 opacity-0 transition-opacity group-hover:opacity-100"
            {...props}
          >
            <Ellipsis class="size-3.5" />
            <span class="sr-only">Actions</span>
          </Button>
        {/snippet}
      </DropdownMenu.Trigger>
      <DropdownMenu.Content align="end" class="w-44">
        <DropdownMenu.Item disabled={busy} onclick={enterEdit}>
          <Pencil class="mr-2 size-3.5" />
          Edit
        </DropdownMenu.Item>
        {#if !status.isDefault}
          <DropdownMenu.Item disabled={busy} onclick={() => onSetDefault(status.id)}>
            <CircleDot class="mr-2 size-3.5" />
            Set as default
          </DropdownMenu.Item>
        {/if}
        {#if canMoveUp}
          <DropdownMenu.Item disabled={busy} onclick={() => onMoveUp(status.id)}>
            <ArrowUp class="mr-2 size-3.5" />
            Move up
          </DropdownMenu.Item>
        {/if}
        {#if canMoveDown}
          <DropdownMenu.Item disabled={busy} onclick={() => onMoveDown(status.id)}>
            <ArrowDown class="mr-2 size-3.5" />
            Move down
          </DropdownMenu.Item>
        {/if}
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
{/if}
