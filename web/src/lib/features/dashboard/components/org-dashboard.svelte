<script lang="ts">
  import { cn, formatBytes, formatCount } from '$lib/utils'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import {
    getHRAdvisor,
    getSystemDashboard,
    listActivity,
    listAgents,
    listTickets,
    updateProject,
  } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Skeleton } from '$ui/skeleton'
  import { Textarea } from '$ui/textarea'
  import * as Select from '$ui/select'
  import StatCard from './stat-card.svelte'
  import ExceptionPanel from './exception-panel.svelte'
  import ActivityFeedPanel from './activity-feed-panel.svelte'
  import CostSnapshotPanel from './cost-snapshot-panel.svelte'
  import HRAdvisorPanel from './hr-advisor-panel.svelte'
  import MemorySnapshotPanel from './memory-snapshot-panel.svelte'
  import { OnboardingPanel } from '$lib/features/onboarding'
  import { Bot, Coins, Pencil, Ticket, X, Check } from '@lucide/svelte'
  import {
    buildActivityItems,
    buildDashboardStats,
    buildExceptionItems,
    findTopCostTicket,
    findTopTokenAgent,
  } from '../model'
  import { loadOrganizationDashboardSummary } from '../organization-summary'
  import type {
    DashboardStats,
    ProjectStatus,
    DashboardUsageLeader,
    HRAdvisorSnapshot,
    MemorySnapshot,
  } from '../types'

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
  let topCostTicket = $state<DashboardUsageLeader | null>(null)
  let topTokenAgent = $state<DashboardUsageLeader | null>(null)
  let savingStatus = $state(false)
  let editingInfo = $state(false)
  let editName = $state('')
  let editDescription = $state('')
  let savingInfo = $state(false)
  let onboardingDismissed = $state(false)
  const totalTicketTokens = $derived(stats.ticketInputTokens + stats.ticketOutputTokens)

  // Show onboarding when project has no tickets and no agents (empty project)
  const showOnboarding = $derived(
    !onboardingDismissed &&
      !loading &&
      stats.activeTickets === 0 &&
      stats.runningAgents === 0 &&
      activities.length === 0 &&
      Boolean(appStore.currentProject?.id) &&
      Boolean(appStore.currentOrg?.id),
  )
  const currentStatus = $derived((appStore.currentProject?.status ?? 'Planned') as ProjectStatus)
  const projectName = $derived(appStore.currentProject?.name ?? 'Untitled Project')
  const projectDescription = $derived(appStore.currentProject?.description ?? '')

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
      topCostTicket = null
      topTokenAgent = null
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
        topCostTicket = findTopCostTicket(ticketPayload.tickets)
        topTokenAgent = findTopTokenAgent(agentPayload.agents)
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
          onboardingDismissed = true
        }}
      />
    {:else}
      <div class="space-y-6">
        {#if error && !loading}
          <div
            class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
          >
            {error}
          </div>
        {/if}

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard label="Running Agents" value={stats.runningAgents} icon={Bot} {loading} />
          <StatCard label="Active Tickets" value={stats.activeTickets} icon={Ticket} {loading} />
          <StatCard
            label="Ticket Tokens"
            value={formatCount(totalTicketTokens)}
            icon={Coins}
            {loading}
          />
          <StatCard
            label="Heap In Use"
            value={memory ? formatBytes(memory.heap_inuse_bytes) : '—'}
            {loading}
          />
        </div>

        {#if loading}
          <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <div class="border-border bg-card rounded-md border">
              <div class="border-border flex items-center justify-between border-b px-4 py-3">
                <Skeleton class="h-4 w-28" />
                <Skeleton class="size-4" />
              </div>
              <div class="space-y-3 p-4">
                <div class="grid grid-cols-2 gap-3">
                  {#each { length: 4 } as _}
                    <div class="bg-muted/40 rounded-md px-3 py-2">
                      <Skeleton class="h-3 w-20" />
                      <Skeleton class="mt-2 h-5 w-14" />
                    </div>
                  {/each}
                </div>
                <Skeleton class="h-px w-full" />
                <div class="flex justify-between">
                  <Skeleton class="h-5 w-24" />
                  <Skeleton class="h-5 w-24" />
                </div>
              </div>
            </div>
            <div class="border-border bg-card rounded-md border">
              <div class="border-border flex items-center justify-between border-b px-4 py-3">
                <Skeleton class="h-4 w-20" />
                <Skeleton class="size-4" />
              </div>
              <div class="space-y-3 p-4">
                {#each { length: 3 } as _}
                  <div class="flex items-start gap-3">
                    <Skeleton class="mt-0.5 size-4 shrink-0 rounded-full" />
                    <div class="flex-1 space-y-1">
                      <Skeleton class="h-3.5 w-3/4" />
                      <Skeleton class="h-3 w-1/3" />
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          </div>

          <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
            <div class="border-border bg-card rounded-md border lg:col-span-2">
              <div class="border-border flex items-center justify-between border-b px-4 py-3">
                <Skeleton class="h-4 w-24" />
                <Skeleton class="size-4" />
              </div>
              <div class="space-y-3 p-4">
                {#each { length: 4 } as _}
                  <div class="flex items-center gap-3">
                    <Skeleton class="size-6 shrink-0 rounded-full" />
                    <div class="flex-1 space-y-1">
                      <Skeleton class="h-3.5 w-2/3" />
                      <Skeleton class="h-3 w-1/4" />
                    </div>
                    <Skeleton class="h-3 w-16" />
                  </div>
                {/each}
              </div>
            </div>
            <div class="border-border bg-card rounded-md border">
              <div class="border-border flex items-center justify-between border-b px-4 py-3">
                <Skeleton class="h-4 w-20" />
                <Skeleton class="size-4" />
              </div>
              <div class="space-y-3 p-4">
                <Skeleton class="h-4 w-full rounded-full" />
                <div class="grid grid-cols-2 gap-3">
                  {#each { length: 4 } as _}
                    <div class="space-y-1">
                      <Skeleton class="h-3 w-16" />
                      <Skeleton class="h-4 w-12" />
                    </div>
                  {/each}
                </div>
              </div>
            </div>
          </div>
        {:else if !error}
          <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <CostSnapshotPanel
              ticketSpendToday={stats.ticketSpendToday}
              ticketSpendTotal={stats.ticketSpendTotal}
              ticketInputTokens={stats.ticketInputTokens}
              ticketOutputTokens={stats.ticketOutputTokens}
              agentLifetimeTokens={stats.agentLifetimeTokens}
              ticketsCreatedToday={stats.ticketsCreatedToday}
              ticketsCompletedToday={stats.ticketsCompletedToday}
              {topCostTicket}
              {topTokenAgent}
            />
            <ExceptionPanel {exceptions} />
          </div>

          <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
            <ActivityFeedPanel {activities} class="lg:col-span-2" />
            <MemorySnapshotPanel {memory} />
          </div>

          {#if hrAdvisor && appStore.currentProject}
            {#key appStore.currentProject.id}
              <HRAdvisorPanel projectId={appStore.currentProject.id} advisor={hrAdvisor} />
            {/key}
          {/if}
        {/if}
      </div>
    {/if}
  </div>
</div>
