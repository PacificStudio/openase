<script lang="ts">
  import type { Agent, Ticket } from '$lib/api/contracts'
  import { listAgents, listTickets } from '$lib/api/openase'
  import { ProjectCreationDialog, ProviderCreationDialog } from '$lib/features/catalog-creation'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { formatCurrency } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Bot, Coins, FolderOpen, Ticket as TicketIcon } from '@lucide/svelte'
  import StatCard from './stat-card.svelte'
  import { buildDashboardStats } from '../model'
  import type { DashboardStats } from '../types'
  import OrganizationProjectsSection from './organization-projects-section.svelte'
  import OrganizationProvidersSection from './organization-providers-section.svelte'

  const currentOrg = $derived(appStore.currentOrg),
    projects = $derived(appStore.projects),
    providers = $derived(appStore.providers)

  let showProjectDialog = $state(false)
  let showProviderDialog = $state(false)

  type ProjectMetrics = {
    runningAgents: number
    activeTickets: number
    todayCost: number
    lastActivity: string | null
  }

  const emptyStats: DashboardStats = {
    runningAgents: 0,
    activeTickets: 0,
    pendingApprovals: 0,
    newTicketsTodayCost: 0,
    projectCost: 0,
    ticketsCreatedToday: 0,
    ticketsCompletedToday: 0,
    ticketInputTokens: 0,
    ticketOutputTokens: 0,
    totalAgentTokens: 0,
    avgCycleMinutes: 0,
    prMergeRate: 0,
  }

  let loading = $state(false)
  let projectMetrics = $state<Record<string, ProjectMetrics>>({})
  let orgStats = $state<DashboardStats>(emptyStats)

  $effect(() => {
    const projectList = projects
    if (projectList.length === 0) {
      projectMetrics = {}
      orgStats = emptyStats
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true

      try {
        const results = await Promise.all(
          projectList.map(async (project) => {
            const [agentPayload, ticketPayload] = await Promise.all([
              listAgents(project.id),
              listTickets(project.id),
            ])
            return {
              projectId: project.id,
              agents: agentPayload.agents,
              tickets: ticketPayload.tickets,
            }
          }),
        )

        if (cancelled) return

        const allAgents: Agent[] = []
        const allTickets: Ticket[] = []
        const nextMetrics: Record<string, ProjectMetrics> = {}

        for (const { projectId, agents, tickets } of results) {
          allAgents.push(...agents)
          allTickets.push(...tickets)

          const stats = buildDashboardStats(agents, tickets)
          const latestTicket = tickets.reduce<Ticket | null>((latest, ticket) => {
            if (!latest || ticket.created_at > latest.created_at) return ticket
            return latest
          }, null)

          nextMetrics[projectId] = {
            runningAgents: stats.runningAgents,
            activeTickets: stats.activeTickets,
            todayCost: stats.newTicketsTodayCost,
            lastActivity: latestTicket?.created_at ?? null,
          }
        }

        projectMetrics = nextMetrics
        orgStats = buildDashboardStats(allAgents, allTickets)
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
    projects.filter((project) => {
      const status = project.status?.toLowerCase()
      return status !== 'archived' && status !== 'canceled'
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
