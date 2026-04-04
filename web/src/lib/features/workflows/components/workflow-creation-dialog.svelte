<script lang="ts">
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { createWorkflowWithBinding } from '../data'
  import { resolveTemplateStatusSelection } from '../model'
  import {
    createWorkflowHooksDraft,
    parseWorkflowHooksDraft,
    validateWorkflowHooksDraft,
    type WorkflowHooksDraft,
  } from '../workflow-hooks'
  import {
    buildDispatcherFinishStatusIds,
    buildPickupStatusBlockedReasonMap,
    buildSelfStatusBlockedReasonMap,
    findOverlappingStatusIds,
    mergeStatusBlockedReasonMaps,
    toggleWorkflowStatusSelection,
  } from '../workflow-lifecycle'
  import { describeWorkflowApiError } from '../workflow-api-errors'
  import type {
    WorkflowAgentOption,
    WorkflowStatusOption,
    WorkflowSummary,
    WorkflowTemplateDraft,
  } from '../types'
  import WorkflowCreationAdvancedSection from './workflow-creation-advanced-section.svelte'
  import WorkflowStatusChipGroup from './workflow-status-chip-group.svelte'

  let {
    open = $bindable(false),
    projectId,
    statuses,
    agentOptions,
    workflows = [],
    existingCount,
    builtinRoleContent,
    templateDraft = null,
    onCreated,
  }: {
    open?: boolean
    projectId: string
    statuses: WorkflowStatusOption[]
    agentOptions: WorkflowAgentOption[]
    workflows?: WorkflowSummary[]
    existingCount: number
    builtinRoleContent: string
    templateDraft?: WorkflowTemplateDraft | null
    onCreated?: (payload: { workflow: WorkflowSummary; selectedId: string }) => void
  } = $props()

  let saving = $state(false)
  let name = $state('')
  let typeLabel = $state('')
  let agentId = $state('')
  let pickupStatusIds = $state<string[]>([])
  let finishStatusIds = $state<string[]>([])
  let templateStatusError = $state('')
  let advancedOpen = $state(false)
  let hookDraft = $state<WorkflowHooksDraft>(createWorkflowHooksDraft())
  let hookError = $state('')
  let wasOpen = false

  const selectedAgentLabel = $derived(
    agentOptions.find((option) => option.id === agentId)?.label ?? 'Select bound agent',
  )
  const pickupBlockedReasonMap = $derived(
    mergeStatusBlockedReasonMaps(
      buildPickupStatusBlockedReasonMap(workflows),
      buildSelfStatusBlockedReasonMap(
        finishStatusIds,
        'Already selected as a finish status in this workflow.',
      ),
    ),
  )
  const finishBlockedReasonMap = $derived(
    buildSelfStatusBlockedReasonMap(
      pickupStatusIds,
      'Already selected as a pickup status in this workflow.',
    ),
  )
  const hookValidation = $derived(validateWorkflowHooksDraft(hookDraft))

  $effect(() => {
    if (open && !wasOpen) {
      name = templateDraft?.name ?? `Workflow ${existingCount + 1}`
      typeLabel = templateDraft?.workflowType ?? 'Workflow'
      agentId = agentOptions[0]?.id ?? ''
      templateStatusError = ''
      advancedOpen = false
      hookDraft = createWorkflowHooksDraft()
      hookError = ''

      const blockedNow = buildPickupStatusBlockedReasonMap(workflows)
      const defaultPickupStatusId = statuses.find((status) => !blockedNow[status.id])?.id ?? ''
      const defaultFinishStatusId =
        statuses.find((status) => status.id !== defaultPickupStatusId)?.id ?? ''
      pickupStatusIds = defaultPickupStatusId ? [defaultPickupStatusId] : []
      finishStatusIds = defaultFinishStatusId ? [defaultFinishStatusId] : []

      if (templateDraft) {
        const templateSelection = resolveTemplateStatusSelection(
          templateDraft.pickupStatusNames ?? [],
          templateDraft.roleSlug === 'dispatcher' ? [] : (templateDraft.finishStatusNames ?? []),
          statuses,
        )
        templateStatusError = templateSelection.error
        if (templateSelection.pickupStatusIds.length > 0 || templateSelection.error) {
          pickupStatusIds = templateSelection.pickupStatusIds.filter((id) => !blockedNow[id])
        }
        if (templateDraft.roleSlug === 'dispatcher') {
          finishStatusIds = buildDispatcherFinishStatusIds(statuses, workflows, pickupStatusIds)
        } else if (templateSelection.finishStatusIds.length > 0 || templateSelection.error) {
          finishStatusIds = templateSelection.finishStatusIds
        }
      }
    }

    wasOpen = open
  })

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()
    if (!projectId) {
      toastStore.error('Select a project before creating a workflow.')
      return
    }
    if (!name.trim()) {
      toastStore.error('Workflow name is required.')
      return
    }
    if (!agentId) {
      toastStore.error('Bound agent is required.')
      return
    }
    if (!typeLabel.trim()) {
      toastStore.error('Workflow type label is required.')
      return
    }
    if (templateStatusError) {
      toastStore.error(templateStatusError)
      return
    }
    if (pickupStatusIds.length === 0 || finishStatusIds.length === 0) {
      toastStore.error('Pickup and finish status are required.')
      return
    }
    if (findOverlappingStatusIds(pickupStatusIds, finishStatusIds).length > 0) {
      toastStore.error('Pickup and finish statuses must be mutually exclusive.')
      return
    }

    const parsedHooks = parseWorkflowHooksDraft(hookDraft)
    if (!parsedHooks.ok) {
      hookError = parsedHooks.error
      advancedOpen = true
      return
    }

    saving = true

    try {
      const payload = await createWorkflowWithBinding(
        projectId,
        {
          agentId,
          name: name.trim(),
          workflowType: typeLabel.trim(),
          roleSlug: templateDraft?.roleSlug ?? '',
          roleName: templateDraft?.roleName ?? name.trim(),
          roleDescription: templateDraft?.roleDescription ?? '',
          platformAccessAllowed: templateDraft?.platformAccessAllowed ?? [],
          skillNames: templateDraft?.skillNames ?? [],
          harnessPath: templateDraft?.harnessPath ?? null,
          pickupStatusIds,
          finishStatusIds,
          hooks: parsedHooks.value,
        },
        statuses,
        templateDraft?.content ?? builtinRoleContent,
      )
      onCreated?.(payload)
      open = false
    } catch (caughtError) {
      toastStore.error(describeWorkflowApiError(caughtError, 'Failed to create workflow.'))
    } finally {
      saving = false
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <Dialog.Title>Create Workflow</Dialog.Title>
      <Dialog.Description>
        Bind a workflow to an explicit agent definition before it can dispatch.
      </Dialog.Description>
    </Dialog.Header>

    <form class="space-y-4" onsubmit={handleSubmit}>
      <div class="space-y-2">
        <Label for="workflow-create-name">Name</Label>
        <Input
          id="workflow-create-name"
          bind:value={name}
          disabled={saving}
          placeholder="Workflow name"
        />
      </div>

      <div class="space-y-2">
        <Label for="workflow-create-type">Type Label</Label>
        <Input
          id="workflow-create-type"
          bind:value={typeLabel}
          disabled={saving}
          placeholder="Fullstack Developer"
        />
      </div>

      <div class="space-y-2">
        <Label>Bound Agent</Label>
        <Select.Root
          type="single"
          value={agentId}
          disabled={saving || agentOptions.length === 0}
          onValueChange={(value) => (agentId = value || '')}
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
        <WorkflowStatusChipGroup
          label="Pickup Statuses"
          {statuses}
          selectedIds={pickupStatusIds}
          disabled={saving}
          disabledReasonById={pickupBlockedReasonMap}
          onToggle={(statusId) =>
            (pickupStatusIds = toggleWorkflowStatusSelection(
              pickupStatusIds,
              statusId,
              pickupBlockedReasonMap,
            ))}
        />

        <WorkflowStatusChipGroup
          label="Finish Statuses"
          {statuses}
          selectedIds={finishStatusIds}
          disabled={saving}
          disabledReasonById={finishBlockedReasonMap}
          onToggle={(statusId) =>
            (finishStatusIds = toggleWorkflowStatusSelection(
              finishStatusIds,
              statusId,
              finishBlockedReasonMap,
            ))}
        />
      </div>

      {#if templateStatusError}
        <p class="text-destructive text-xs">{templateStatusError}</p>
      {/if}
      <WorkflowCreationAdvancedSection
        bind:open={advancedOpen}
        draft={hookDraft}
        validation={hookValidation}
        disabled={saving}
        error={hookError}
        onChange={(nextDraft) => {
          hookDraft = nextDraft
          hookError = ''
        }}
      />

      <Dialog.Footer showCloseButton>
        <Button type="submit" disabled={saving || !projectId || !!templateStatusError}>
          {saving ? 'Creating…' : 'Create workflow'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
