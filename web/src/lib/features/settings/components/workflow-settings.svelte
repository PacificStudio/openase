<script lang="ts">
  import type { StatusPayload } from '$lib/api/contracts'
  import { listStatuses } from '$lib/api/openase'
  import { connectEventStream } from '$lib/api/sse'
  import {
    WorkflowRepositoryPrerequisiteCard,
    WorkflowLifecycleSidebar,
    WorkflowList,
    type WorkflowAgentOption,
    type WorkflowRepositoryPrerequisite,
    type WorkflowStatusOption,
    type WorkflowSummary,
  } from '$lib/features/workflows'
  import { loadWorkflowCatalog, loadWorkflowRepositoryPrerequisite } from '$lib/features/workflows'
  import { appStore } from '$lib/stores/app.svelte'
  import { ApiError } from '$lib/api/client'
  import { Separator } from '$ui/separator'
  import StatusStageConcurrency from './status-stage-concurrency.svelte'
  import { startStageRuntimeSync } from './stage-runtime-sync'

  const props = $props<{
    onOpenRepositories?: (() => void) | undefined
  }>()

  let loading = $state(false)
  let error = $state('')
  let prerequisite = $state<WorkflowRepositoryPrerequisite | null>(null)
  let workflows = $state<WorkflowSummary[]>([])
  let agentOptions = $state<WorkflowAgentOption[]>([])
  let statuses = $state<WorkflowStatusOption[]>([])
  let stages = $state<StatusPayload['stages']>([])
  let selectedId = $state('')

  let selectedWorkflow = $derived(workflows.find((workflow) => workflow.id === selectedId) ?? null)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      workflows = []
      agentOptions = []
      statuses = []
      stages = []
      prerequisite = null
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
        const [nextPrerequisite, catalog, statusPayload] = await Promise.all([
          loadWorkflowRepositoryPrerequisite(projectId),
          loadWorkflowCatalog(projectId, orgId),
          listStatuses(projectId),
        ])
        if (cancelled) return
        prerequisite = nextPrerequisite
        workflows = catalog.workflows
        agentOptions = catalog.agentOptions
        statuses = catalog.statuses
        stages = statusPayload.stages
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
      stages = []
      return
    }

    const stopSync = startStageRuntimeSync({
      projectId,
      loadStatuses: listStatuses,
      connectEventStream,
      skipInitialLoad: true,
      applySnapshot: (payload) => {
        stages = payload.stages
      },
      onRefreshError: (caughtError) => {
        console.error('Failed to refresh workflow stage concurrency:', caughtError)
      },
    })

    return stopSync
  })
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Workflow Lifecycle</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Manage workflow agent binding, scheduling policy, activation, and deletion.
    </p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading workflows…</div>
  {:else if prerequisite && prerequisite.kind !== 'ready'}
    <WorkflowRepositoryPrerequisiteCard
      {prerequisite}
      onOpenRepositories={props.onOpenRepositories}
    />
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

      {#if stages.length > 0}
        <StatusStageConcurrency {stages} />
      {/if}
    </div>
  {/if}
</div>
