<script lang="ts">
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import { Plus } from '@lucide/svelte'
  import type { WorkflowHookRowDraft, WorkflowHookRowDraftErrors } from '../workflow-hooks'
  import WorkflowHookRowEditor from './workflow-hook-row-editor.svelte'

  let {
    label,
    description,
    rows = [],
    rowErrors = {},
    allowWorkdir = false,
    disabled = false,
    onAdd,
    onChange,
    onDuplicate,
    onDelete,
  }: {
    label: string
    description: string
    rows?: WorkflowHookRowDraft[]
    rowErrors?: Record<string, WorkflowHookRowDraftErrors>
    allowWorkdir?: boolean
    disabled?: boolean
    onAdd?: () => void
    onChange?: (index: number, row: WorkflowHookRowDraft) => void
    onDuplicate?: (index: number) => void
    onDelete?: (index: number) => void
  } = $props()
</script>

<div class="space-y-3 rounded-xl border p-3">
  <div class="flex items-start justify-between gap-3">
    <div class="space-y-1">
      <div class="text-sm font-medium">{label}</div>
      <p class="text-muted-foreground text-xs">{description}</p>
    </div>
    <Button type="button" variant="outline" size="sm" {disabled} onclick={() => onAdd?.()}>
      <Plus class="size-3.5" />
      Add row
    </Button>
  </div>

  {#if rows.length === 0}
    <div class="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-sm">
      No hooks configured for {label.toLowerCase()}.
    </div>
  {:else}
    <div class="space-y-3">
      {#each rows as row, index (row.id)}
        {#if index > 0}
          <Separator />
        {/if}
        <WorkflowHookRowEditor
          {row}
          title={`${label} row ${index + 1}`}
          errors={rowErrors[row.id] ?? {}}
          {allowWorkdir}
          {disabled}
          onChange={(nextRow) => onChange?.(index, nextRow)}
          onDuplicate={() => onDuplicate?.(index)}
          onDelete={() => onDelete?.(index)}
        />
      {/each}
    </div>
  {/if}
</div>
