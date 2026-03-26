<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import {
    capabilityStateClasses,
    capabilityStateLabel,
    getSettingsSectionCapability,
  } from '$lib/features/capabilities'
  import {
    loadWorkflowCatalog,
    WorkflowLifecycleSidebar,
    WorkflowList,
    type WorkflowAgentOption,
    type WorkflowStatusOption,
    type WorkflowSummary,
  } from '$lib/features/workflows'
  import { appStore } from '$lib/stores/app.svelte'
  import { Separator } from '$ui/separator'
  import WorkflowScheduledJobsPanel from './workflow-scheduled-jobs-panel.svelte'

  const workflowCapability = getSettingsSectionCapability('workflows')

  let loading = $state(false)
  let error = $state('')
  let workflows = $state<WorkflowSummary[]>([])
  let agentOptions = $state<WorkflowAgentOption[]>([])
  let statuses = $state<WorkflowStatusOption[]>([])
  let selectedId = $state('')

  let selectedWorkflow = $derived(workflows.find((workflow) => workflow.id === selectedId) ?? null)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      workflows = []
      agentOptions = []
      statuses = []
      selectedId = ''
      error = ''
      loading = false
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const payload = await loadWorkflowCatalog(projectId, orgId)
        if (cancelled) return

        workflows = payload.workflows
        agentOptions = payload.agentOptions
        statuses = payload.statuses
        if (!selectedId || !payload.workflows.some((workflow) => workflow.id === selectedId)) {
          selectedId = payload.workflows[0]?.id ?? ''
        }
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load workflows.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })
</script>

<div class="space-y-6">
  <div>
    <div class="flex items-center gap-2">
      <h2 class="text-foreground text-base font-semibold">Workflow Lifecycle</h2>
      <span
        class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(workflowCapability.state)}`}
      >
        {capabilityStateLabel(workflowCapability.state)}
      </span>
    </div>
    <p class="text-muted-foreground mt-1 text-sm">{workflowCapability.summary}</p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading workflows…</div>
  {:else if error && workflows.length === 0}
    <div class="text-destructive text-sm">{error}</div>
  {:else if workflows.length === 0}
    <div class="text-muted-foreground text-sm">No workflows found in the current project.</div>
  {:else}
    <div class="space-y-6">
      <div class="border-border flex min-h-[34rem] overflow-hidden rounded-lg border">
        <div class="w-64 shrink-0">
          <WorkflowList {workflows} {selectedId} onselect={(id) => (selectedId = id)} />
        </div>
        {#if selectedWorkflow}
          <div class="min-w-0 flex-1">
            <WorkflowLifecycleSidebar
              workflow={selectedWorkflow}
              {workflows}
              {statuses}
              {agentOptions}
              onWorkflowsChange={(nextWorkflows) => (workflows = nextWorkflows)}
              onSelectedIdChange={(nextSelectedId) => (selectedId = nextSelectedId)}
              class="border-l-0"
            />
          </div>
        {/if}
      </div>

      {#if appStore.currentProject?.id}
        <WorkflowScheduledJobsPanel projectId={appStore.currentProject.id} {workflows} />
      {/if}
    </div>
  {/if}
</div>
