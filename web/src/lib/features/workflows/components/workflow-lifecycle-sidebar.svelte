<script lang="ts">
  import { toastStore } from '$lib/stores/toast.svelte'
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
      toastStore.success('Workflow updated.')
    } catch (caughtError) {
      toastStore.error(describeWorkflowApiError(caughtError, 'Failed to update workflow.'))
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
        toastStore.success(
          'Workflow retired. It will no longer receive new pickup or scheduled traffic.',
        )
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
      toastStore.success('Workflow deleted.')
    } catch (caughtError) {
      const impact = workflowImpactFromError(caughtError)
      if (impact) {
        toastStore.error(describeWorkflowImpact(impact))
        return
      }
      toastStore.error(describeWorkflowApiError(caughtError, 'Failed to delete workflow.'))
    } finally {
      deleting = false
    }
  }

  function promptForReplacementWorkflow() {
    const replacementChoices = workflows.filter((item) => item.id !== workflow.id && item.isActive)
    if (replacementChoices.length === 0) {
      return null
    }

    const message = [
      'This workflow still has replaceable ticket or scheduled job references.',
      'Enter the replacement workflow ID to migrate those references before purge.',
      '',
      ...replacementChoices.map((item) => `${item.id}  ${item.name}`),
    ].join('\n')

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
      moved.push(`${result.ticket_count} active tickets`)
    }
    if (result.scheduled_job_count > 0) {
      moved.push(`${result.scheduled_job_count} scheduled jobs`)
    }
    const summary = moved.length > 0 ? moved.join(', ') : 'No references'

    if (canPurge) {
      return `${summary} moved to the replacement workflow. Purging will continue.`
    }
    return `${summary} moved to the replacement workflow.`
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
