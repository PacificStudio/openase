<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { WorkflowStatusOption, WorkflowSummary } from '../types'
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
    class: className = '',
    onWorkflowsChange,
    onSelectedIdChange,
  }: {
    workflow: WorkflowSummary
    workflows: WorkflowSummary[]
    statuses: WorkflowStatusOption[]
    class?: string
    onWorkflowsChange?: (workflows: WorkflowSummary[]) => void
    onSelectedIdChange?: (selectedId: string) => void
  } = $props()

  let saving = $state(false)
  let deleting = $state(false)
  let error = $state('')
  let statusMessage = $state('')

  async function handleSave(payload: WorkflowLifecyclePayload) {
    saving = true
    error = ''
    statusMessage = ''

    try {
      const updated = await saveWorkflowLifecycle(workflow.id, payload, statuses)
      onWorkflowsChange?.(workflows.map((item) => (item.id === updated.id ? updated : item)))
      statusMessage = 'Workflow updated.'
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to update workflow.'
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    deleting = true
    error = ''
    statusMessage = ''

    try {
      await destroyWorkflow(workflow.id)
      const nextState = removeWorkflowFromList(workflows, workflow.id)
      onWorkflowsChange?.(nextState.remaining)
      onSelectedIdChange?.(nextState.nextSelectedId)
      statusMessage = 'Workflow deleted.'
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete workflow.'
    } finally {
      deleting = false
    }
  }
</script>

<WorkflowDetailPanel
  {workflow}
  {statuses}
  {saving}
  {deleting}
  {statusMessage}
  {error}
  class={className}
  onSave={(payload) => void handleSave(payload)}
  onDelete={() => void handleDelete()}
/>
