<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { ScheduledJob } from '$lib/api/contracts'
  import {
    createScheduledJob,
    deleteScheduledJob,
    listScheduledJobs,
    triggerScheduledJob,
    updateScheduledJob,
  } from '$lib/api/openase'
  import type { WorkflowStatusOption } from '$lib/features/workflows'
  import { toastStore } from '$lib/stores/toast.svelte'
  import WorkflowScheduledJobEditor from './workflow-scheduled-job-editor.svelte'
  import WorkflowScheduledJobsHeader from './workflow-scheduled-jobs-header.svelte'
  import WorkflowScheduledJobsLoading from './workflow-scheduled-jobs-loading.svelte'
  import WorkflowScheduledJobList from './workflow-scheduled-job-list.svelte'
  import WorkflowScheduledJobsSummary from './workflow-scheduled-jobs-summary.svelte'
  import {
    emptyScheduledJobDraft,
    parseScheduledJobDraft,
    scheduledJobDraftFromRecord,
    type ScheduledJobDraft,
  } from './workflow-scheduled-jobs'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import type { TranslationKey } from '$lib/i18n'

  let {
    projectId,
    statuses,
    loading: parentLoading = false,
    showHeader = true,
    title = '',
    description = '',
  }: {
    projectId: string
    statuses: WorkflowStatusOption[]
    loading?: boolean
    showHeader?: boolean
    title?: string
    description?: string
  } = $props()

  let jobs = $state<ScheduledJob[]>([])
  let loadingJobs = $state(false)
  const loading = $derived(parentLoading || loadingJobs)
  let saving = $state(false)
  let deleting = $state(false)
  let triggering = $state(false)
  let actionJobId = $state<string | null>(null)
  let editorOpen = $state(false)
  let selectedJobId = $state('')
  let draft = $state<ScheduledJobDraft>(emptyScheduledJobDraft(''))

  const selectedJob = $derived(jobs.find((job) => job.id === selectedJobId) ?? null)
  const enabledCount = $derived(jobs.filter((j) => j.is_enabled).length)

  $effect(() => {
    const defaultStatusId = statuses[0]?.id ?? ''

    if (!defaultStatusId) {
      if (draft.ticketStatusId) {
        draft = { ...draft, ticketStatusId: '' }
      }
      return
    }

    if (!draft.ticketStatusId || !statuses.some((status) => status.id === draft.ticketStatusId)) {
      draft = { ...draft, ticketStatusId: defaultStatusId }
    }
  })

  $effect(() => {
    if (!projectId) return

    let cancelled = false

    const load = async () => {
      loadingJobs = true

      try {
        const payload = await listScheduledJobs(projectId)
        if (cancelled) return
        jobs = payload.scheduled_jobs
      } catch (caughtError) {
        if (cancelled) return
        jobs = []
        showApiError(caughtError, 'settings.workflowScheduledJobs.errors.load')
      } finally {
        if (!cancelled) loadingJobs = false
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  function openNewJob() {
    selectedJobId = ''
    draft = emptyScheduledJobDraft(statuses[0]?.id ?? '')
    editorOpen = true
  }

  function openEditJob(job: ScheduledJob) {
    selectedJobId = job.id
    draft = scheduledJobDraftFromRecord(job, statuses)
    editorOpen = true
  }

  function handleEditorClose(open: boolean) {
    if (!open) {
      selectedJobId = ''
      draft = emptyScheduledJobDraft(statuses[0]?.id ?? '')
    }
  }

  async function refreshJobs() {
    const payload = await listScheduledJobs(projectId)
    jobs = payload.scheduled_jobs
  }

  function showApiError(caughtError: unknown, fallbackKey: TranslationKey) {
    toastStore.error(
      caughtError instanceof ApiError ? caughtError.detail : i18nStore.t(fallbackKey),
    )
  }

  const scheduledJobToggleMessageKeys: Record<'enabled' | 'disabled', TranslationKey> = {
    enabled: 'settings.workflowScheduledJobs.messages.enabled',
    disabled: 'settings.workflowScheduledJobs.messages.disabled',
  }

  async function handleSubmit() {
    const parsed = parseScheduledJobDraft(draft, statuses)
    if (!parsed.ok) {
      toastStore.error(parsed.error)
      return
    }

    saving = true

    try {
      if (selectedJob) {
        await updateScheduledJob(selectedJob.id, parsed.value)
        await refreshJobs()
        toastStore.success(i18nStore.t('settings.workflowScheduledJobs.messages.updated'))
      } else {
        const payload = await createScheduledJob(projectId, parsed.value)
        await refreshJobs()
        selectedJobId = payload.scheduled_job.id
        draft = scheduledJobDraftFromRecord(
          jobs.find((j) => j.id === payload.scheduled_job.id)!,
          statuses,
        )
        toastStore.success(i18nStore.t('settings.workflowScheduledJobs.messages.created'))
      }
    } catch (caughtError) {
      showApiError(caughtError, 'settings.workflowScheduledJobs.errors.save')
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (!selectedJob) return

    deleting = true

    try {
      await deleteScheduledJob(selectedJob.id)
      editorOpen = false
      selectedJobId = ''
      await refreshJobs()
      toastStore.success(i18nStore.t('settings.workflowScheduledJobs.messages.deleted'))
    } catch (caughtError) {
      showApiError(caughtError, 'settings.workflowScheduledJobs.errors.delete')
    } finally {
      deleting = false
    }
  }

  async function handleToggleEnabled(job: ScheduledJob) {
    actionJobId = job.id

    try {
      await updateScheduledJob(job.id, { is_enabled: !job.is_enabled })
      await refreshJobs()
      toastStore.success(
        i18nStore.t(
          job.is_enabled
            ? scheduledJobToggleMessageKeys.disabled
            : scheduledJobToggleMessageKeys.enabled,
        ),
      )
    } catch (caughtError) {
      showApiError(caughtError, 'settings.workflowScheduledJobs.errors.toggle')
    } finally {
      actionJobId = null
    }
  }

  async function handleTriggerJob(job: ScheduledJob) {
    actionJobId = job.id

    try {
      await triggerScheduledJob(job.id)
      await refreshJobs()
      toastStore.success(i18nStore.t('settings.workflowScheduledJobs.messages.triggered'))
    } catch (caughtError) {
      showApiError(caughtError, 'settings.workflowScheduledJobs.errors.trigger')
    } finally {
      actionJobId = null
    }
  }

  async function handleTriggerFromEditor() {
    if (!selectedJob) return

    triggering = true

    try {
      await triggerScheduledJob(selectedJob.id)
      await refreshJobs()
      toastStore.success(i18nStore.t('settings.workflowScheduledJobs.messages.triggered'))
    } catch (caughtError) {
      showApiError(caughtError, 'settings.workflowScheduledJobs.errors.trigger')
    } finally {
      triggering = false
    }
  }

  async function handleDeleteJob(job: ScheduledJob) {
    actionJobId = job.id

    try {
      await deleteScheduledJob(job.id)
      if (selectedJobId === job.id) {
        editorOpen = false
        selectedJobId = ''
      }
      await refreshJobs()
      toastStore.success(i18nStore.t('settings.workflowScheduledJobs.messages.deleted'))
    } catch (caughtError) {
      showApiError(caughtError, 'settings.workflowScheduledJobs.errors.delete')
    } finally {
      actionJobId = null
    }
  }

  function handleDraftFieldChange(field: keyof ScheduledJobDraft, value: string | boolean) {
    draft = { ...draft, [field]: value }
  }
</script>

<div class="flex h-full min-h-0 flex-col">
  {#if showHeader}
    <WorkflowScheduledJobsHeader
      title={title || i18nStore.t('settings.workflowScheduledJobs.heading')}
      description={description || i18nStore.t('settings.workflowScheduledJobs.description')}
      onCreate={openNewJob}
    />
  {/if}

  <div class="flex-1 overflow-y-auto p-4">
    {#if loading}
      <WorkflowScheduledJobsLoading />
    {:else}
      {#if jobs.length > 0}
        <WorkflowScheduledJobsSummary total={jobs.length} enabled={enabledCount} />
      {/if}

      <WorkflowScheduledJobList
        {jobs}
        {actionJobId}
        onNewJob={openNewJob}
        onEditJob={openEditJob}
        onToggleEnabled={handleToggleEnabled}
        onTriggerJob={handleTriggerJob}
        onDeleteJob={handleDeleteJob}
      />
    {/if}
  </div>
</div>

<WorkflowScheduledJobEditor
  bind:open={editorOpen}
  {projectId}
  {draft}
  {selectedJob}
  statusOptions={statuses}
  {saving}
  {deleting}
  {triggering}
  onFieldChange={handleDraftFieldChange}
  onSubmit={handleSubmit}
  onDelete={handleDelete}
  onTrigger={handleTriggerFromEditor}
  onOpenChange={handleEditorClose}
/>
