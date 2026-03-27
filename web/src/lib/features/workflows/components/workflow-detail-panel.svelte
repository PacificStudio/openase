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
    type WorkflowLifecycleDraft,
    type WorkflowLifecyclePayload,
  } from '../workflow-lifecycle'
  import WorkflowDetailActions from './workflow-detail-actions.svelte'
  import WorkflowDetailHeader from './workflow-detail-header.svelte'
  import WorkflowBindingSummary from './workflow-binding-summary.svelte'
  import WorkflowNumberField from './workflow-number-field.svelte'

  const unchangedFinishStatusValue = '__unchanged__'

  let {
    workflow,
    statuses = [],
    agentOptions = [],
    saving = false,
    deleting = false,
    statusMessage = '',
    error = '',
    onSave,
    onDelete,
    class: className = '',
  }: {
    workflow: WorkflowSummary
    statuses?: WorkflowStatusOption[]
    agentOptions?: WorkflowAgentOption[]
    saving?: boolean
    deleting?: boolean
    statusMessage?: string
    error?: string
    onSave?: (payload: WorkflowLifecyclePayload) => void | Promise<void>
    onDelete?: () => void | Promise<void>
    class?: string
  } = $props()

  let draft = $state<WorkflowLifecycleDraft>({
    agentId: '',
    name: '',
    pickupStatusId: '',
    finishStatusId: '',
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
      draft.pickupStatusId !== baseDraft.pickupStatusId ||
      draft.finishStatusId !== baseDraft.finishStatusId ||
      draft.maxConcurrent !== baseDraft.maxConcurrent ||
      draft.maxRetryAttempts !== baseDraft.maxRetryAttempts ||
      draft.timeoutMinutes !== baseDraft.timeoutMinutes ||
      draft.stallTimeoutMinutes !== baseDraft.stallTimeoutMinutes ||
      draft.isActive !== baseDraft.isActive,
  )
  const selectedPickupStatusLabel = $derived(
    statuses.find((status) => status.id === draft.pickupStatusId)?.name ?? 'Select status',
  )
  const selectedAgentLabel = $derived(
    agentOptions.find((option) => option.id === draft.agentId)?.label ?? 'Select bound agent',
  )
  const selectedFinishStatusLabel = $derived(
    draft.finishStatusId
      ? (statuses.find((status) => status.id === draft.finishStatusId)?.name ?? 'Unknown status')
      : 'Leave unchanged',
  )
  const finishStatusValue = $derived(draft.finishStatusId || unchangedFinishStatusValue)
  const selectedAgent = $derived(agentOptions.find((option) => option.id === draft.agentId) ?? null)
  const machineSummary = $derived(
    selectedAgent?.machineName
      ? `Provider machine: ${selectedAgent.machineName}`
      : 'Select bound agent',
  )

  $effect(() => {
    const nextKey = [
      workflow.id,
      workflow.version,
      workflow.agentId ?? '',
      workflow.name,
      workflow.isActive,
      workflow.pickupStatusId,
      workflow.finishStatusId ?? '',
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

<div class={cn('border-border flex h-full flex-col overflow-y-auto border-l', className)}>
  <WorkflowDetailHeader
    {workflow}
    isActive={draft.isActive}
    disabled={saving || deleting}
    onToggle={() => updateDraftField('isActive', !draft.isActive)}
  />

  <Separator />

  <form class="flex flex-1 flex-col" onsubmit={handleSubmit}>
    <div class="flex-1 space-y-4 px-4 py-4">
      <div class="space-y-2">
        <Label for="workflow-name">Name</Label>
        <Input
          id="workflow-name"
          value={draft.name}
          disabled={saving || deleting}
          oninput={(event) =>
            updateDraftField('name', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="space-y-2">
        <Label>Bound Agent</Label>
        <Select.Root
          type="single"
          value={draft.agentId}
          disabled={saving || deleting || agentOptions.length === 0}
          onValueChange={(value) => updateDraftField('agentId', value || '')}
        >
          <Select.Trigger class="w-full">{selectedAgentLabel}</Select.Trigger>
          <Select.Content>
            {#each agentOptions as option (option.id)}
              <Select.Item value={option.id}>{option.label}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
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
          label="Max Retry Attempts"
          value={draft.maxRetryAttempts}
          icon={RotateCcw}
          disabled={saving || deleting}
          oninput={(value) => updateDraftField('maxRetryAttempts', value)}
        />
        <WorkflowNumberField
          id="workflow-timeout"
          label="Timeout Minutes"
          value={draft.timeoutMinutes}
          icon={Clock3}
          disabled={saving || deleting}
          oninput={(value) => updateDraftField('timeoutMinutes', value)}
        />
        <WorkflowNumberField
          id="workflow-stall-timeout"
          label="Stall Timeout Minutes"
          value={draft.stallTimeoutMinutes}
          icon={Clock3}
          disabled={saving || deleting}
          oninput={(value) => updateDraftField('stallTimeoutMinutes', value)}
        />
      </div>

      <WorkflowBindingSummary
        providerName={selectedAgent?.providerName}
        modelName={selectedAgent?.modelName}
        {machineSummary}
        workspacePath={selectedAgent?.workspacePath}
      />

      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-2">
          <Label>Pickup Status</Label>
          <Select.Root
            type="single"
            value={draft.pickupStatusId}
            disabled={saving || deleting || statuses.length === 0}
            onValueChange={(value) => updateDraftField('pickupStatusId', value || '')}
          >
            <Select.Trigger class="w-full">{selectedPickupStatusLabel}</Select.Trigger>
            <Select.Content>
              {#each statuses as status (status.id)}
                <Select.Item value={status.id}>{status.name}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label>Finish Status</Label>
          <Select.Root
            type="single"
            value={finishStatusValue}
            disabled={saving || deleting}
            onValueChange={(value) =>
              updateDraftField(
                'finishStatusId',
                value === unchangedFinishStatusValue ? '' : (value ?? ''),
              )}
          >
            <Select.Trigger class="w-full">{selectedFinishStatusLabel}</Select.Trigger>
            <Select.Content>
              <Select.Item value={unchangedFinishStatusValue}>Leave unchanged</Select.Item>
              {#each statuses as status (status.id)}
                <Select.Item value={status.id}>{status.name}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>
      </div>
    </div>

    <WorkflowDetailActions
      {statusMessage}
      errorMessage={formError || error}
      {saving}
      {deleting}
      {isDirty}
      onDelete={handleDelete}
    />
  </form>
</div>
