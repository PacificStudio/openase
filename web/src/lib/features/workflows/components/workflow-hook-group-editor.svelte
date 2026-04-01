<script lang="ts" generics="TEvent extends string">
  import type {
    WorkflowHookEventOption,
    WorkflowHookRowDraft,
    WorkflowHookRowDraftErrors,
  } from '../workflow-hooks'
  import WorkflowHookEventEditor from './workflow-hook-event-editor.svelte'

  let {
    label,
    description,
    events,
    rowsByEvent,
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
    events: WorkflowHookEventOption<TEvent>[]
    rowsByEvent: Record<TEvent, WorkflowHookRowDraft[]>
    rowErrors?: Record<string, WorkflowHookRowDraftErrors>
    allowWorkdir?: boolean
    disabled?: boolean
    onAdd?: (event: TEvent) => void
    onChange?: (event: TEvent, index: number, row: WorkflowHookRowDraft) => void
    onDuplicate?: (event: TEvent, index: number) => void
    onDelete?: (event: TEvent, index: number) => void
  } = $props()
</script>

<section class="space-y-4">
  <div class="space-y-1">
    <h3 class="text-sm font-semibold">{label}</h3>
    <p class="text-muted-foreground text-xs">{description}</p>
  </div>

  <div class="space-y-3">
    {#each events as option (option.event)}
      <WorkflowHookEventEditor
        label={option.label}
        description={option.description}
        rows={rowsByEvent[option.event] ?? []}
        {rowErrors}
        {allowWorkdir}
        {disabled}
        onAdd={() => onAdd?.(option.event)}
        onChange={(index, row) => onChange?.(option.event, index, row)}
        onDuplicate={(index) => onDuplicate?.(option.event, index)}
        onDelete={(index) => onDelete?.(option.event, index)}
      />
    {/each}
  </div>
</section>
