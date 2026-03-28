<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { ApiError } from '$lib/api/client'
  import { listStatuses, listWorkflows } from '$lib/api/openase'
  import { WorkflowScheduledJobsPanel } from '$lib/features/settings'
  import { mapStatusOptions, mapWorkflowSummary } from '$lib/features/workflows/data'
  import type { WorkflowSummary } from '$lib/features/workflows'
  import { projectPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'

  let loading = $state(false)
  let error = $state('')
  let workflows = $state<WorkflowSummary[]>([])

  const workflowsHref = $derived(
    appStore.currentOrg?.id && appStore.currentProject?.id
      ? projectPath(appStore.currentOrg.id, appStore.currentProject.id, 'workflows')
      : null,
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId || !appStore.currentOrg?.id) {
      workflows = []
      error = ''
      loading = false
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const [workflowPayload, statusPayload] = await Promise.all([
          listWorkflows(projectId),
          listStatuses(projectId),
        ])
        if (cancelled) return

        const statuses = mapStatusOptions(statusPayload.statuses)
        const statusNamesById = new Map(statuses.map((status) => [status.id, status.name]))
        workflows = workflowPayload.workflows.map((workflow) =>
          mapWorkflowSummary(workflow, statusNamesById),
        )
      } catch (caughtError) {
        if (cancelled) return
        error =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load scheduled jobs.'
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

<div class="space-y-6 p-6">
  {#if loading}
    <div class="text-muted-foreground text-sm">Loading scheduled jobs…</div>
  {:else if error}
    <div class="text-destructive text-sm">{error}</div>
  {:else if !appStore.currentProject?.id}
    <div class="text-muted-foreground text-sm">Project context is unavailable.</div>
  {:else if workflows.length === 0}
    <div class="border-border bg-muted/20 space-y-3 rounded-lg border p-4">
      <div class="space-y-1">
        <h2 class="text-foreground text-base font-semibold">No workflows available</h2>
        <p class="text-muted-foreground text-sm">
          Create a workflow before scheduling recurring ticket creation.
        </p>
      </div>
      <Button variant="outline" href={workflowsHref ?? undefined} disabled={!workflowsHref}>
        Open Workflows
      </Button>
    </div>
  {:else}
    <WorkflowScheduledJobsPanel
      projectId={appStore.currentProject.id}
      {workflows}
      title="Scheduled Jobs"
      description="Manage recurring ticket creation for project workflows from a dedicated project page."
    />
  {/if}
</div>
