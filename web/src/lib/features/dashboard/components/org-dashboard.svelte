<script lang="ts">
  import { cn, formatBytes, formatCount, formatCurrency } from '$lib/utils'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import {
    createProjectUpdateComment,
    createProjectUpdateThread,
    deleteProjectUpdateComment,
    deleteProjectUpdateThread,
    getHRAdvisor,
    getSystemDashboard,
    listActivity,
    listAgents,
    listProjectUpdates,
    listTickets,
    updateProject,
    updateProjectUpdateComment,
    updateProjectUpdateThread,
  } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import {
    isProjectUpdateEvent,
    subscribeProjectEvents,
    type ProjectEventEnvelope,
  } from '$lib/features/project-events'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Skeleton } from '$ui/skeleton'
  import { Textarea } from '$ui/textarea'
  import * as Select from '$ui/select'
  import ActivityFeedPanel from './activity-feed-panel.svelte'
  import HRAdvisorPanel from './hr-advisor-panel.svelte'
  import {
    markProjectOnboardingCompleted,
    OnboardingPanel,
    readProjectOnboardingCompletion,
  } from '$lib/features/onboarding'
  import {
    parseProjectUpdateThreads,
    ProjectUpdateComposer,
    ProjectUpdateThreadCard,
    type ProjectUpdateStatus,
    type ProjectUpdateThread,
  } from '$lib/features/project-updates'
  import { Bot, Coins, MessageSquare, Pencil, Ticket, X, Check } from '@lucide/svelte'
  import {
    buildActivityItems,
    buildDashboardStats,
    buildExceptionItems,
    shouldShowProjectOnboarding,
  } from '../model'
  import { loadOrganizationDashboardSummary } from '../organization-summary'
  import type { DashboardStats, ProjectStatus, HRAdvisorSnapshot, MemorySnapshot } from '../types'

  const dashboardPollIntervalMs = 1000

  const projectStatusOptions: ProjectStatus[] = [
    'Backlog',
    'Planned',
    'In Progress',
    'Completed',
    'Canceled',
    'Archived',
  ]

  const statusClassName: Record<ProjectStatus, string> = {
    Backlog:
      'border-slate-200 bg-slate-100 text-slate-700 hover:bg-slate-200 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200',
    Planned:
      'border-amber-200 bg-amber-50 text-amber-800 hover:bg-amber-100 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-200',
    'In Progress':
      'border-emerald-200 bg-emerald-50 text-emerald-800 hover:bg-emerald-100 dark:border-emerald-900/60 dark:bg-emerald-950/40 dark:text-emerald-200',
    Completed:
      'border-sky-200 bg-sky-50 text-sky-800 hover:bg-sky-100 dark:border-sky-900/60 dark:bg-sky-950/40 dark:text-sky-200',
    Canceled:
      'border-rose-200 bg-rose-50 text-rose-800 hover:bg-rose-100 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200',
    Archived:
      'border-border bg-background text-muted-foreground hover:bg-muted dark:hover:bg-muted/60',
  }

  let loading = $state(false)
  let error = $state('')
  let stats = $state<DashboardStats>({
    runningAgents: 0,
    activeTickets: 0,
    pendingApprovals: 0,
    ticketSpendToday: 0,
    ticketSpendTotal: 0,
    ticketsCreatedToday: 0,
    ticketsCompletedToday: 0,
    ticketInputTokens: 0,
    ticketOutputTokens: 0,
    agentLifetimeTokens: 0,
    avgCycleMinutes: 0,
    prMergeRate: 0,
  })
  let exceptions = $state<ReturnType<typeof buildExceptionItems>>([])
  let activities = $state<ReturnType<typeof buildActivityItems>>([])
  let hrAdvisor = $state<HRAdvisorSnapshot | null>(null)
  let memory = $state<MemorySnapshot | null>(null)
  let savingStatus = $state(false)
  let editingInfo = $state(false)
  let editName = $state('')
  let editDescription = $state('')
  let savingInfo = $state(false)
  let onboardingDismissed = $state(false)
  const totalTicketTokens = $derived(stats.ticketInputTokens + stats.ticketOutputTokens)

  const showOnboarding = $derived(
    shouldShowProjectOnboarding({
      dismissed: onboardingDismissed,
      loading,
      stats,
      projectId: appStore.currentProject?.id,
      orgId: appStore.currentOrg?.id,
    }),
  )
  const currentStatus = $derived((appStore.currentProject?.status ?? 'Planned') as ProjectStatus)
  const projectName = $derived(appStore.currentProject?.name ?? 'Untitled Project')
  const projectDescription = $derived(appStore.currentProject?.description ?? '')

  // --- Project Updates state ---
  let updateThreads = $state<ProjectUpdateThread[]>([])
  let updatesLoading = $state(false)
  let updatesError = $state('')
  let updatesNotice = $state('')
  let updatesInitialLoaded = $state(false)
  let updatesRequestVersion = 0
  let creatingThread = $state(false)

  $effect(() => {
    const projectId = appStore.currentProject?.id ?? ''
    onboardingDismissed = projectId ? readProjectOnboardingCompletion(projectId) : false
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      updateThreads = []
      updatesInitialLoaded = false
      return
    }

    updatesInitialLoaded = false
    void loadProjectUpdates(projectId, { showLoading: true })

    return subscribeProjectEvents(projectId, (event) => {
      if (isProjectUpdateFrame(event)) {
        void loadProjectUpdates(projectId, { preserveMessages: true })
      }
    })
  })

  async function loadProjectUpdates(
    projectId: string,
    options: { showLoading?: boolean; preserveMessages?: boolean } = {},
  ) {
    const version = ++updatesRequestVersion
    if (options.showLoading) updatesLoading = true
    updatesError = ''
    if (!options.preserveMessages) updatesNotice = ''

    try {
      const payload = await listProjectUpdates(projectId)
      if (version !== updatesRequestVersion) return
      updateThreads = parseProjectUpdateThreads(payload.threads)
      updatesInitialLoaded = true
    } catch (caughtError) {
      if (version !== updatesRequestVersion) return
      updatesError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load updates.'
    } finally {
      if (version === updatesRequestVersion) updatesLoading = false
    }
  }

  async function handleCreateThread(draft: {
    status: ProjectUpdateStatus
    title: string
    body: string
  }) {
    const projectId = appStore.currentProject?.id
    if (!projectId || creatingThread) return false
    creatingThread = true
    updatesError = ''
    updatesNotice = ''
    try {
      await createProjectUpdateThread(projectId, draft)
      updatesNotice = 'Update posted.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      updatesError = caughtError instanceof ApiError ? caughtError.detail : 'Failed to post update.'
      return false
    } finally {
      creatingThread = false
    }
  }

  async function handleSaveThread(
    threadId: string,
    draft: { status: ProjectUpdateStatus; title: string; body: string },
  ) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return false
    updatesError = ''
    updatesNotice = ''
    try {
      await updateProjectUpdateThread(projectId, threadId, draft)
      updatesNotice = 'Update edited.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      updatesError = caughtError instanceof ApiError ? caughtError.detail : 'Failed to edit update.'
      return false
    }
  }

  async function handleDeleteThread(threadId: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return false
    updatesError = ''
    updatesNotice = ''
    try {
      await deleteProjectUpdateThread(projectId, threadId)
      updatesNotice = 'Update deleted.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      updatesError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete update.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return false
    }
  }

  async function handleCreateComment(threadId: string, body: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return false
    updatesError = ''
    updatesNotice = ''
    try {
      await createProjectUpdateComment(projectId, threadId, { body })
      updatesNotice = 'Comment added.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      updatesError = caughtError instanceof ApiError ? caughtError.detail : 'Failed to add comment.'
      return false
    }
  }

  async function handleSaveComment(threadId: string, commentId: string, body: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId || !body) return false
    updatesError = ''
    updatesNotice = ''
    try {
      await updateProjectUpdateComment(projectId, threadId, commentId, { body })
      updatesNotice = 'Comment edited.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      updatesError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to edit comment.'
      return false
    }
  }

  async function handleDeleteComment(threadId: string, commentId: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return false
    updatesError = ''
    updatesNotice = ''
    try {
      await deleteProjectUpdateComment(projectId, threadId, commentId)
      updatesNotice = 'Comment deleted.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return true
    } catch (caughtError) {
      updatesError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete comment.'
      await loadProjectUpdates(projectId, { preserveMessages: true })
      return false
    }
  }

  function isProjectUpdateFrame(event: ProjectEventEnvelope) {
    return isProjectUpdateEvent(event)
  }

  function startEditInfo() {
    editName = projectName
    editDescription = projectDescription
    editingInfo = true
  }

  function cancelEditInfo() {
    editingInfo = false
  }

  async function saveInfo() {
    const projectId = appStore.currentProject?.id
    if (!projectId || savingInfo) return

    const name = editName.trim()
    if (!name) {
      toastStore.error('Project name is required.')
      return
    }

    savingInfo = true
    try {
      const payload = await updateProject(projectId, {
        name,
        description: editDescription.trim() || null,
      })
      appStore.currentProject = payload.project
      editingInfo = false
      toastStore.success('Project info updated.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update project info.',
      )
    } finally {
      savingInfo = false
    }
  }

  async function handleProjectStatusChange(status: ProjectStatus) {
    const projectId = appStore.currentProject?.id
    if (!projectId || savingStatus) return

    savingStatus = true

    try {
      const payload = await updateProject(projectId, { status })
      appStore.currentProject = payload.project
      toastStore.success('Project status updated.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update project status.',
      )
    } finally {
      savingStatus = false
    }
  }

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId) {
      activities = []
      exceptions = []
      hrAdvisor = null
      memory = null
      return
    }

    let cancelled = false
    let hasLoaded = false
    let inFlight = false

    const load = async (showLoading: boolean) => {
      if (inFlight) return

      inFlight = true
      if (showLoading) {
        loading = true
      }

      try {
        const [
          agentPayload,
          ticketPayload,
          activityPayload,
          systemPayload,
          hrAdvisorPayload,
          organizationSummary,
        ] = await Promise.all([
          listAgents(projectId),
          listTickets(projectId),
          listActivity(projectId, { limit: 24 }),
          getSystemDashboard(),
          getHRAdvisor(projectId).catch(() => null),
          orgId ? loadOrganizationDashboardSummary(orgId).catch(() => null) : Promise.resolve(null),
        ])

        if (cancelled) return

        stats = buildDashboardStats(agentPayload.agents, ticketPayload.tickets, {
          ticketSpendToday: organizationSummary?.projectMetrics[projectId]?.todayCost ?? 0,
        })
        memory = systemPayload.memory
        hrAdvisor = hrAdvisorPayload
          ? {
              summary: hrAdvisorPayload.summary,
              staffing: hrAdvisorPayload.staffing,
              recommendations: hrAdvisorPayload.recommendations,
            }
          : null

        activities = buildActivityItems(activityPayload.events)
        exceptions = buildExceptionItems(activityPayload.events)

        error = ''
        hasLoaded = true
      } catch (caughtError) {
        if (cancelled) return
        if (!hasLoaded) {
          error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load dashboard.'
        }
      } finally {
        inFlight = false
        if (showLoading && !cancelled) {
          loading = false
        }
      }
    }

    void load(true)

    const interval = window.setInterval(() => {
      void load(false)
    }, dashboardPollIntervalMs)

    return () => {
      cancelled = true
      window.clearInterval(interval)
    }
  })
