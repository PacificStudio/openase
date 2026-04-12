<script lang="ts">
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { TranslationKey, TranslationParams } from '$lib/i18n/index'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import type {
    ScopeGroup,
    WorkflowAgentOption,
    WorkflowReplaceReferencesResult,
    WorkflowStatusOption,
    WorkflowSummary,
  } from '../types'
  import type { WorkflowLifecyclePayload } from '../workflow-lifecycle'
  import {
    destroyWorkflow,
    loadWorkflowImpact,
    replaceWorkflowLifecycleReferences,
    removeWorkflowFromList,
    retireWorkflowLifecycle,
    saveWorkflowLifecycle,
  } from '../workflow-management'
  import {
    describeWorkflowApiError,
    describeWorkflowImpact,
    workflowImpactFromError,
  } from '../workflow-api-errors'
  import WorkflowDetailPanel from './workflow-detail-panel.svelte'

  function t(key: TranslationKey, params?: TranslationParams) {
    return i18nStore.t(key, params)
  }

  let {
    workflow,
    workflows,
    statuses,
    agentOptions = [],
    scopeGroups = [],
    class: className = '',
    onWorkflowsChange,
    onSelectedIdChange,
  }: {
    workflow: WorkflowSummary
    workflows: WorkflowSummary[]
    statuses: WorkflowStatusOption[]
    agentOptions?: WorkflowAgentOption[]
    scopeGroups?: ScopeGroup[]
    class?: string
    onWorkflowsChange?: (workflows: WorkflowSummary[]) => void
    onSelectedIdChange?: (selectedId: string) => void
  } = $props()

  let saving = $state(false)
  let deleting = $state(false)

  async function handleSave(payload: WorkflowLifecyclePayload) {
    saving = true

    try {
      const updated = await saveWorkflowLifecycle(workflow.id, payload, statuses, workflow)
      onWorkflowsChange?.(workflows.map((item) => (item.id === updated.id ? updated : item)))
      toastStore.success(t('workflows.lifecycle.sidebar.toast.updated'))
    } catch (caughtError) {
      toastStore.error(
        describeWorkflowApiError(caughtError, t('workflows.lifecycle.sidebar.toast.updateFailed')),
      )
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    deleting = true

    try {
      if (workflow.isActive) {
        const updated = await retireWorkflowLifecycle(workflow.id, statuses, workflow)
        onWorkflowsChange?.(workflows.map((item) => (item.id === updated.id ? updated : item)))
        toastStore.success(t('workflows.lifecycle.sidebar.toast.retired'))
        return
      }

      const impact = await loadWorkflowImpact(workflow.id)
      if (!impact.can_purge) {
        if (!impact.can_replace_references) {
          toastStore.error(describeWorkflowImpact(impact))
          return
        }

        const replacementId = promptForReplacementWorkflow()
        if (!replacementId) {
          toastStore.error(describeWorkflowImpact(impact))
          return
        }

        const result = await replaceWorkflowLifecycleReferences(workflow.id, replacementId)
        const nextImpact = await loadWorkflowImpact(workflow.id)
        toastStore.success(describeReplacementResult(result, nextImpact.can_purge))
        if (!nextImpact.can_purge) {
          toastStore.error(describeWorkflowImpact(nextImpact))
          return
        }
      }

      await destroyWorkflow(workflow.id)
      const nextState = removeWorkflowFromList(workflows, workflow.id)
      onWorkflowsChange?.(nextState.remaining)
      onSelectedIdChange?.(nextState.nextSelectedId)
      toastStore.success(t('workflows.lifecycle.sidebar.toast.deleted'))
    } catch (caughtError) {
      const impact = workflowImpactFromError(caughtError)
      if (impact) {
        toastStore.error(describeWorkflowImpact(impact))
        return
      }
      toastStore.error(
        describeWorkflowApiError(caughtError, t('workflows.lifecycle.sidebar.toast.deleteFailed')),
      )
    } finally {
      deleting = false
    }
  }

  function promptForReplacementWorkflow() {
    const replacementChoices = workflows.filter((item) => item.id !== workflow.id && item.isActive)
    if (replacementChoices.length === 0) {
      return null
    }

    const messageLines = [
      t('workflows.lifecycle.sidebar.prompt.replacementReferences'),
      t('workflows.lifecycle.sidebar.prompt.replacementInstructions'),
      '',
      ...replacementChoices.map((item) => `${item.id}  ${item.name}`),
    ]
    const message = messageLines.join('\n')

    const input = window.prompt(message, replacementChoices[0]?.id ?? '')
    const replacementId = input?.trim()
    if (!replacementId) {
      return null
    }
    return replacementId
  }

  function describeReplacementResult(result: WorkflowReplaceReferencesResult, canPurge: boolean) {
    const moved: string[] = []
    if (result.ticket_count > 0) {
      moved.push(
        t('workflows.lifecycle.sidebar.replacementResult.activeTickets', {
          count: result.ticket_count,
        }),
      )
    }
    if (result.scheduled_job_count > 0) {
      moved.push(
        t('workflows.lifecycle.sidebar.replacementResult.scheduledJobs', {
          count: result.scheduled_job_count,
        }),
      )
    }
    const summary =
      moved.length > 0 ? moved.join(', ') : t('workflows.lifecycle.sidebar.replacementResult.none')

    if (canPurge) {
      return t('workflows.lifecycle.sidebar.replacementResult.canPurge', { summary })
    }
    return t('workflows.lifecycle.sidebar.replacementResult.cannotPurge', { summary })
  }
</script>

<WorkflowDetailPanel
  {workflow}
  {workflows}
  {statuses}
  {agentOptions}
  {scopeGroups}
  {saving}
  {deleting}
  class={className}
  onSave={(payload) => void handleSave(payload)}
  onDelete={() => void handleDelete()}
/>
