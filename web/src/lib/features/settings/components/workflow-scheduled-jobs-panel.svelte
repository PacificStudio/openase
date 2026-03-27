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
  import type { WorkflowSummary } from '$lib/features/workflows'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import WorkflowScheduledJobEditor from './workflow-scheduled-job-editor.svelte'
  import WorkflowScheduledJobList from './workflow-scheduled-job-list.svelte'
  import {
    emptyScheduledJobDraft,
    parseScheduledJobDraft,
    scheduledJobDraftFromRecord,
    type ScheduledJobDraft,
  } from './workflow-scheduled-jobs'

  let {
    projectId,
    workflows,
    showHeader = true,
    title = 'Scheduled Jobs',
    description = 'Manage recurring ticket creation for project workflows.',
  }: {
    projectId: string
    workflows: WorkflowSummary[]
    showHeader?: boolean
    title?: string
    description?: string
  } = $props()

  let jobs = $state<ScheduledJob[]>([])
  let loading = $state(false)
  let saving = $state(false)
  let deleting = $state(false)
  let triggering = $state(false)
  let selectedJobId = $state('')
  let draft = $state<ScheduledJobDraft>(emptyScheduledJobDraft(''))

  const selectedJob = $derived(jobs.find((job) => job.id === selectedJobId) ?? null)
  const workflowOptions = $derived(
    workflows.map((workflow) => ({
      value: workflow.id,
      label: workflow.name,
    })),
  )
  const workflowLabelById = $derived(
    new Map(workflowOptions.map((workflow) => [workflow.value, workflow.label])),
  )

  $effect(() => {
    const defaultWorkflowId = workflowOptions[0]?.value ?? ''

    if (!defaultWorkflowId) {
      if (draft.workflowId) {
        draft = {
          ...draft,
          workflowId: '',
        }
      }
      return
    }

    if (
      !draft.workflowId ||
      !workflowOptions.some((workflow) => workflow.value === draft.workflowId)
    ) {
      draft = {
        ...draft,
        workflowId: defaultWorkflowId,
      }
    }
  })

  $effect(() => {
    let cancelled = false

    const load = async () => {
      loading = true

      try {
        const payload = await listScheduledJobs(projectId)
        if (cancelled) return

        jobs = payload.scheduled_jobs
        if (selectedJobId && payload.scheduled_jobs.some((job) => job.id === selectedJobId)) {
          draft = scheduledJobDraftFromRecord(
            payload.scheduled_jobs.find((job) => job.id === selectedJobId)!,
            workflowOptions[0]?.value ?? '',
          )
        } else {
          selectedJobId = ''
          draft = emptyScheduledJobDraft(workflowOptions[0]?.value ?? '')
        }
      } catch (caughtError) {
        if (cancelled) return
        jobs = []
        toastStore.error(
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load scheduled jobs.',
        )
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

  function selectJob(job: ScheduledJob) {
    selectedJobId = job.id
    draft = scheduledJobDraftFromRecord(job, workflowOptions[0]?.value ?? '')
  }

  function selectNewJob() {
    selectedJobId = ''
    draft = emptyScheduledJobDraft(workflowOptions[0]?.value ?? '')
  }

  async function refreshJobs(nextSelectedId = selectedJobId) {
    const payload = await listScheduledJobs(projectId)
    jobs = payload.scheduled_jobs
    if (nextSelectedId && payload.scheduled_jobs.some((job) => job.id === nextSelectedId)) {
      selectedJobId = nextSelectedId
      draft = scheduledJobDraftFromRecord(
        payload.scheduled_jobs.find((job) => job.id === nextSelectedId)!,
        workflowOptions[0]?.value ?? '',
      )
      return
    }

    selectedJobId = ''
    draft = emptyScheduledJobDraft(workflowOptions[0]?.value ?? '')
  }

  async function handleSubmit() {
    const parsed = parseScheduledJobDraft(draft)
    if (!parsed.ok) {
      toastStore.error(parsed.error)
      return
    }

    saving = true

    try {
      if (selectedJob) {
        await updateScheduledJob(selectedJob.id, parsed.value)
        await refreshJobs(selectedJob.id)
        toastStore.success('Scheduled job updated.')
      } else {
        const payload = await createScheduledJob(projectId, parsed.value)
        await refreshJobs(payload.scheduled_job.id)
        toastStore.success('Scheduled job created.')
      }
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save scheduled job.',
      )
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (!selectedJob) return

    deleting = true

    try {
      await deleteScheduledJob(selectedJob.id)
      await refreshJobs('')
      toastStore.success('Scheduled job deleted.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete scheduled job.',
      )
    } finally {
      deleting = false
    }
  }

  async function handleTrigger() {
    if (!selectedJob) return

    triggering = true

    try {
      await triggerScheduledJob(selectedJob.id)
      await refreshJobs(selectedJob.id)
      toastStore.success('Scheduled job triggered.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to trigger scheduled job.',
      )
    } finally {
      triggering = false
    }
  }

  function handleDraftFieldChange(field: keyof ScheduledJobDraft, value: string | boolean) {
    draft = {
      ...draft,
      [field]: value,
    }
  }
</script>

<div class="space-y-4">
  {#if showHeader}
    <div class="flex items-center justify-between gap-3">
      <div>
        <h3 class="text-foreground text-base font-semibold">{title}</h3>
        <p class="text-muted-foreground mt-1 text-sm">{description}</p>
      </div>
      <Button variant="outline" size="sm" onclick={selectNewJob}>New job</Button>
    </div>
  {/if}

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading scheduled jobs…</div>
  {:else}
    <div class="border-border flex min-h-[30rem] overflow-hidden rounded-lg border">
      <WorkflowScheduledJobList {jobs} {selectedJobId} {workflowLabelById} onSelect={selectJob} />

      <WorkflowScheduledJobEditor
        {draft}
        {selectedJob}
        {workflowOptions}
        {saving}
        {deleting}
        {triggering}
        onFieldChange={handleDraftFieldChange}
        onSubmit={handleSubmit}
        onDelete={handleDelete}
        onTrigger={handleTrigger}
      />
    </div>
  {/if}
</div>