</script>

<div class="flex min-h-0 flex-col">
  <!-- Project header -->
  <div class="border-border border-b px-6 py-4">
    <div class="flex items-start justify-between gap-3">
      {#if editingInfo}
        <div class="min-w-0 flex-1 space-y-2">
          <Input
            value={editName}
            placeholder="Project name"
            class="text-lg font-semibold"
            oninput={(e) => (editName = (e.currentTarget as HTMLInputElement).value)}
          />
          <Textarea
            value={editDescription}
            placeholder="Project description (optional)"
            rows={2}
            class="text-sm"
            oninput={(e) => (editDescription = (e.currentTarget as HTMLTextAreaElement).value)}
          />
          <div class="flex items-center gap-2">
            <Button size="sm" disabled={savingInfo} onclick={saveInfo}>
              <Check class="mr-1.5 size-3.5" />
              {savingInfo ? 'Saving\u2026' : 'Save'}
            </Button>
            <Button variant="ghost" size="sm" disabled={savingInfo} onclick={cancelEditInfo}>
              <X class="mr-1.5 size-3.5" />
              Cancel
            </Button>
          </div>
        </div>
      {:else}
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <h1 class="text-foreground truncate text-lg font-semibold">{projectName}</h1>
            <button
              type="button"
              class="text-muted-foreground hover:text-foreground shrink-0 transition-colors"
              title="Edit project info"
              onclick={startEditInfo}
            >
              <Pencil class="size-3.5" />
            </button>
          </div>
          {#if projectDescription}
            <p class="text-muted-foreground mt-0.5 text-sm">{projectDescription}</p>
          {:else}
            <p class="text-muted-foreground/50 mt-0.5 text-sm">No description</p>
          {/if}
        </div>
      {/if}

      <div class="flex shrink-0 items-center gap-2">
        <Select.Root
          type="single"
          value={currentStatus}
          onValueChange={(value) => {
            if (value && value !== currentStatus)
              void handleProjectStatusChange(value as ProjectStatus)
          }}
        >
          <Select.Trigger
            class={cn(
              'h-auto min-h-5 w-auto rounded-full border px-2.5 py-1 text-xs font-medium shadow-none',
              statusClassName[currentStatus],
            )}
            disabled={savingStatus}
          >
            {currentStatus}
          </Select.Trigger>
          <Select.Content>
            {#each projectStatusOptions as status (status)}
              <Select.Item value={status}>{status}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>
    </div>
  </div>

  <!-- Content -->
  <div class="min-h-0 flex-1 overflow-y-auto px-4 py-4 pb-8 sm:px-6">
    {#if showOnboarding && appStore.currentProject && appStore.currentOrg}
      <OnboardingPanel
        projectId={appStore.currentProject.id}
        orgId={appStore.currentOrg.id}
        {projectName}
        projectStatus={currentStatus}
        onOnboardingComplete={() => {
          markProjectOnboardingCompleted(appStore.currentProject!.id)
          onboardingDismissed = true
        }}
      />
    {:else}
      <div class="space-y-3">
        {#if error && !loading}
          <div
            class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-3 py-2 text-sm"
          >
            {error}
          </div>
        {/if}

        <!-- Compact stats strip: agents, tickets, usage, memory, exceptions -->
        <div
          class="border-border bg-card flex flex-wrap items-center gap-x-5 gap-y-1.5 rounded-md border px-3 py-2"
        >
          {#if loading}
            {#each { length: 6 } as _}
              <div class="flex items-center gap-1.5">
                <Skeleton class="h-3 w-12" />
                <Skeleton class="h-3.5 w-8" />
              </div>
            {/each}
          {:else}
            <div class="flex items-center gap-1">
              <Bot class="text-muted-foreground size-3" />
              <span class="text-muted-foreground text-[11px]">Agents</span>
              <span class="text-foreground text-xs font-semibold">{stats.runningAgents}</span>
            </div>
            <div class="flex items-center gap-1">
              <Ticket class="text-muted-foreground size-3" />
              <span class="text-muted-foreground text-[11px]">Tickets</span>
              <span class="text-foreground text-xs font-semibold">{stats.activeTickets}</span>
            </div>
            <div class="flex items-center gap-1">
              <Coins class="text-muted-foreground size-3" />
              <span class="text-muted-foreground text-[11px]">Spend</span>
              <span class="text-foreground text-xs font-semibold"
                >{formatCurrency(stats.ticketSpendToday)}</span
              >
            </div>
            <div class="flex items-center gap-1">
              <span class="text-muted-foreground text-[11px]">Tokens</span>
              <span class="text-foreground text-xs font-semibold"
                >{formatCount(totalTicketTokens)}</span
              >
            </div>
            <div class="flex items-center gap-1">
              <span class="text-muted-foreground text-[11px]">Heap</span>
              <span class="text-foreground text-xs font-semibold"
                >{memory ? formatBytes(memory.heap_inuse_bytes) : '—'}</span
              >
            </div>
            {#if exceptions.length > 0}
              <div class="flex items-center gap-1">
                <span
                  class="bg-destructive/10 text-destructive inline-flex size-4 items-center justify-center rounded-full text-[9px] font-semibold"
                  >{exceptions.length}</span
                >
                <span class="text-destructive text-[11px]">
                  {exceptions.length === 1 ? 'exception' : 'exceptions'}
                </span>
              </div>
            {/if}
          {/if}
        </div>

        {#if loading}
          <!-- Skeleton for two-column feeds -->
          <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
            {#each { length: 2 } as _}
              <div class="border-border bg-card rounded-md border">
                <div class="border-border flex items-center justify-between border-b px-3 py-2">
                  <Skeleton class="h-3.5 w-20" />
                  <Skeleton class="h-3 w-10" />
                </div>
                <div class="space-y-2.5 p-3">
                  {#each { length: 4 } as _}
                    <div class="flex items-start gap-2">
                      <Skeleton class="mt-0.5 size-3.5 shrink-0 rounded-full" />
                      <div class="flex-1 space-y-1">
                        <Skeleton class="h-3 w-3/4" />
                        <Skeleton class="h-2.5 w-1/3" />
                      </div>
                    </div>
                  {/each}
                </div>
              </div>
            {/each}
          </div>
        {:else if !error}
          <!-- Two-column feeds: Activity + Updates -->
          <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <!-- Left: Activity feed -->
            <div class="flex min-h-0 flex-col">
              <ActivityFeedPanel {activities} />

              {#if hrAdvisor && appStore.currentProject}
                <div class="mt-3">
                  {#key appStore.currentProject.id}
                    <HRAdvisorPanel projectId={appStore.currentProject.id} advisor={hrAdvisor} />
                  {/key}
                </div>
              {/if}
            </div>

            <!-- Right: Project Updates feed -->
            <div class="flex min-h-0 flex-col">
              <div class="mb-2 flex items-center gap-1.5">
                <MessageSquare class="text-muted-foreground size-3.5" />
                <span class="text-foreground text-xs font-medium">Updates</span>
                {#if updateThreads.length > 0}
                  <span class="text-muted-foreground text-[11px]">
                    {updateThreads.length}
                  </span>
                {/if}
              </div>

              <ProjectUpdateComposer creating={creatingThread} onSubmit={handleCreateThread} />

              {#if updatesError}
                <div
                  class="border-destructive/40 bg-destructive/10 text-destructive mt-2 rounded-md border px-3 py-2 text-xs"
                >
                  {updatesError}
                </div>
              {/if}

              {#if updatesNotice}
                <div
                  class="mt-2 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs text-emerald-700"
                >
                  {updatesNotice}
                </div>
              {/if}

              {#if updatesLoading && !updatesInitialLoaded}
                <div class="mt-2 space-y-2">
                  {#each { length: 3 } as _}
                    <div class="flex items-center gap-2">
                      <Skeleton class="size-3.5 shrink-0 rounded-full" />
                      <Skeleton class="h-3.5 w-3/4" />
                    </div>
                  {/each}
                </div>
              {:else if updateThreads.length === 0}
                <div
                  class="text-muted-foreground mt-2 flex flex-col items-center rounded-xl border border-dashed py-8 text-center text-xs"
                >
                  <MessageSquare class="mb-2 size-4 opacity-40" />
                  No updates yet
                </div>
              {:else}
                <div class="mt-2 space-y-1.5">
                  {#each updateThreads as thread (thread.id)}
                    <ProjectUpdateThreadCard
                      {thread}
                      onUpdateThread={handleSaveThread}
                      onDeleteThread={handleDeleteThread}
                      onCreateComment={handleCreateComment}
                      onUpdateComment={handleSaveComment}
                      onDeleteComment={handleDeleteComment}
                    />
                  {/each}
                </div>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>
