<script lang="ts">
  import type { EditableStage, ParsedStageDraft } from '$lib/features/stages/public'
  import { parseStageDraft, type StageDraft } from '$lib/features/stages/public'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { ArrowDown, ArrowUp, Save, Trash2 } from '@lucide/svelte'

  let {
    stage,
    order,
    statusCount,
    busy = false,
    canMoveUp = false,
    canMoveDown = false,
    onSave,
    onDelete,
    onMoveUp,
    onMoveDown,
  }: {
    stage: EditableStage
    order: number
    statusCount: number
    busy?: boolean
    canMoveUp?: boolean
    canMoveDown?: boolean
    onSave: (stageId: string, draft: ParsedStageDraft) => Promise<void> | void
    onDelete: (stage: EditableStage) => Promise<void> | void
    onMoveUp: (stageId: string) => Promise<void> | void
    onMoveDown: (stageId: string) => Promise<void> | void
  } = $props()

  let draft = $state<StageDraft>({
    key: '',
    name: '',
    maxActiveRuns: '',
  })
  let validationError = $state('')

  const dirty = $derived(
    draft.name.trim() !== stage.name ||
      (draft.maxActiveRuns.trim() || null) !==
        (stage.maxActiveRuns === null ? null : String(stage.maxActiveRuns)),
  )

  $effect(() => {
    draft = {
      key: stage.key,
      name: stage.name,
      maxActiveRuns: stage.maxActiveRuns === null ? '' : String(stage.maxActiveRuns),
    }
    validationError = ''
  })

  function capacityLabel() {
    return stage.maxActiveRuns === null
      ? `${stage.activeRuns}`
      : `${stage.activeRuns}/${stage.maxActiveRuns}`
  }

  async function handleSave() {
    const parsed = parseStageDraft(draft)
    if (!parsed.ok) {
      validationError = parsed.error
      return
    }

    validationError = ''
    await onSave(stage.id, parsed.value)
  }
</script>

<div class="border-border rounded-md border px-3 py-3">
  <div class="flex flex-col gap-3 lg:flex-row lg:items-center">
    <div class="flex min-w-0 flex-1 items-center gap-3">
      <span class="text-muted-foreground w-7 shrink-0 text-xs font-medium">{order + 1}.</span>
      <div class="min-w-0 flex-1 space-y-3">
        <div class="flex flex-col gap-3 md:flex-row md:items-center">
          <Input
            bind:value={draft.name}
            disabled={busy}
            class="h-9 flex-1 text-sm"
            placeholder="Stage name"
          />
          <Input
            value={draft.maxActiveRuns}
            type="number"
            min="1"
            step="1"
            disabled={busy}
            class="h-9 w-full text-sm md:w-44"
            placeholder="Unlimited"
            oninput={(event) =>
              (draft = {
                ...draft,
                maxActiveRuns: (event.currentTarget as HTMLInputElement).value,
              })}
          />
        </div>
        <div class="flex flex-wrap items-center gap-2 text-xs">
          <Badge variant="outline" class="font-mono text-[10px]">{stage.key}</Badge>
          <span class="text-muted-foreground">
            {stage.activeRuns} active now, {stage.maxActiveRuns === null
              ? 'unlimited capacity'
              : `capacity ${stage.maxActiveRuns}`}
          </span>
          <Badge variant="secondary" class="text-[10px]">
            {capacityLabel()}
          </Badge>
          <span class="text-muted-foreground">
            {statusCount}
            {statusCount === 1 ? 'status' : 'statuses'}
          </span>
        </div>
      </div>
    </div>

    <div class="flex shrink-0 items-center justify-end gap-1">
      <Button
        variant="ghost"
        size="sm"
        class="h-8 w-8 p-0"
        disabled={busy || !canMoveUp}
        onclick={() => onMoveUp(stage.id)}
      >
        <ArrowUp class="size-3.5" />
        <span class="sr-only">Move up</span>
      </Button>
      <Button
        variant="ghost"
        size="sm"
        class="h-8 w-8 p-0"
        disabled={busy || !canMoveDown}
        onclick={() => onMoveDown(stage.id)}
      >
        <ArrowDown class="size-3.5" />
        <span class="sr-only">Move down</span>
      </Button>
      <Button
        variant="ghost"
        size="sm"
        class="h-8 w-8 p-0"
        disabled={busy || !dirty}
        onclick={handleSave}
      >
        <Save class="size-3.5" />
        <span class="sr-only">Save</span>
      </Button>
      <Button
        variant="ghost"
        size="sm"
        class="text-destructive hover:text-destructive h-8 w-8 p-0"
        disabled={busy}
        onclick={() => onDelete(stage)}
      >
        <Trash2 class="size-3.5" />
        <span class="sr-only">Delete</span>
      </Button>
    </div>
  </div>

  {#if validationError}
    <p class="text-destructive mt-2 text-xs">{validationError}</p>
  {/if}
</div>
