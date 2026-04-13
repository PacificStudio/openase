<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { ApiError } from '$lib/api/client'
  import { listStatuses } from '$lib/api/openase'
  import { WorkflowScheduledJobsPanel } from '$lib/features/settings'
  import { mapStatusOptions, type WorkflowStatusOption } from '$lib/features/workflows'
  import { scheduledJobsT } from './i18n'

  let loading = $state(false)
  let error = $state('')
  let statuses = $state<WorkflowStatusOption[]>([])

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      statuses = []
      error = ''
      loading = false
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const statusPayload = await listStatuses(projectId)
        if (cancelled) return

        statuses = mapStatusOptions(statusPayload.statuses)
      } catch (caughtError) {
        if (cancelled) return
        error =
          caughtError instanceof ApiError
            ? caughtError.detail
            : scheduledJobsT('scheduledJobs.failedToLoad')
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

<div class="flex h-full flex-col">
  {#if error}
    <div class="text-destructive p-6 text-sm">{error}</div>
  {:else if !loading && !appStore.currentProject?.id}
    <div class="text-muted-foreground p-6 text-sm">
      {scheduledJobsT('scheduledJobs.projectContextUnavailable')}
    </div>
  {:else if !loading && statuses.length === 0}
    <div class="text-muted-foreground p-6 text-sm">
      {scheduledJobsT('scheduledJobs.noTicketStatuses')}
    </div>
  {:else}
    <WorkflowScheduledJobsPanel
      projectId={appStore.currentProject?.id ?? ''}
      statuses={loading ? [] : statuses}
      {loading}
      title={scheduledJobsT('scheduledJobs.title')}
      description={scheduledJobsT('scheduledJobs.description')}
    />
  {/if}
</div>
