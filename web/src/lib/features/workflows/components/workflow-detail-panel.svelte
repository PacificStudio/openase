<script lang="ts">
  import { cn } from '$lib/utils'
  import { Separator } from '$ui/separator'
  import { Clock3, Layers3, RotateCcw } from '@lucide/svelte'
  import type { WorkflowAgentOption, WorkflowStatusOption, WorkflowSummary } from '../types'
  import {
    createWorkflowLifecycleDraft,
    parseWorkflowLifecycleDraft,
    toggleWorkflowStatusSelection,
    type WorkflowLifecycleDraft,
    type WorkflowLifecyclePayload,
  } from '../workflow-lifecycle'
  import {
    createWorkflowHooksDraft,
    parseWorkflowHooksDraft,
    validateWorkflowHooksDraft,
    workflowHooksDraftSignature,
    type WorkflowHooksDraft,
  } from '../workflow-hooks'
  import WorkflowDetailActions from './workflow-detail-actions.svelte'
  import WorkflowDetailHeader from './workflow-detail-header.svelte'
  import WorkflowDetailHistorySection from './workflow-detail-history-section.svelte'
  import WorkflowDetailIdentitySection from './workflow-detail-identity-section.svelte'
  import WorkflowDetailHooksSection from './workflow-detail-hooks-section.svelte'
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
    roleSlug: '',
    roleName: '',
    roleDescription: '',
    platformAccessAllowed: '',
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
  let hookDraft = $state<WorkflowHooksDraft>(createWorkflowHooksDraft())

  const baseDraft = $derived(createWorkflowLifecycleDraft(workflow))
  const baseHookDraft = $derived(createWorkflowHooksDraft(workflow.rawHooks ?? workflow.hooks))
  const hookValidation = $derived(validateWorkflowHooksDraft(hookDraft))
  const isDirty = $derived(
    draft.agentId !== baseDraft.agentId ||
      draft.name !== baseDraft.name ||
      draft.roleSlug !== baseDraft.roleSlug ||
      draft.roleName !== baseDraft.roleName ||
      draft.roleDescription !== baseDraft.roleDescription ||
      draft.platformAccessAllowed !== baseDraft.platformAccessAllowed ||
      draft.pickupStatusIds.join(':') !== baseDraft.pickupStatusIds.join(':') ||
      draft.finishStatusIds.join(':') !== baseDraft.finishStatusIds.join(':') ||
      draft.maxConcurrent !== baseDraft.maxConcurrent ||
      draft.maxRetryAttempts !== baseDraft.maxRetryAttempts ||
      draft.timeoutMinutes !== baseDraft.timeoutMinutes ||
      draft.stallTimeoutMinutes !== baseDraft.stallTimeoutMinutes ||
      draft.isActive !== baseDraft.isActive ||
      workflowHooksDraftSignature(hookDraft) !== workflowHooksDraftSignature(baseHookDraft),
  )
  const selectedAgent = $derived(agentOptions.find((option) => option.id === draft.agentId) ?? null)
  const selectableStatuses = $derived(statuses)

  $effect(() => {
    const nextKey = [
      workflow.id,
      workflow.version,
      workflow.agentId ?? '',
      workflow.name,
      workflow.roleSlug,
      workflow.roleName,
      workflow.roleDescription,
      (workflow.platformAccessAllowed ?? []).join(','),
      workflow.isActive,
      workflow.pickupStatusIds.join(','),
      workflow.finishStatusIds.join(','),
      workflow.maxConcurrent,
      workflow.maxRetry,
      workflow.timeoutMinutes,
      workflow.stallTimeoutMinutes,
      JSON.stringify(workflow.rawHooks ?? workflow.hooks ?? {}),
    ].join(':')

    if (nextKey === draftKey) return

    draft = createWorkflowLifecycleDraft(workflow)
    hookDraft = createWorkflowHooksDraft(workflow.rawHooks ?? workflow.hooks)
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

    const parsedHooks = parseWorkflowHooksDraft(hookDraft)
    if (!parsedHooks.ok) {
      formError = parsedHooks.error
      return
    }

    await onSave?.({
      ...parsed.value,
      hooks: parsedHooks.value,
    })
  }

  async function handleDelete() {
    formError = ''
    if (!confirm(`Delete workflow "${workflow.name}"? This cannot be undone.`)) return
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

  <WorkflowDetailHistorySection {workflow} />

  <form class="flex flex-1 flex-col" onsubmit={handleSubmit}>
    <div class="flex-1 space-y-6 px-4 py-4">
      <WorkflowDetailIdentitySection
        {draft}
        {saving}
        {deleting}
        {agentOptions}
        {selectedAgent}
        onFieldChange={(field, value) => updateDraftField(field, value)}
      />

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
            placeholder="Unlimited"
            min="1"
            disabled={saving || deleting}
            oninput={(value) => updateDraftField('maxConcurrent', value)}
          />
          <WorkflowNumberField
            id="workflow-max-retry"
            label="Max Retry"
            value={draft.maxRetryAttempts}
            icon={RotateCcw}
            min="0"
            disabled={saving || deleting}
            oninput={(value) => updateDraftField('maxRetryAttempts', value)}
          />
          <WorkflowNumberField
            id="workflow-timeout"
            label="Timeout (min)"
            value={draft.timeoutMinutes}
            icon={Clock3}
            min="1"
            disabled={saving || deleting}
            oninput={(value) => updateDraftField('timeoutMinutes', value)}
          />
          <WorkflowNumberField
            id="workflow-stall-timeout"
            label="Stall Timeout (min)"
            value={draft.stallTimeoutMinutes}
            icon={Clock3}
            min="1"
            disabled={saving || deleting}
            oninput={(value) => updateDraftField('stallTimeoutMinutes', value)}
          />
        </div>
      </div>

      <Separator />

      <div class="space-y-4">
        <WorkflowStatusChipSelector
          label="Pickup Statuses"
          statuses={selectableStatuses}
          selectedStatusIds={draft.pickupStatusIds}
          disabled={saving || deleting}
          onToggle={(statusId) =>
            updateDraftField(
              'pickupStatusIds',
              toggleWorkflowStatusSelection(draft.pickupStatusIds, statusId),
            )}
        />
        <WorkflowStatusChipSelector
          label="Finish Statuses"
          statuses={selectableStatuses}
          selectedStatusIds={draft.finishStatusIds}
          disabled={saving || deleting}
          onToggle={(statusId) =>
            updateDraftField(
              'finishStatusIds',
              toggleWorkflowStatusSelection(draft.finishStatusIds, statusId),
            )}
        />
      </div>

      <Separator />

      <WorkflowDetailHooksSection
        draft={hookDraft}
        validation={hookValidation}
        disabled={saving || deleting}
        onChange={(nextDraft) => {
          hookDraft = nextDraft
          formError = ''
        }}
      />
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
