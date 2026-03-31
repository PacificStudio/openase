<script lang="ts">
  import type { StatusPayload } from '$lib/api/contracts'
  import { listStatuses } from '$lib/api/openase'
  import { connectEventStream } from '$lib/api/sse'
  import {
    WorkflowLifecycleSidebar,
    WorkflowList,
    type WorkflowAgentOption,
    type WorkflowStatusOption,
    type WorkflowSummary,
  } from '$lib/features/workflows'
  import { loadWorkflowCatalog } from '$lib/features/workflows'
  import { appStore } from '$lib/stores/app.svelte'
  import { ApiError } from '$lib/api/client'
  import { Separator } from '$ui/separator'
  import StatusConcurrency from './status-concurrency.svelte'
  import { startStatusRuntimeSync } from './status-runtime-sync'

  let loading = $state(false)
  let error = $state('')
  let workflows = $state<WorkflowSummary[]>([])
  let agentOptions = $state<WorkflowAgentOption[]>([])
  let statuses = $state<WorkflowStatusOption[]>([])
  let statusCapacity = $state<StatusPayload['statuses']>([])
  let selectedId = $state('')

  let selectedWorkflow = $derived(workflows.find((workflow) => workflow.id === selectedId) ?? null)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      workflows = []
      agentOptions = []
      statuses = []
      statusCapacity = []
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
        const [catalog, statusPayload] = await Promise.all([
          loadWorkflowCatalog(projectId, orgId),
          listStatuses(projectId),
        ])
        if (cancelled) return
        workflows = catalog.workflows
        agentOptions = catalog.agentOptions
        statuses = catalog.statuses
        statusCapacity = statusPayload.statuses
        if (
          !selectedId ||
          !catalog.workflows.some((workflow: WorkflowSummary) => workflow.id === selectedId)
        ) {
          selectedId = catalog.workflows[0]?.id ?? ''
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

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      statusCapacity = []
      return
    }

    const stopSync = startStatusRuntimeSync({
      projectId,
      loadSnapshot: listStatuses,
      connectEventStream,
      skipInitialLoad: true,
      applySnapshot: (payload) => {
        statusCapacity = payload.statuses
      },
      onRefreshError: (caughtError) => {
        console.error('Failed to refresh workflow status concurrency:', caughtError)
      },
    })

    return stopSync
  })
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Workflow Lifecycle</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Manage workflow agent binding, published harness versions, scheduling policy, activation, and
      deletion.
    </p>
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

      {#if statusCapacity.some((status) => status.max_active_runs != null)}
        <StatusConcurrency statuses={statusCapacity} />
      {/if}
    </div>
  {/if}
</div>
