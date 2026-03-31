<script lang="ts">
  import {
    emptyDashboardStats,
    loadOrganizationMetrics,
    type ProjectMetrics,
  } from '$lib/features/dashboard'
  import StatCard from '$lib/features/dashboard/components/stat-card.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { formatCurrency } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Bot, Coins, FolderOpen, Ticket as TicketIcon } from '@lucide/svelte'
  import ProjectCreationDialog from '$lib/features/catalog-creation/components/project-creation-dialog.svelte'
  import ProviderCreationDialog from '$lib/features/catalog-creation/components/provider-creation-dialog.svelte'
  import OrgProjectSection from './org-project-section.svelte'
  import OrgProviderSection from './org-provider-section.svelte'

  const currentOrg = $derived(appStore.currentOrg),
    projects = $derived(appStore.projects),
    providers = $derived(appStore.providers)

  let showProjectDialog = $state(false)
  let showProviderDialog = $state(false)

  let loading = $state(false)
  let projectMetrics = $state<Record<string, ProjectMetrics>>({})
  let orgStats = $state(emptyDashboardStats)

  $effect(() => {
    const projectList = projects
    if (projectList.length === 0) {
      projectMetrics = {}
      orgStats = emptyDashboardStats
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true

      try {
        const results = await loadOrganizationMetrics(projectList)
        if (cancelled) return
        projectMetrics = results.projectMetrics
        orgStats = results.orgStats
      } finally {
        if (!cancelled) loading = false
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  })

  const activeProjectCount = $derived(
    projects.filter((p) => {
      const s = p.status?.toLowerCase()
      return s !== 'archived' && s !== 'canceled'
    }).length,
  )
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
        label="Today's Cost"
        value={formatCurrency(orgStats.newTicketsTodayCost)}
        icon={Coins}
      />
      <StatCard label="Active Tickets" value={orgStats.activeTickets} icon={TicketIcon} />
    </div>
  {/if}

  <section class="space-y-4">
    <h2 class="text-foreground text-lg font-semibold">Projects</h2>
    <OrgProjectSection
      currentOrgId={currentOrg?.id}
      {projects}
      {projectMetrics}
      {loading}
      onCreateProject={() => (showProjectDialog = true)}
    />
  </section>

  <section class="space-y-4">
    <h2 class="text-foreground text-lg font-semibold">Providers</h2>

    {#if providers.length > 0}
      <OrgProviderSection {providers} defaultProviderId={currentOrg?.default_agent_provider_id} />
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
