<script lang="ts">
  import type { Organization } from '$lib/api/contracts'
  import OrganizationCreationDialog from '$lib/features/catalog-creation/components/organization-creation-dialog.svelte'
  import OrganizationDeleteDialog from '$lib/features/catalog-creation/components/organization-delete-dialog.svelte'
  import StatCard from '$lib/features/dashboard/components/stat-card.svelte'
  import {
    providerAvailabilityBadgeVariant,
    providerAvailabilityLabel,
  } from '$lib/features/providers'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { formatCount, formatCurrency } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Bot, Coins, FolderOpen, Ticket } from '@lucide/svelte'

  const organizations = $derived(appStore.organizations)
  const providers = $derived(appStore.providers)

  let showCreateDialog = $state(false)
  let deleteTarget = $state<Organization | null>(null)
  let showDeleteDialog = $state(false)

  // TODO: Replace mock data with real API aggregation.
  // For each org, call listProjects(orgId), then for each project call
  // listAgents(projectId) + listTickets(projectId), and aggregate using
  // buildDashboardStats(). Consider adding a backend endpoint
  // GET /api/v1/workspace/summary for efficiency at scale.

  type OrgMetrics = {
    projectCount: number
    providerCount: number
    runningAgents: number
    activeTickets: number
    todayCost: number
  }

  // TODO: wire to real per-org aggregation
  // Need to call listProjects(orgId), then listAgents/listTickets per project.
  // Provider count also needs listProviders(orgId) per org.
  const orgMetrics = $derived<Record<string, OrgMetrics>>(
    Object.fromEntries(
      organizations.map((org) => [
        org.id,
        {
          projectCount: 0,
          providerCount: 0,
          runningAgents: 0,
          activeTickets: 0,
          todayCost: 0,
        },
      ]),
    ),
  )

  // TODO: wire to real workspace-wide aggregation
  const workspaceStats = $derived({
    runningAgents: Object.values(orgMetrics).reduce((s, m) => s + m.runningAgents, 0),
    activeTickets: Object.values(orgMetrics).reduce((s, m) => s + m.activeTickets, 0),
    todayCost: Object.values(orgMetrics).reduce((s, m) => s + m.todayCost, 0),
    totalTokens: 0,
  })

  const totalProjects = $derived(
    Object.values(orgMetrics).reduce((s, m) => s + m.projectCount, 0),
  )

  function openDelete(org: Organization) {
    deleteTarget = org
    showDeleteDialog = true
  }
</script>

<svelte:head>
  <title>Workspace - OpenASE</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6 py-6">
  <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
    <div>
      <h1 class="text-foreground text-2xl font-semibold">Workspace</h1>
      <p class="text-muted-foreground mt-1 text-sm">
        {organizations.length}
        {organizations.length === 1 ? 'organization' : 'organizations'} · {totalProjects}
        {totalProjects === 1 ? 'project' : 'projects'} · {providers.length}
        {providers.length === 1 ? 'provider' : 'providers'}
      </p>
    </div>
    <Button onclick={() => (showCreateDialog = true)}>New organization</Button>
  </div>

  {#if organizations.length > 0}
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard label="Running Agents" value={workspaceStats.runningAgents} icon={Bot} />
      <StatCard label="Active Tickets" value={workspaceStats.activeTickets} icon={Ticket} />
      <StatCard label="Today's Cost" value={formatCurrency(workspaceStats.todayCost)} icon={Coins} />
      <StatCard
        label="Total Tokens"
        value={formatCount(workspaceStats.totalTokens)}
        icon={FolderOpen}
      />
    </div>
  {/if}

  <section class="space-y-4">
    <h2 class="text-foreground text-lg font-semibold">Organizations</h2>

    {#if organizations.length > 0}
      <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {#each organizations as org (org.id)}
          {@const metrics = orgMetrics[org.id]}
          <a
            href={organizationPath(org.id)}
            class="border-border bg-card hover:bg-muted/30 group rounded-lg border p-5 transition-colors"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex-1">
                <h3 class="text-foreground truncate text-sm font-semibold">{org.name}</h3>
                <p class="text-muted-foreground mt-0.5 truncate text-xs">{org.slug}</p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                class="text-destructive hover:text-destructive -mr-2 -mt-1 opacity-0 transition-opacity group-hover:opacity-100"
                onclick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  openDelete(org)
                }}
              >
                Archive
              </Button>
            </div>

            {#if metrics}
              <div class="text-muted-foreground mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
                <span>{metrics.projectCount} project{metrics.projectCount !== 1 ? 's' : ''}</span>
                <span>{metrics.providerCount} provider{metrics.providerCount !== 1 ? 's' : ''}</span>
                <span class="flex items-center gap-1">
                  <Bot class="size-3" />
                  {metrics.runningAgents}
                </span>
                <span class="flex items-center gap-1">
                  <Ticket class="size-3" />
                  {metrics.activeTickets}
                </span>
                <span class="flex items-center gap-1">
                  <Coins class="size-3" />
                  {formatCurrency(metrics.todayCost)}
                </span>
              </div>
            {/if}
          </a>
        {/each}
      </div>
    {:else if appStore.appContextLoading}
      <div class="text-muted-foreground text-sm">Loading organizations…</div>
    {:else}
      <button
        type="button"
        class="border-border hover:border-foreground/20 hover:bg-card w-full rounded-lg border border-dashed px-4 py-12 text-center transition-colors"
        onclick={() => (showCreateDialog = true)}
      >
        <p class="text-muted-foreground text-sm">No organizations yet.</p>
        <p class="text-foreground mt-1 text-sm font-medium">
          Create your first organization to get started
        </p>
      </button>
    {/if}
  </section>

  {#if providers.length > 0}
    <section class="space-y-4">
      <h2 class="text-foreground text-lg font-semibold">Providers</h2>
      <div class="border-border divide-border divide-y rounded-lg border">
        {#each providers as provider (provider.id)}
          <div class="flex items-center justify-between gap-4 px-4 py-3">
            <div class="min-w-0">
              <p class="text-foreground truncate text-sm font-medium">{provider.name}</p>
              <p class="text-muted-foreground truncate text-xs">{provider.model_name}</p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <Badge variant={providerAvailabilityBadgeVariant(provider.availability_state)}>
                {providerAvailabilityLabel(provider.availability_state)}
              </Badge>
            </div>
          </div>
        {/each}
      </div>
    </section>
  {/if}
</div>

<OrganizationCreationDialog bind:open={showCreateDialog} />

{#if deleteTarget}
  <OrganizationDeleteDialog organization={deleteTarget} bind:open={showDeleteDialog} />
{/if}
