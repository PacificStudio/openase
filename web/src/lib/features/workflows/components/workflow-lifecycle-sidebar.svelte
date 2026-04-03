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
    removeWorkflowFromList,
    saveWorkflowLifecycle,
  } from '../workflow-management'
  import { describeWorkflowApiError } from '../workflow-api-errors'
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
      await destroyWorkflow(workflow.id)
      const nextState = removeWorkflowFromList(workflows, workflow.id)
      onWorkflowsChange?.(nextState.remaining)
      onSelectedIdChange?.(nextState.nextSelectedId)
      toastStore.success('Workflow deleted.')
    } catch (caughtError) {
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
