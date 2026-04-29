<script lang="ts">
  import type { ScheduledJob } from '$lib/api/contracts'
  import {
    createScheduledJob,
    deleteScheduledJob,
    listScheduledJobs,
    listProjectRepos,
    triggerScheduledJob,
    updateScheduledJob,
  } from '$lib/api/openase'
  import {
    mapProjectRepoOptions,
    type RepoScopeOption as TicketRepoOption,
  } from '$lib/features/repo-scope-selection'
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
  import {
    scheduledJobToggleMessageKeys,
    showScheduledJobError,
  } from './workflow-scheduled-jobs-panel-helpers'
  import { i18nStore } from '$lib/i18n/store.svelte'

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
  let repoOptions = $state<TicketRepoOption[]>([])
  let loadingJobs = $state(false)
  let loadingRepos = $state(false)
  const loading = $derived(parentLoading || loadingJobs || loadingRepos)
  let saving = $state(false)
  let deleting = $state(false)
  let triggering = $state(false)
  let actionJobId = $state<string | null>(null)
  let editorOpen = $state(false)
  let selectedJobId = $state('')
  let draft = $state<ScheduledJobDraft>(emptyScheduledJobDraft('', []))
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
    const nextRepoOptions = repoOptions
    const validRepoIds = new Set(nextRepoOptions.map((repo) => repo.id))
    const filteredRepoIds = draft.ticketRepoIds.filter((repoId) => validRepoIds.has(repoId))
    const filteredBranchOverrides = Object.fromEntries(
      Object.entries(draft.ticketRepoBranchOverrides).filter(([repoId]) =>
        validRepoIds.has(repoId),
      ),
    )
    const fallbackRepoIds =
      filteredRepoIds.length > 0 || nextRepoOptions.length !== 1
        ? filteredRepoIds
        : [nextRepoOptions[0].id]

    if (
      filteredRepoIds.length !== draft.ticketRepoIds.length ||
      Object.keys(filteredBranchOverrides).length !==
        Object.keys(draft.ticketRepoBranchOverrides).length ||
      fallbackRepoIds.length !== filteredRepoIds.length
    ) {
      draft = {
        ...draft,
        ticketRepoIds: fallbackRepoIds,
        ticketRepoBranchOverrides: filteredBranchOverrides,
      }
    }
  })

  $effect(() => {
    if (!projectId) return
    let cancelled = false
    const loadJobs = async () => {
      loadingJobs = true
      try {
        const payload = await listScheduledJobs(projectId)
        if (cancelled) return
        jobs = payload.scheduled_jobs
      } catch (caughtError) {
        if (cancelled) return
        jobs = []
        showScheduledJobError(caughtError, 'settings.workflowScheduledJobs.errors.load')
      } finally {
        if (!cancelled) loadingJobs = false
      }
    }

    const loadRepos = async () => {
      loadingRepos = true
      try {
        const payload = await listProjectRepos(projectId)
        if (cancelled) return
        repoOptions = mapProjectRepoOptions(payload.repos)
      } catch (caughtError) {
        if (cancelled) return
        repoOptions = []
        showScheduledJobError(
          caughtError,
          'settings.workflowScheduledJobs.errors.loadProjectRepositories',
        )
      } finally {
        if (!cancelled) loadingRepos = false
      }
    }

    void Promise.all([loadJobs(), loadRepos()])
    return () => {
      cancelled = true
    }
  })

  const openNewJob = () => {
    selectedJobId = ''
    draft = emptyScheduledJobDraft(statuses[0]?.id ?? '', repoOptions)
    editorOpen = true
  }
  const openEditJob = (job: ScheduledJob) => {
    selectedJobId = job.id
    draft = scheduledJobDraftFromRecord(job, statuses, repoOptions)
    editorOpen = true
  }
  const handleEditorClose = (open: boolean) => {
    if (!open) {
      selectedJobId = ''
      draft = emptyScheduledJobDraft(statuses[0]?.id ?? '', repoOptions)
    }
  }
  const refreshJobs = async () => (jobs = (await listScheduledJobs(projectId)).scheduled_jobs)
  async function handleSubmit() {
    const parsed = parseScheduledJobDraft(draft, statuses, repoOptions)
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
          repoOptions,
        )
        toastStore.success(i18nStore.t('settings.workflowScheduledJobs.messages.created'))
      }
    } catch (caughtError) {
      showScheduledJobError(caughtError, 'settings.workflowScheduledJobs.errors.save')
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
      showScheduledJobError(caughtError, 'settings.workflowScheduledJobs.errors.delete')
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
      showScheduledJobError(caughtError, 'settings.workflowScheduledJobs.errors.toggle')
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
      showScheduledJobError(caughtError, 'settings.workflowScheduledJobs.errors.trigger')
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
      showScheduledJobError(caughtError, 'settings.workflowScheduledJobs.errors.trigger')
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
      showScheduledJobError(caughtError, 'settings.workflowScheduledJobs.errors.delete')
    } finally {
      actionJobId = null
    }
  }

  const handleDraftFieldChange = (field: keyof ScheduledJobDraft, value: string | boolean) =>
    (draft = { ...draft, [field]: value })

  function handleToggleDraftRepoScope(repoId: string) {
    draft = {
      ...draft,
      ticketRepoIds: draft.ticketRepoIds.includes(repoId)
        ? draft.ticketRepoIds.filter((value) => value !== repoId)
        : [...draft.ticketRepoIds, repoId],
    }
  }

  function handleDraftRepoBranchOverride(repoId: string, value: string) {
    draft = {
      ...draft,
      ticketRepoBranchOverrides: {
        ...draft.ticketRepoBranchOverrides,
        [repoId]: value,
      },
    }
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
        <div data-tour="scheduled-jobs-summary">
          <WorkflowScheduledJobsSummary total={jobs.length} enabled={enabledCount} />
        </div>
      {/if}
      <div data-tour="scheduled-jobs-list">
        <WorkflowScheduledJobList
          {jobs}
          {actionJobId}
          onNewJob={openNewJob}
          onEditJob={openEditJob}
          onToggleEnabled={handleToggleEnabled}
          onTriggerJob={handleTriggerJob}
          onDeleteJob={handleDeleteJob}
        />
      </div>
    {/if}
  </div>
</div>

<WorkflowScheduledJobEditor
  bind:open={editorOpen}
  {projectId}
  {draft}
  {repoOptions}
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
  onToggleRepoScope={handleToggleDraftRepoScope}
  onUpdateRepoBranchOverride={handleDraftRepoBranchOverride}
/>
