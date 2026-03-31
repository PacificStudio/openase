<script lang="ts">
  import { cn } from '$lib/utils'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Separator } from '$ui/separator'
  import * as Select from '$ui/select'
  import { Clock3, Layers3, RotateCcw } from '@lucide/svelte'
  import type { WorkflowAgentOption, WorkflowStatusOption, WorkflowSummary } from '../types'
  import {
    createWorkflowLifecycleDraft,
    parseWorkflowLifecycleDraft,
    toggleWorkflowStatusSelection,
    type WorkflowLifecycleDraft,
    type WorkflowLifecyclePayload,
  } from '../workflow-lifecycle'
  import WorkflowAgentBindingCard from './workflow-agent-binding-card.svelte'
  import WorkflowAgentSelectOption from './workflow-agent-select-option.svelte'
  import WorkflowAgentSelectTrigger from './workflow-agent-select-trigger.svelte'
  import WorkflowDetailActions from './workflow-detail-actions.svelte'
  import WorkflowDetailHeader from './workflow-detail-header.svelte'
  import WorkflowNumberField from './workflow-number-field.svelte'
  import WorkflowStatusChipSelector from './workflow-status-chip-selector.svelte'
  let {
    workflow,
    statuses = [],
    agentOptions = [],
    saving = false,
    deleting = false,
    onSave,
    onDelete,
    class: className = '',
  }: {
    workflow: WorkflowSummary
    statuses?: WorkflowStatusOption[]
    agentOptions?: WorkflowAgentOption[]
    saving?: boolean
    deleting?: boolean
    onSave?: (payload: WorkflowLifecyclePayload) => void | Promise<void>
    onDelete?: () => void | Promise<void>
    class?: string
  } = $props()

  let draft = $state<WorkflowLifecycleDraft>({
    agentId: '',
    name: '',
    pickupStatusIds: [],
    finishStatusIds: [],
    maxConcurrent: '',
    maxRetryAttempts: '',
    timeoutMinutes: '',
    stallTimeoutMinutes: '',
    isActive: false,
  })
  let draftKey = $state('')
  let formError = $state('')

  const baseDraft = $derived(createWorkflowLifecycleDraft(workflow))
  const isDirty = $derived(
    draft.agentId !== baseDraft.agentId ||
      draft.name !== baseDraft.name ||
      draft.pickupStatusIds.join(':') !== baseDraft.pickupStatusIds.join(':') ||
      draft.finishStatusIds.join(':') !== baseDraft.finishStatusIds.join(':') ||
      draft.maxConcurrent !== baseDraft.maxConcurrent ||
      draft.maxRetryAttempts !== baseDraft.maxRetryAttempts ||
      draft.timeoutMinutes !== baseDraft.timeoutMinutes ||
      draft.stallTimeoutMinutes !== baseDraft.stallTimeoutMinutes ||
      draft.isActive !== baseDraft.isActive,
  )
  const selectedAgent = $derived(agentOptions.find((option) => option.id === draft.agentId) ?? null)

  $effect(() => {
    const nextKey = [
      workflow.id,
      workflow.version,
      workflow.agentId ?? '',
      workflow.name,
      workflow.isActive,
      workflow.pickupStatusIds.join(','),
      workflow.finishStatusIds.join(','),
      workflow.maxConcurrent,
      workflow.maxRetry,
      workflow.timeoutMinutes,
      workflow.stallTimeoutMinutes,
    ].join(':')

    if (nextKey === draftKey) return

    draft = createWorkflowLifecycleDraft(workflow)
    draftKey = nextKey
    formError = ''
  })

  function updateDraftField<K extends keyof WorkflowLifecycleDraft>(
    field: K,
    value: WorkflowLifecycleDraft[K],
  ) {
    draft = {
      ...draft,
      [field]: value,
    }
    formError = ''
  }

  function togglePickupStatus(statusId: string) {
    updateDraftField(
      'pickupStatusIds',
      toggleWorkflowStatusSelection(draft.pickupStatusIds, statusId),
    )
  }

  function toggleFinishStatus(statusId: string) {
    updateDraftField(
      'finishStatusIds',
      toggleWorkflowStatusSelection(draft.finishStatusIds, statusId),
    )
  }

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()
    formError = ''

    const parsed = parseWorkflowLifecycleDraft(draft)
    if (!parsed.ok) {
      formError = parsed.error
      return
    }

    await onSave?.(parsed.value)
  }

  async function handleDelete() {
    formError = ''
    if (!confirm(`Delete workflow "${workflow.name}"? This cannot be undone.`)) {
      return
    }

    await onDelete?.()
  }
