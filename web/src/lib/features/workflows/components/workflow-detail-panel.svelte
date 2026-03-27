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
  import WorkflowDetailActions from './workflow-detail-actions.svelte'
  import WorkflowDetailHeader from './workflow-detail-header.svelte'
  import WorkflowBindingSummary from './workflow-binding-summary.svelte'
  import WorkflowNumberField from './workflow-number-field.svelte'
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
  const selectedAgentLabel = $derived(
    agentOptions.find((option) => option.id === draft.agentId)?.label ?? 'Select bound agent',
  )
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
    updateDraftField('pickupStatusIds', toggleWorkflowStatusSelection(draft.pickupStatusIds, statusId))
  }

  function toggleFinishStatus(statusId: string) {
    updateDraftField('finishStatusIds', toggleWorkflowStatusSelection(draft.finishStatusIds, statusId))
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
      />

      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-2">
          <Label>Pickup Statuses</Label>
          <div class="flex flex-wrap gap-2">
            {#each statuses as status (status.id)}
              <button
                type="button"
                class={cn(
                  'rounded-full border px-3 py-1.5 text-xs transition-colors',
                  draft.pickupStatusIds.includes(status.id)
                    ? 'border-primary/40 bg-primary/10 text-foreground'
                    : 'border-border text-muted-foreground hover:bg-muted',
                )}
                disabled={saving || deleting}
                onclick={() => togglePickupStatus(status.id)}
              >
                {status.name}
              </button>
            {/each}
          </div>
        </div>

        <div class="space-y-2">
          <Label>Finish Statuses</Label>
          <div class="flex flex-wrap gap-2">
            {#each statuses as status (status.id)}
              <button
                type="button"
                class={cn(
                  'rounded-full border px-3 py-1.5 text-xs transition-colors',
                  draft.finishStatusIds.includes(status.id)
                    ? 'border-primary/40 bg-primary/10 text-foreground'
                    : 'border-border text-muted-foreground hover:bg-muted',
                )}
                disabled={saving || deleting}
                onclick={() => toggleFinishStatus(status.id)}
              >
                {status.name}
              </button>
            {/each}
          </div>
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
