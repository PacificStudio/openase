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
  import {
    REQUIRED_WORKFLOW_PLATFORM_SCOPE,
    REQUIRED_WORKFLOW_SKILL_NAME,
  } from '../workflow-requirements'
  import WorkflowCreationAdvancedSection from './workflow-creation-advanced-section.svelte'
  import WorkflowStatusChipGroup from './workflow-status-chip-group.svelte'
  import type { TranslationKey, TranslationParams } from '$lib/i18n/index'
  import { i18nStore } from '$lib/i18n/store.svelte'

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
    agentOptions.find((option) => option.id === agentId)?.label ??
      t('workflows.agentSelect.trigger.placeholder'),
  )
  const pickupBlockedReasonMap = $derived(
    mergeStatusBlockedReasonMaps(
      buildPickupStatusBlockedReasonMap(workflows),
      buildSelfStatusBlockedReasonMap(
        finishStatusIds,
        t('workflows.creation.dialog.statusBlock.finishSelected'),
      ),
    ),
  )
  const finishBlockedReasonMap = $derived(
    buildSelfStatusBlockedReasonMap(
      pickupStatusIds,
      t('workflows.creation.dialog.statusBlock.pickupSelected'),
    ),
  )
  const hookValidation = $derived(validateWorkflowHooksDraft(hookDraft))

  function t(key: TranslationKey, params?: TranslationParams) {
    return i18nStore.t(key, params)
  }

  $effect(() => {
    if (open && !wasOpen) {
      name = templateDraft?.name ?? `Workflow ${existingCount + 1}`
      typeLabel =
        templateDraft?.workflowType ?? t('workflows.creation.dialog.defaults.workflowTypeLabel')
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
      toastStore.error(t('workflows.creation.dialog.errors.noProject'))
      return
    }
    if (!name.trim()) {
      toastStore.error(t('workflows.creation.dialog.errors.nameRequired'))
      return
    }
    if (!agentId) {
      toastStore.error(t('workflows.creation.dialog.errors.agentRequired'))
      return
    }
    if (!typeLabel.trim()) {
      toastStore.error(t('workflows.creation.dialog.errors.typeRequired'))
      return
    }
    if (templateStatusError) {
      toastStore.error(templateStatusError)
      return
    }
    if (pickupStatusIds.length === 0 || finishStatusIds.length === 0) {
      toastStore.error(t('workflows.creation.dialog.errors.statusRequired'))
      return
    }
    if (findOverlappingStatusIds(pickupStatusIds, finishStatusIds).length > 0) {
      toastStore.error(t('workflows.creation.dialog.errors.statusExclusive'))
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
      toastStore.error(
        describeWorkflowApiError(caughtError, t('workflows.creation.dialog.errors.creationFailed')),
      )
    } finally {
      saving = false
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <Dialog.Title>{t('workflows.creation.dialog.title')}</Dialog.Title>
      <Dialog.Description>
        {t('workflows.creation.dialog.description')}
      </Dialog.Description>
    </Dialog.Header>

    <form class="flex min-h-0 flex-1 flex-col gap-6" onsubmit={handleSubmit}>
      <Dialog.Body class="space-y-4">
        <div class="space-y-2">
          <Label for="workflow-create-name">
            {t('workflows.creation.dialog.labels.name')}
          </Label>
          <Input
            id="workflow-create-name"
            bind:value={name}
            disabled={saving}
            placeholder={t('workflows.creation.dialog.placeholders.name')}
          />
        </div>

        <div class="space-y-2">
          <Label for="workflow-create-type">
            {t('workflows.creation.dialog.labels.typeLabel')}
          </Label>
          <Input
            id="workflow-create-type"
            bind:value={typeLabel}
            disabled={saving}
            placeholder={t('workflows.creation.dialog.placeholders.typeLabel')}
          />
        </div>

        <div class="space-y-2">
          <Label>{t('workflows.creation.dialog.labels.boundAgent')}</Label>
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
            label={t('workflows.creation.dialog.labels.pickupStatuses')}
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
            label={t('workflows.creation.dialog.labels.finishStatuses')}
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

        <div class="bg-muted/40 rounded-md border px-3 py-2 text-xs leading-relaxed">
          <span class="font-medium">
            {t('workflows.creation.dialog.runtimeAccess.heading')}
          </span>
          {t('workflows.creation.dialog.runtimeAccess.description.prefix')}
          <code class="bg-background rounded px-1 py-0.5 font-mono"
            >{REQUIRED_WORKFLOW_PLATFORM_SCOPE}</code
          >
          {t('workflows.creation.dialog.runtimeAccess.description.middle')}
          <code class="bg-background rounded px-1 py-0.5 font-mono"
            >{REQUIRED_WORKFLOW_SKILL_NAME}</code
          >
          {t('workflows.creation.dialog.runtimeAccess.description.suffix')}
        </div>

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
      </Dialog.Body>

      <Dialog.Footer showCloseButton>
        <Button type="submit" disabled={saving || !projectId || !!templateStatusError}>
          {saving
            ? t('workflows.creation.dialog.actions.creating')
            : t('workflows.creation.dialog.actions.create')}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
