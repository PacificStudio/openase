<script lang="ts">
  import { cn } from '$lib/utils'
  import { Separator } from '$ui/separator'
  import { Clock3, Layers3, RotateCcw } from '@lucide/svelte'
  import type {
    ScopeGroup,
    WorkflowAgentOption,
    WorkflowStatusOption,
    WorkflowSummary,
  } from '../types'
  import {
    buildPickupStatusBlockedReasonMap,
    buildSelfStatusBlockedReasonMap,
    createWorkflowLifecycleDraft,
    mergeStatusBlockedReasonMaps,
    parseWorkflowLifecycleDraft,
    toggleWorkflowStatusSelection,
    type WorkflowLifecycleDraft,
    type WorkflowLifecyclePayload,
  } from '../workflow-lifecycle'
  import {
    createWorkflowHooksDraft,
    parseWorkflowHooksDraft,
    validateWorkflowHooksDraft,
    type WorkflowHooksDraft,
  } from '../workflow-hooks'
  import WorkflowDetailActions from './workflow-detail-actions.svelte'
  import WorkflowDetailHeader from './workflow-detail-header.svelte'
  import WorkflowDetailHistorySection from './workflow-detail-history-section.svelte'
  import WorkflowDetailIdentitySection from './workflow-detail-identity-section.svelte'
  import WorkflowDetailHooksSection from './workflow-detail-hooks-section.svelte'
  import WorkflowNumberField from './workflow-number-field.svelte'
  import {
    isWorkflowLifecycleDraftDirty,
    workflowLifecycleDraftKey,
  } from './workflow-detail-panel-state'
  import WorkflowStatusChipSelector from './workflow-status-chip-selector.svelte'
  let {
    workflow,
    workflows = [],
    statuses = [],
    agentOptions = [],
    scopeGroups = [],
    saving = false,
    deleting = false,
    onSave,
    onDelete,
    class: className = '',
  }: {
    workflow: WorkflowSummary
    workflows?: WorkflowSummary[]
    statuses?: WorkflowStatusOption[]
    agentOptions?: WorkflowAgentOption[]
    scopeGroups?: ScopeGroup[]
    saving?: boolean
    deleting?: boolean
    onSave?: (payload: WorkflowLifecyclePayload) => void | Promise<void>
    onDelete?: () => void | Promise<void>
    class?: string
  } = $props()

  let draft = $state<WorkflowLifecycleDraft>({
    agentId: '',
    name: '',
    typeLabel: '',
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
    isWorkflowLifecycleDraftDirty(draft, baseDraft, hookDraft, baseHookDraft),
  )
  const selectedAgent = $derived(agentOptions.find((option) => option.id === draft.agentId) ?? null)
  const selectableStatuses = $derived(statuses)
  const pickupBlockedReasonMap = $derived(
    mergeStatusBlockedReasonMaps(
      buildPickupStatusBlockedReasonMap(workflows, workflow.id),
      buildSelfStatusBlockedReasonMap(
        draft.finishStatusIds,
        'Already selected as a finish status in this workflow.',
      ),
    ),
  )
  const finishBlockedReasonMap = $derived(
    buildSelfStatusBlockedReasonMap(
      draft.pickupStatusIds,
      'Already selected as a pickup status in this workflow.',
    ),
  )

  $effect(() => {
    const nextKey = workflowLifecycleDraftKey(workflow)

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
    const message = workflow.isActive
      ? `Retire workflow "${workflow.name}"? It will stop participating in new pickup and scheduling.`
      : `Delete workflow "${workflow.name}" permanently? This cannot be undone.`
    if (!confirm(message)) return
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
        {scopeGroups}
        onFieldChange={(field, value) => updateDraftField(field, value)}
      />

      <Separator />

      <div class="space-y-2">
        <span class="text-muted-foreground text-xs font-medium tracking-wide uppercase">Limits</span
        >
        <div class="bg-muted/40 divide-border space-y-0 divide-y rounded-md border px-3 py-0.5">
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
          disabledReasonById={pickupBlockedReasonMap}
          disabled={saving || deleting}
          onToggle={(statusId) =>
            updateDraftField(
              'pickupStatusIds',
              toggleWorkflowStatusSelection(
                draft.pickupStatusIds,
                statusId,
                pickupBlockedReasonMap,
              ),
            )}
        />
        <WorkflowStatusChipSelector
          label="Finish Statuses"
          statuses={selectableStatuses}
          selectedStatusIds={draft.finishStatusIds}
          disabledReasonById={finishBlockedReasonMap}
          disabled={saving || deleting}
          onToggle={(statusId) =>
            updateDraftField(
              'finishStatusIds',
              toggleWorkflowStatusSelection(
                draft.finishStatusIds,
                statusId,
                finishBlockedReasonMap,
              ),
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
      isActive={workflow.isActive}
      onDelete={handleDelete}
    />
  </form>
</div>
