<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { WorkflowAgentOption, WorkflowStatusOption, WorkflowSummary } from '../types'
  import type { WorkflowLifecyclePayload } from '../workflow-lifecycle'
  import {
    destroyWorkflow,
    removeWorkflowFromList,
    saveWorkflowLifecycle,
  } from '../workflow-management'
  import WorkflowDetailPanel from './workflow-detail-panel.svelte'

  let {
    workflow,
    workflows,
    statuses,
    agentOptions = [],
    class: className = '',
    onWorkflowsChange,
    onSelectedIdChange,
  }: {
    workflow: WorkflowSummary
    workflows: WorkflowSummary[]
    statuses: WorkflowStatusOption[]
    agentOptions?: WorkflowAgentOption[]
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
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update workflow.',
      )
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
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete workflow.',
      )
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
  {saving}
  {deleting}
  class={className}
  onSave={(payload) => void handleSave(payload)}
  onDelete={() => void handleDelete()}
/>
