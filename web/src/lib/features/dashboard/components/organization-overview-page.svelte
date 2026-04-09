<script lang="ts">
  import { ProjectCreationDialog, ProviderCreationDialog } from '$lib/features/catalog-creation'
  import {
    emptyOrganizationDashboardStats,
    loadOrganizationDashboardSummary,
    type ProjectMetrics,
  } from '$lib/features/dashboard/organization-summary'
  import {
    emptyOrganizationTokenUsageAnalytics,
    loadOrganizationTokenUsage,
  } from '$lib/features/dashboard/organization-token-usage'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { formatCurrency } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Bot, Coins, FolderOpen, Ticket as TicketIcon } from '@lucide/svelte'
  import StatCard from './stat-card.svelte'
  import type {
    DashboardStats,
    OrganizationTokenUsageAnalytics,
    OrganizationTokenUsageRange,
  } from '../types'
  import OrganizationTokenAnalyticsPanel from './organization-token-analytics-panel.svelte'
  import OrganizationProjectsSection from './organization-projects-section.svelte'
  import OrganizationProvidersSection from './organization-providers-section.svelte'

  const currentOrg = $derived(appStore.currentOrg),
    projects = $derived(appStore.projects),
    providers = $derived(appStore.providers)

  let showProjectDialog = $state(false)
  let showProviderDialog = $state(false)

  let loading = $state(false)
  let analyticsLoading = $state(false)
  let projectMetrics = $state<Record<string, ProjectMetrics>>({})
  let orgStats = $state<DashboardStats>(emptyOrganizationDashboardStats)
  let activeProjectCount = $state(0)
  let selectedUsageRange = $state<OrganizationTokenUsageRange>(30)
  let tokenUsage = $state<OrganizationTokenUsageAnalytics>(emptyOrganizationTokenUsageAnalytics(30))

  $effect(() => {
    const orgId = currentOrg?.id
    const refreshKey = `${orgId ?? ''}:${appStore.appContextFetchedAt}`
    void refreshKey

    if (!orgId) {
      projectMetrics = {}
      orgStats = emptyOrganizationDashboardStats
      activeProjectCount = 0
      return
    }

    let cancelled = false
    const controller = new AbortController()

    const load = async () => {
      loading = true

      try {
        const summary = await loadOrganizationDashboardSummary(orgId, {
          signal: controller.signal,
        })
        if (cancelled) return

        projectMetrics = summary.projectMetrics
        activeProjectCount = summary.activeProjectCount
        orgStats = summary.orgStats
      } catch {
        if (cancelled || controller.signal.aborted) return
        projectMetrics = {}
        orgStats = emptyOrganizationDashboardStats
        activeProjectCount = 0
      } finally {
        if (!cancelled) loading = false
      }
    }

    void load()
    return () => {
      cancelled = true
      controller.abort()
    }
  })

  $effect(() => {
    const orgId = currentOrg?.id
    const refreshKey = `${orgId ?? ''}:${selectedUsageRange}:${appStore.appContextFetchedAt}`
    void refreshKey

    if (!orgId) {
      tokenUsage = emptyOrganizationTokenUsageAnalytics(selectedUsageRange)
      return
    }

    let cancelled = false
    const controller = new AbortController()

    const load = async () => {
      analyticsLoading = true

      try {
        const analytics = await loadOrganizationTokenUsage(orgId, selectedUsageRange, {
          signal: controller.signal,
        })
        if (cancelled) return

        tokenUsage = analytics
      } catch {
        if (cancelled || controller.signal.aborted) return
        tokenUsage = emptyOrganizationTokenUsageAnalytics(selectedUsageRange)
      } finally {
        if (!cancelled) analyticsLoading = false
      }
    }

    void load()
    return () => {
      cancelled = true
      controller.abort()
    }
  })
</script>

<svelte:head>
  <title>{currentOrg?.name ?? 'Organization'} - OpenASE</title>
</svelte:head>

<div data-testid="route-scroll-container" class="min-h-0 flex-1 overflow-y-auto">
  <div class="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6 py-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div class="flex flex-col gap-2">
        <p class="text-muted-foreground text-sm">
          <a href="/" class="hover:text-foreground transition-colors">Workspace</a>
          <span class="mx-2">/</span>
          <a
            href={currentOrg ? organizationPath(currentOrg.id) : '/'}
            class="hover:text-foreground transition-colors"
          >
            {currentOrg?.name ?? 'Organization'}
          </a>
        </p>
        <div>
          <h1 class="text-foreground text-2xl font-semibold">
            {currentOrg?.name ?? 'Organization'}
          </h1>
          <p class="text-muted-foreground mt-1 text-sm">
            {projects.length}
            {projects.length === 1 ? 'project' : 'projects'} · {providers.length}
            {providers.length === 1 ? 'provider' : 'providers'}
          </p>
        </div>
      </div>

      <div class="flex gap-2">
        <Button variant="outline" onclick={() => (showProviderDialog = true)}>Add provider</Button>
        <Button onclick={() => (showProjectDialog = true)}>New project</Button>
      </div>
    </div>

    {#if projects.length > 0}
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Active Projects" value={activeProjectCount} icon={FolderOpen} {loading} />
        <StatCard label="Running Agents" value={orgStats.runningAgents} icon={Bot} {loading} />
        <StatCard
          label="Today's Spend"
          value={formatCurrency(orgStats.ticketSpendToday)}
          icon={Coins}
          {loading}
        />
        <StatCard
          label="Active Tickets"
          value={orgStats.activeTickets}
          icon={TicketIcon}
          {loading}
        />
      </div>

      <OrganizationTokenAnalyticsPanel
        analytics={tokenUsage}
        selectedRange={selectedUsageRange}
        loading={analyticsLoading}
        onSelectRange={(range) => {
          selectedUsageRange = range
        }}
      />
    {/if}

    <OrganizationProjectsSection
      currentOrgId={currentOrg?.id ?? null}
      {projects}
      {projectMetrics}
      {loading}
      onCreateProject={() => (showProjectDialog = true)}
    />

    <OrganizationProvidersSection
      {providers}
      defaultProviderId={currentOrg?.default_agent_provider_id ?? null}
      onAddProvider={() => (showProviderDialog = true)}
    />
  </div>
</div>

{#if currentOrg}
  <ProjectCreationDialog
    orgId={currentOrg.id}
    defaultProviderId={currentOrg.default_agent_provider_id}
    {providers}
    bind:open={showProjectDialog}
  />

  <ProviderCreationDialog orgId={currentOrg.id} bind:open={showProviderDialog} />
{/if}

{#if !currentOrg && appStore.appContextLoading}
  <div class="mx-auto w-full max-w-6xl px-6 pb-6">
    <div class="text-muted-foreground text-sm">Loading organization…</div>
  </div>
{/if}
