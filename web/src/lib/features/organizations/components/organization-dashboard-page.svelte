<script lang="ts">
  import { ProjectCreationDialog } from '$lib/features/catalog-creation'
  import {
    OrgProjectCard,
    StatCard,
    emptyOrganizationDashboardStats,
    loadOrganizationDashboardSummary,
    type DashboardStats,
    type ProjectMetrics,
  } from '$lib/features/dashboard'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { formatCurrency } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Bot, Coins, FolderOpen, Ticket as TicketIcon } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  const currentOrg = $derived(appStore.currentOrg),
    projects = $derived(appStore.projects)

  let showProjectDialog = $state(false)
  let loading = $state(false)
  let projectMetrics = $state<Record<string, ProjectMetrics>>({})
  let orgStats = $state<DashboardStats>(emptyOrganizationDashboardStats)
  let activeProjectCount = $state(0)

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
</script>

<svelte:head>
  <title>
    {(currentOrg?.name ?? i18nStore.t('organizations.dashboard.title.defaultOrg')) + ' - OpenASE'}
  </title>
</svelte:head>

<div data-testid="route-scroll-container" class="min-h-0 flex-1 overflow-y-auto">
  <div class="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6 py-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div class="flex flex-col gap-2">
        <p class="text-muted-foreground text-sm">
          <a href="/" class="hover:text-foreground transition-colors">
            {i18nStore.t('organizations.dashboard.breadcrumb.workspace')}
          </a>
          <span class="mx-2">/</span>
          <a
            href={currentOrg ? organizationPath(currentOrg.id) : '/'}
            class="hover:text-foreground transition-colors"
          >
            {currentOrg?.name ?? i18nStore.t('organizations.dashboard.breadcrumb.organization')}
          </a>
        </p>
        <div>
          <h1 class="text-foreground text-2xl font-semibold">
            {currentOrg?.name ?? i18nStore.t('organizations.dashboard.heading.defaultOrg')}
          </h1>
          <p class="text-muted-foreground mt-1 text-sm">
            {projects.length}
            {projects.length === 1
              ? i18nStore.t('organizations.dashboard.projects.single')
              : i18nStore.t('organizations.dashboard.projects.plural')}
          </p>
        </div>
      </div>
      <div class="flex gap-2">
          <Button onclick={() => (showProjectDialog = true)}>
          {i18nStore.t('organizations.dashboard.actions.newProject')}
          </Button>
      </div>
    </div>

    {#if projects.length > 0}
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label={i18nStore.t('organizations.dashboard.stats.activeProjects')}
          value={activeProjectCount}
          icon={FolderOpen}
        />
        <StatCard
          label={i18nStore.t('organizations.dashboard.stats.runningAgents')}
          value={orgStats.runningAgents}
          icon={Bot}
        />
        <StatCard
          label={i18nStore.t('organizations.dashboard.stats.todaysSpend')}
          value={formatCurrency(orgStats.ticketSpendToday)}
          icon={Coins}
        />
        <StatCard
          label={i18nStore.t('organizations.dashboard.stats.activeTickets')}
          value={orgStats.activeTickets}
          icon={TicketIcon}
        />
      </div>
    {/if}

    <section class="space-y-4">
      <h2 class="text-foreground text-lg font-semibold">
        {i18nStore.t('organizations.dashboard.sections.projects.title')}
      </h2>

      {#if projects.length > 0}
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {#each projects as project (project.id)}
            <OrgProjectCard
              currentOrgId={currentOrg?.id}
              {project}
              metrics={projectMetrics[project.id]}
              {loading}
            />
          {/each}
        </div>
      {:else}
        <button
          type="button"
          class="border-border hover:border-foreground/20 hover:bg-card w-full rounded-lg border border-dashed px-4 py-12 text-center transition-colors"
          onclick={() => (showProjectDialog = true)}
        >
          <p class="text-muted-foreground text-sm">
            {i18nStore.t('organizations.dashboard.empty.noProjects')}
          </p>
          <p class="text-foreground mt-1 text-sm font-medium">
            {i18nStore.t('organizations.dashboard.empty.createFirst')}
          </p>
        </button>
      {/if}
    </section>
  </div>
</div>

{#if currentOrg}
  <ProjectCreationDialog
    orgId={currentOrg.id}
    defaultProviderId={currentOrg.default_agent_provider_id}
    providers={appStore.providers}
    bind:open={showProjectDialog}
  />
{/if}

{#if !currentOrg && appStore.appContextLoading}
  <div class="mx-auto w-full max-w-6xl px-6 pb-6">
    <div class="text-muted-foreground text-sm">
      {i18nStore.t('organizations.dashboard.loading')}
    </div>
  </div>
{/if}
