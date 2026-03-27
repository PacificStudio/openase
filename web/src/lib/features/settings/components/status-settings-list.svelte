<script lang="ts">
  import type { EditableStatus, StatusDraft } from '$lib/features/statuses/public'
  import StatusSettingsRow from './status-settings-row.svelte'

  let {
    statuses,
    loading = false,
    resetting = false,
    busyStatusId = '',
    onSave,
    onDelete,
    onMove,
    onSetDefault,
  }: {
    statuses: EditableStatus[]
    loading?: boolean
    resetting?: boolean
    busyStatusId?: string
    onSave: (statusId: string, draft: StatusDraft) => Promise<void> | void
    onDelete: (status: EditableStatus) => Promise<void> | void
    onMove: (statusId: string, direction: 'up' | 'down') => Promise<void> | void
    onSetDefault: (statusId: string) => Promise<void> | void
  } = $props()
</script>

{#if loading}
  <div class="text-muted-foreground text-sm">Loading statuses…</div>
{:else if statuses.length === 0}
  <div class="text-muted-foreground rounded-md border border-dashed px-4 py-6 text-sm">
    No statuses yet. Add one above or use reset to seed the default workflow template.
  </div>
{:else}
  <div class="space-y-2">
    {#each statuses as status, index (status.id)}
      <StatusSettingsRow
        {status}
        order={index}
        busy={busyStatusId === status.id || resetting || loading}
        canMoveUp={index > 0}
        canMoveDown={index < statuses.length - 1}
        {onSave}
        {onDelete}
        onMoveUp={(statusId) => onMove(statusId, 'up')}
        onMoveDown={(statusId) => onMove(statusId, 'down')}
        {onSetDefault}
      />
    {/each}
  </div>
{/if}
