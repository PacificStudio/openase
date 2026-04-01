<script lang="ts">
  import {
    ticketStatusStageOptions,
    type EditableStatus,
    type ParsedStatusDraft,
    type TicketStatusStage,
  } from '$lib/features/statuses/public'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Plus, X } from '@lucide/svelte'
  import StatusSettingsRow from './status-settings-row.svelte'

  let {
    statuses,
    loading = false,
    resetting = false,
    creating = false,
    busyStatusId = '',
    onSave,
    onDelete,
    onMoveInStage,
    onSetDefault,
    onMoveToStage,
    onCreateInStage,
  }: {
    statuses: EditableStatus[]
    loading?: boolean
    resetting?: boolean
    creating?: boolean
    busyStatusId?: string
    onSave: (statusId: string, draft: ParsedStatusDraft) => Promise<void> | void
    onDelete: (status: EditableStatus) => Promise<void> | void
    onMoveInStage: (statusId: string, direction: 'up' | 'down') => Promise<void> | void
    onSetDefault: (statusId: string) => Promise<void> | void
    onMoveToStage: (statusId: string, newStage: TicketStatusStage) => Promise<void> | void
    onCreateInStage: (
      stage: TicketStatusStage,
      name: string,
      color: string,
      maxActiveRuns: string,
    ) => Promise<boolean>
  } = $props()

  const stageAccent: Record<TicketStatusStage, string> = {
    backlog: 'border-l-slate-400',
    unstarted: 'border-l-blue-400',
    started: 'border-l-amber-400',
    completed: 'border-l-emerald-400',
    canceled: 'border-l-rose-400',
  }

  const stageDropHighlight: Record<TicketStatusStage, string> = {
    backlog: 'ring-slate-400/40 bg-slate-400/5',
    unstarted: 'ring-blue-400/40 bg-blue-400/5',
    started: 'ring-amber-400/40 bg-amber-400/5',
    completed: 'ring-emerald-400/40 bg-emerald-400/5',
    canceled: 'ring-rose-400/40 bg-rose-400/5',
  }

  type StageGroup = {
    stage: TicketStatusStage
    label: string
    statuses: EditableStatus[]
  }

  const stageGroups = $derived<StageGroup[]>(
    ticketStatusStageOptions.map((opt) => ({
      stage: opt.value,
      label: opt.label,
      statuses: statuses
        .filter((s) => s.stage === opt.value)
        .sort((a, b) => a.position - b.position),
    })),
  )

  // Drag state
  let draggedStatusId = $state('')
  let dragSourceStage = $state<TicketStatusStage | ''>('')
  let dropTargetStage = $state<TicketStatusStage | ''>('')

  function handleStatusDragStart(statusId: string) {
    draggedStatusId = statusId
    const s = statuses.find((st) => st.id === statusId)
    dragSourceStage = s?.stage ?? ''
  }

  function handleStatusDragEnd() {
    draggedStatusId = ''
    dragSourceStage = ''
    dropTargetStage = ''
  }

  function handleStageDragOver(stage: TicketStatusStage, e: DragEvent) {
    if (!draggedStatusId || stage === dragSourceStage) return
    e.preventDefault()
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
    dropTargetStage = stage
  }

  function handleStageDragLeave(stage: TicketStatusStage) {
    if (dropTargetStage === stage) dropTargetStage = ''
  }

  function handleStageDrop(stage: TicketStatusStage, e: DragEvent) {
    e.preventDefault()
    const statusId = e.dataTransfer?.getData('text/plain') || draggedStatusId
    dropTargetStage = ''
    draggedStatusId = ''
    dragSourceStage = ''
    if (statusId) onMoveToStage(statusId, stage)
  }

  // Inline add state
  let addingInStage = $state<TicketStatusStage | null>(null)
  let addName = $state('')
  let addColor = $state('#94a3b8')
  let addMaxActiveRuns = $state('')

  function openInlineAdd(stage: TicketStatusStage) {
    addingInStage = stage
    addName = ''
    addColor = '#94a3b8'
    addMaxActiveRuns = ''
  }

  function closeInlineAdd() {
    addingInStage = null
  }

  async function submitInlineAdd() {
    if (!addingInStage) return
    const ok = await onCreateInStage(addingInStage, addName, addColor, addMaxActiveRuns)
    if (ok) closeInlineAdd()
  }
</script>

{#if loading}
  <div class="text-muted-foreground text-sm">Loading statuses…</div>
{:else}
  <div class="space-y-4">
    {#each stageGroups as group (group.stage)}
      <section
        class={cn(
          'rounded-lg border-l-[3px] transition-all',
          stageAccent[group.stage],
          dropTargetStage === group.stage && dragSourceStage !== group.stage
            ? `ring-2 ${stageDropHighlight[group.stage]}`
            : '',
        )}
        ondragover={(e) => handleStageDragOver(group.stage, e)}
        ondragleave={() => handleStageDragLeave(group.stage)}
        ondrop={(e) => handleStageDrop(group.stage, e)}
        role="group"
        aria-label="{group.label} stage"
      >
        <div class="flex items-center justify-between px-4 py-2.5">
          <div class="flex items-center gap-2">
            <h3 class="text-foreground text-sm font-semibold">{group.label}</h3>
            <span class="text-muted-foreground text-xs">({group.statuses.length})</span>
          </div>
          {#if addingInStage !== group.stage}
            <Button
              variant="ghost"
              size="sm"
              class="text-muted-foreground hover:text-foreground h-7 gap-1 px-2 text-xs"
              disabled={creating || resetting}
              onclick={() => openInlineAdd(group.stage)}
            >
              <Plus class="size-3" />
              Add
            </Button>
          {/if}
        </div>

        <div class="px-2 pb-2">
          {#if group.statuses.length === 0 && addingInStage !== group.stage}
            <div
              class="text-muted-foreground rounded-md border border-dashed px-4 py-3 text-center text-xs"
            >
              No statuses
            </div>
          {:else}
            <div class="space-y-0.5" role="list">
              {#each group.statuses as status, idx (status.id)}
                <StatusSettingsRow
                  {status}
                  busy={busyStatusId === status.id || resetting || loading}
                  canMoveUp={idx > 0}
                  canMoveDown={idx < group.statuses.length - 1}
                  {onSave}
                  {onDelete}
                  onMoveUp={(id) => onMoveInStage(id, 'up')}
                  onMoveDown={(id) => onMoveInStage(id, 'down')}
                  {onSetDefault}
                  onDragStart={handleStatusDragStart}
                  onDragEnd={handleStatusDragEnd}
                />
              {/each}
            </div>
          {/if}

          {#if addingInStage === group.stage}
            <div class="bg-muted/30 mt-1 flex items-center gap-2 rounded-md px-2 py-2">
              <input
                type="color"
                bind:value={addColor}
                class="size-8 shrink-0 cursor-pointer rounded border-0 bg-transparent p-0"
              />
              <Input
                bind:value={addName}
                class="h-8 flex-1 text-sm"
                placeholder="Status name"
                autofocus
              />
              <Input
                bind:value={addMaxActiveRuns}
                type="number"
                min="1"
                step="1"
                class="h-8 w-28 text-sm"
                placeholder="Max runs"
              />
              <Button
                size="sm"
                class="h-8"
                disabled={creating || !addName.trim()}
                onclick={submitInlineAdd}
              >
                {creating ? 'Adding…' : 'Add'}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                class="h-8 w-8 p-0"
                disabled={creating}
                onclick={closeInlineAdd}
              >
                <X class="size-3.5" />
              </Button>
            </div>
          {/if}
        </div>
      </section>
    {/each}
  </div>
{/if}
