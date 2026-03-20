<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import {
    loadWorkflowCatalog,
    WorkflowLifecycleSidebar,
    WorkflowList,
    type WorkflowStatusOption,
    type WorkflowSummary,
  } from '$lib/features/workflows'
  import { appStore } from '$lib/stores/app.svelte'
  import { Separator } from '$ui/separator'

  let loading = $state(false)
  let error = $state('')
  let workflows = $state<WorkflowSummary[]>([])
  let statuses = $state<WorkflowStatusOption[]>([])
  let selectedId = $state('')

  let selectedWorkflow = $derived(workflows.find((workflow) => workflow.id === selectedId) ?? null)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      workflows = []
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
        const payload = await loadWorkflowCatalog(projectId)
        if (cancelled) return

        workflows = payload.workflows
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
    <h2 class="text-foreground text-base font-semibold">Workflow Lifecycle</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Rename workflows, adjust scheduling policy, toggle active state, and delete obsolete workflows
      from the same backend boundary used on the main workflows page.
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
            onWorkflowsChange={(nextWorkflows) => (workflows = nextWorkflows)}
            onSelectedIdChange={(nextSelectedId) => (selectedId = nextSelectedId)}
            class="border-l-0"
          />
        </div>
      {/if}
    </div>
  {/if}
</div>
