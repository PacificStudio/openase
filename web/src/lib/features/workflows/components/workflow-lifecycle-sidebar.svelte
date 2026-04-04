<script lang="ts">
  import { toastStore } from '$lib/stores/toast.svelte'
  import type {
    ScopeGroup,
    WorkflowAgentOption,
    WorkflowStatusOption,
    WorkflowSummary,
  } from '../types'
  import type { WorkflowLifecyclePayload } from '../workflow-lifecycle'
  import {
    destroyWorkflow,
    loadWorkflowImpact,
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
        toastStore.error(describeWorkflowImpact(impact))
        return
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
