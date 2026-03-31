<script lang="ts">
  import type { EditableStage, ParsedStageDraft } from '$lib/features/stages/public'
  import {
    createEmptyStageDraft,
    parseStageDraft,
    stageKeyFromName,
    type StageDraft,
  } from '$lib/features/stages/public'
  import type { EditableStatus } from '$lib/features/statuses/public'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import Plus from '@lucide/svelte/icons/plus'
  import StageSettingsRow from './stage-settings-row.svelte'

  let {
    stages,
    statuses,
    loading = false,
    creating = false,
    busyStageId = '',
    onCreate,
    onSave,
    onDelete,
    onMove,
  }: {
    stages: EditableStage[]
    statuses: EditableStatus[]
    loading?: boolean
    creating?: boolean
    busyStageId?: string
    onCreate: (draft: ParsedStageDraft) => Promise<boolean> | boolean
    onSave: (stageId: string, draft: ParsedStageDraft) => Promise<void> | void
    onDelete: (stage: EditableStage) => Promise<void> | void
    onMove: (stageId: string, direction: 'up' | 'down') => Promise<void> | void
  } = $props()

  let createDraft = $state<StageDraft>(createEmptyStageDraft())
  let createKeyDirty = $state(false)
  let createValidationError = $state('')

  function selectedStatusCount(stageId: string) {
    return statuses.filter((status) => status.stageId === stageId).length
  }

  function updateCreateName(value: string) {
    createDraft = {
      ...createDraft,
      name: value,
      key: createKeyDirty ? createDraft.key : stageKeyFromName(value),
    }
  }

  function updateCreateKey(value: string) {
    createKeyDirty = value.trim().length > 0
    createDraft = { ...createDraft, key: value }
  }

  async function handleCreate() {
    const parsed = parseStageDraft(createDraft)
    if (!parsed.ok) {
      createValidationError = parsed.error
      return
    }

    createValidationError = ''
    const created = await onCreate(parsed.value)
    if (!created) return
    createDraft = createEmptyStageDraft()
    createKeyDirty = false
  }
</script>

<Card.Root class="gap-4">
  <Card.Header class="gap-1">
    <Card.Title>Stages</Card.Title>
    <Card.Description>
      Workflows only pick up tickets when the matched pickup status still has spare capacity in its
      stage semaphore. Configure the shared capacity here and keep statuses grouped under the right
      stage.
    </Card.Description>
  </Card.Header>

  <Card.Content class="space-y-4">
    <div class="bg-muted/40 border-border/70 rounded-xl border p-3">
      <div class="grid gap-3 md:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_11rem_auto]">
        <Input
          value={createDraft.name}
          class="h-9 text-sm"
          placeholder="New stage name"
          disabled={creating || loading}
          oninput={(event) => updateCreateName((event.currentTarget as HTMLInputElement).value)}
        />
        <Input
          value={createDraft.key}
          class="h-9 font-mono text-sm"
          placeholder="stage-key"
          disabled={creating || loading}
          oninput={(event) => updateCreateKey((event.currentTarget as HTMLInputElement).value)}
        />
        <Input
          value={createDraft.maxActiveRuns}
          type="number"
          min="1"
          step="1"
          class="h-9 text-sm"
          placeholder="Unlimited"
          disabled={creating || loading}
          oninput={(event) =>
            (createDraft = {
              ...createDraft,
              maxActiveRuns: (event.currentTarget as HTMLInputElement).value,
            })}
        />
        <Button class="shrink-0" onclick={handleCreate} disabled={creating || loading}>
          <Plus class="size-3.5" />
          {creating ? 'Adding…' : 'Add stage'}
        </Button>
      </div>
      <p class="text-muted-foreground mt-2 text-xs">
        Leave capacity blank for unlimited pickup. Stage keys stay stable after creation so workflow
        semantics remain predictable.
      </p>
      {#if createValidationError}
        <p class="text-destructive mt-2 text-xs">{createValidationError}</p>
      {/if}
    </div>

    {#if loading}
      <div class="text-muted-foreground text-sm">Loading stages…</div>
    {:else if stages.length === 0}
      <div class="text-muted-foreground rounded-md border border-dashed px-4 py-6 text-sm">
        No stages yet. Add one above to start grouping statuses behind shared concurrency limits.
      </div>
    {:else}
      <div class="space-y-2">
        {#each stages as stage, index (stage.id)}
          <StageSettingsRow
            {stage}
            order={index}
            statusCount={selectedStatusCount(stage.id)}
            busy={busyStageId === stage.id || loading}
            canMoveUp={index > 0}
            canMoveDown={index < stages.length - 1}
            {onSave}
            {onDelete}
            onMoveUp={(stageId) => onMove(stageId, 'up')}
            onMoveDown={(stageId) => onMove(stageId, 'down')}
          />
        {/each}
      </div>
    {/if}
  </Card.Content>
</Card.Root>
