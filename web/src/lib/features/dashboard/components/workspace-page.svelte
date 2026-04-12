<script lang="ts">
  import type { Organization } from '$lib/api/contracts'
  import {
    OrganizationCreationDialog,
    OrganizationDeleteDialog,
  } from '$lib/features/catalog-creation'
  import {
    emptyWorkspaceStats,
    loadWorkspaceDashboardSummary,
    type WorkspaceOrgMetrics,
    type WorkspaceStats,
  } from '$lib/features/dashboard/workspace-summary'
  import {
    providerAvailabilityBadgeVariant,
    providerAvailabilityLabel,
    summarizeAgentProviderRateLimit,
    ProviderRateLimitDisplay,
  } from '$lib/features/providers'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { formatCount, formatCurrency } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Skeleton } from '$ui/skeleton'
  import { Bot, Coins, FolderOpen, Ticket } from '@lucide/svelte'
  import StatCard from './stat-card.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  const organizations = $derived(appStore.organizations)
  const providers = $derived(appStore.providers)

  let showCreateDialog = $state(false)
  let deleteTarget = $state<Organization | null>(null)
  let showDeleteDialog = $state(false)

  let loading = $state(false)
  let orgMetrics = $state<Record<string, WorkspaceOrgMetrics>>({})
  let workspaceStats = $state<WorkspaceStats>(emptyWorkspaceStats)
  let totalProjects = $state(0)

  $effect(() => {
    const refreshKey = `${appStore.appContextFetchedAt}:${organizations.map((org) => org.id).join(',')}`
    void refreshKey

    let cancelled = false
    const controller = new AbortController()

    const load = async () => {
      loading = true

      try {
        const summary = await loadWorkspaceDashboardSummary({ signal: controller.signal })
        if (cancelled) return

        orgMetrics = summary.orgMetrics
        workspaceStats = summary.workspaceStats
        totalProjects = summary.totalProjects
      } catch {
        if (cancelled || controller.signal.aborted) return
        orgMetrics = {}
        workspaceStats = emptyWorkspaceStats
        totalProjects = 0
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

  function openDelete(org: Organization) {
    deleteTarget = org
    showDeleteDialog = true
  }
</script>

<svelte:head>
  <title>{i18nStore.t('organizations.page.title')}</title>
</svelte:head>

<div data-testid="route-scroll-container" class="min-h-0 flex-1 overflow-y-auto">
  <div class="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6 py-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h1 class="text-foreground text-2xl font-semibold">
          {i18nStore.t('dashboard.workspace.title')}
        </h1>
        <p class="text-muted-foreground mt-1 text-sm">
          {i18nStore.t('dashboard.workspace.summary.organization', {
            count: organizations.length,
          })}
          &middot;
          {i18nStore.t('dashboard.workspace.summary.project', {
            count: totalProjects,
          })}
          &middot;
          {i18nStore.t('dashboard.workspace.summary.provider', {
            count: providers.length,
          })}
        </p>
      </div>
      <Button onclick={() => (showCreateDialog = true)}>
        {i18nStore.t('dashboard.workspace.actions.newOrganization')}
      </Button>
    </div>

    {#if organizations.length > 0}
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label={i18nStore.t('dashboard.workspace.stats.runningAgents')}
          value={workspaceStats.runningAgents}
          icon={Bot}
          {loading}
        />
        <StatCard
          label={i18nStore.t('dashboard.workspace.stats.activeTickets')}
          value={workspaceStats.activeTickets}
          icon={Ticket}
          {loading}
        />
        <StatCard
          label={i18nStore.t('dashboard.workspace.stats.todaySpend')}
          value={formatCurrency(workspaceStats.todayCost)}
          icon={Coins}
          {loading}
        />
        <StatCard
          label={i18nStore.t('dashboard.workspace.stats.totalTokens')}
          value={formatCount(workspaceStats.totalTokens)}
          icon={FolderOpen}
          {loading}
        />
      </div>
    {/if}

    <section class="space-y-4">
      <h2 class="text-foreground text-lg font-semibold">
        {i18nStore.t('dashboard.workspace.sections.organizations')}
      </h2>

      {#if organizations.length > 0}
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {#each organizations as org (org.id)}
            {@const metrics = orgMetrics[org.id]}
            <a
              href={organizationPath(org.id)}
              class="border-border bg-card hover:bg-muted/30 hover-lift group rounded-lg border p-5 transition-colors"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <h3 class="text-foreground truncate text-sm font-semibold">{org.name}</h3>
                  <p class="text-muted-foreground mt-0.5 truncate text-xs">{org.slug}</p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-destructive hover:text-destructive -mt-1 -mr-2 opacity-0 transition-opacity group-hover:opacity-100"
                  onclick={(event) => {
                    event.preventDefault()
                    event.stopPropagation()
                    openDelete(org)
                  }}
                >
                  {i18nStore.t('dashboard.workspace.actions.archive')}
                </Button>
              </div>

              {#if metrics}
                <div
                  class="text-muted-foreground mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs"
                >
                  <span>
                    {i18nStore.t('dashboard.workspace.metrics.projects', {
                      count: metrics.projectCount,
                    })}
                  </span>
                  <span>
                    {i18nStore.t('dashboard.workspace.metrics.providers', {
                      count: metrics.providerCount,
                    })}
                  </span>
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
              {:else if loading}
                <div class="mt-3 flex items-center gap-4">
                  <Skeleton class="h-3.5 w-16" />
                  <Skeleton class="h-3.5 w-16" />
                  <Skeleton class="h-3.5 w-10" />
                  <Skeleton class="h-3.5 w-10" />
                  <Skeleton class="h-3.5 w-14" />
                </div>
              {/if}
            </a>
          {/each}
        </div>
      {:else if appStore.appContextLoading}
        <div class="text-muted-foreground text-sm">
          {i18nStore.t('dashboard.workspace.messages.loadingOrganizations')}
        </div>
      {:else}
        <button
          type="button"
          class="border-border hover:border-foreground/20 hover:bg-card w-full rounded-lg border border-dashed px-4 py-12 text-center transition-colors"
          onclick={() => (showCreateDialog = true)}
        >
          <p class="text-muted-foreground text-sm">
            {i18nStore.t('dashboard.workspace.messages.noOrganizations')}
          </p>
          <p class="text-foreground mt-1 text-sm font-medium">
            {i18nStore.t('dashboard.workspace.messages.firstOrganization')}
          </p>
        </button>
      {/if}
    </section>

    {#if providers.length > 0}
      <section class="space-y-4">
        <h2 class="text-foreground text-lg font-semibold">
          {i18nStore.t('dashboard.workspace.sections.providers')}
        </h2>
        <div class="border-border divide-border divide-y rounded-lg border">
          {#each providers as provider (provider.id)}
            {@const rateLimit = summarizeAgentProviderRateLimit(provider)}
            <div class="flex items-center justify-between gap-4 px-4 py-3">
              <div class="min-w-0 flex-1">
                <p class="text-foreground truncate text-sm font-medium">{provider.name}</p>
                <p class="text-muted-foreground truncate text-xs">{provider.model_name}</p>
                {#if rateLimit}
                  <div class="mt-2">
                    <ProviderRateLimitDisplay {rateLimit} />
                  </div>
                {/if}
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
</div>

<OrganizationCreationDialog bind:open={showCreateDialog} />

{#if deleteTarget}
  <OrganizationDeleteDialog organization={deleteTarget} bind:open={showDeleteDialog} />
{/if}
