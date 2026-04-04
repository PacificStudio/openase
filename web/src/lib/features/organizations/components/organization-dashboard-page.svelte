<script lang="ts">
  import { ProviderCreationDialog, ProjectCreationDialog } from '$lib/features/catalog-creation'
  import {
    OrgProjectCard,
    StatCard,
    emptyOrganizationDashboardStats,
    loadOrganizationDashboardSummary,
    type DashboardStats,
    type ProjectMetrics,
  } from '$lib/features/dashboard'
  import {
    providerAvailabilityBadgeVariant,
    providerAvailabilityLabel,
    summarizeAgentProviderRateLimit,
  } from '$lib/features/providers'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { formatCurrency } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Bot, Coins, FolderOpen, Ticket as TicketIcon } from '@lucide/svelte'

  const currentOrg = $derived(appStore.currentOrg),
    projects = $derived(appStore.projects),
    providers = $derived(appStore.providers)

  let showProjectDialog = $state(false)
  let showProviderDialog = $state(false)
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
  <title>{currentOrg?.name ?? 'Organization'} - OpenASE</title>
</svelte:head>

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
        <h1 class="text-foreground text-2xl font-semibold">{currentOrg?.name ?? 'Organization'}</h1>
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
      <StatCard label="Active Projects" value={activeProjectCount} icon={FolderOpen} />
      <StatCard label="Running Agents" value={orgStats.runningAgents} icon={Bot} />
      <StatCard
        label="Today's Spend"
        value={formatCurrency(orgStats.ticketSpendToday)}
        icon={Coins}
      />
      <StatCard label="Active Tickets" value={orgStats.activeTickets} icon={TicketIcon} />
    </div>
  {/if}

  <section class="space-y-4">
    <h2 class="text-foreground text-lg font-semibold">Projects</h2>

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
        <p class="text-muted-foreground text-sm">No projects yet.</p>
        <p class="text-foreground mt-1 text-sm font-medium">
          Create your first project to get started
        </p>
      </button>
    {/if}
  </section>

  <section class="space-y-4">
    <h2 class="text-foreground text-lg font-semibold">Providers</h2>

    {#if providers.length > 0}
      <div class="border-border divide-border divide-y rounded-lg border">
        {#each providers as provider (provider.id)}
          {@const rateLimit = summarizeAgentProviderRateLimit(provider)}
          <div class="flex items-center justify-between gap-4 px-4 py-3">
            <div class="min-w-0 flex-1">
              <p class="text-foreground truncate text-sm font-medium">{provider.name}</p>
              <p class="text-muted-foreground truncate text-xs">{provider.model_name}</p>
              {#if rateLimit}
                <div class="bg-muted/30 mt-2 rounded-lg border px-3 py-2 text-[11px]">
                  <div class="flex items-center justify-between gap-3">
                    <span class="text-muted-foreground">Rate limit</span>
                    <span class="text-foreground font-medium">{rateLimit.headline}</span>
                  </div>
                  <div class="text-muted-foreground mt-1">{rateLimit.detail}</div>
                  <div class="text-muted-foreground mt-1">{rateLimit.updatedLabel}</div>
                </div>
              {/if}
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <Badge variant={providerAvailabilityBadgeVariant(provider.availability_state)}>
                {providerAvailabilityLabel(provider.availability_state)}
              </Badge>
              {#if currentOrg?.default_agent_provider_id === provider.id}
                <Badge variant="secondary">Default</Badge>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <button
        type="button"
        class="border-border hover:border-foreground/20 hover:bg-card w-full rounded-lg border border-dashed px-4 py-8 text-center transition-colors"
        onclick={() => (showProviderDialog = true)}
      >
        <p class="text-muted-foreground text-sm">No providers configured.</p>
        <p class="text-foreground mt-1 text-sm font-medium">
          Add a provider to enable agent execution
        </p>
      </button>
    {/if}
  </section>
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