</script>

<div class={cn('flex h-full flex-col overflow-y-auto', className)}>
  <WorkflowDetailHeader
    {workflow}
    isActive={draft.isActive}
    disabled={saving || deleting}
    onToggle={() => updateDraftField('isActive', !draft.isActive)}
  />

  <Separator />

  <form class="flex flex-1 flex-col" onsubmit={handleSubmit}>
    <div class="flex-1 space-y-6 px-4 py-4">
      <div class="space-y-1.5">
        <Label
          for="workflow-name"
          class="text-muted-foreground text-xs font-medium tracking-wide uppercase">Name</Label
        >
        <Input
          id="workflow-name"
          value={draft.name}
          disabled={saving || deleting}
          oninput={(event) =>
            updateDraftField('name', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="space-y-1.5">
        <Label class="text-muted-foreground text-xs font-medium tracking-wide uppercase"
          >Bound Agent</Label
        >
        <Select.Root
          type="single"
          value={draft.agentId}
          disabled={saving || deleting || agentOptions.length === 0}
          onValueChange={(value) => updateDraftField('agentId', value || '')}
        >
          <Select.Trigger class="h-auto w-full py-2">
            <WorkflowAgentSelectTrigger {selectedAgent} />
          </Select.Trigger>
          <Select.Content>
            {#each agentOptions as option (option.id)}
              <Select.Item value={option.id}>
                <WorkflowAgentSelectOption {option} />
              </Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <WorkflowAgentBindingCard {selectedAgent} />

      <Separator />

      <div class="space-y-1.5">
        <span class="text-muted-foreground text-xs font-medium tracking-wide uppercase">Limits</span
        >
        <div class="grid gap-3 sm:grid-cols-2">
          <WorkflowNumberField
            id="workflow-max-concurrent"
            label="Max Concurrent"
            value={draft.maxConcurrent}
            icon={Layers3}
            disabled={saving || deleting}
            oninput={(value) => updateDraftField('maxConcurrent', value)}
          />
          <WorkflowNumberField
            id="workflow-max-retry"
            label="Max Retry"
            value={draft.maxRetryAttempts}
            icon={RotateCcw}
            disabled={saving || deleting}
            oninput={(value) => updateDraftField('maxRetryAttempts', value)}
          />
          <WorkflowNumberField
            id="workflow-timeout"
            label="Timeout (min)"
            value={draft.timeoutMinutes}
            icon={Clock3}
            disabled={saving || deleting}
            oninput={(value) => updateDraftField('timeoutMinutes', value)}
          />
          <WorkflowNumberField
            id="workflow-stall-timeout"
            label="Stall Timeout (min)"
            value={draft.stallTimeoutMinutes}
            icon={Clock3}
            disabled={saving || deleting}
            oninput={(value) => updateDraftField('stallTimeoutMinutes', value)}
          />
        </div>
      </div>

      <Separator />

      <div class="space-y-4">
        <WorkflowStatusChipSelector
          label="Pickup Statuses"
          {statuses}
          selectedStatusIds={draft.pickupStatusIds}
          disabled={saving || deleting}
          onToggle={togglePickupStatus}
        />
        <WorkflowStatusChipSelector
          label="Finish Statuses"
          {statuses}
          selectedStatusIds={draft.finishStatusIds}
          disabled={saving || deleting}
          onToggle={toggleFinishStatus}
        />
      </div>
    </div>

    <WorkflowDetailActions
      errorMessage={formError}
      {saving}
      {deleting}
      {isDirty}
      onDelete={handleDelete}
    />
  </form>
</div>
