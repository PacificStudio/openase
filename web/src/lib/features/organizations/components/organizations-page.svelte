<script lang="ts">
  import type { Organization } from '$lib/api/contracts'
  import {
    OrganizationCreationDialog,
    OrganizationDeleteDialog,
  } from '$lib/features/catalog-creation'
  import {
    StatCard,
    emptyWorkspaceStats,
    loadWorkspaceDashboardSummary,
    type WorkspaceOrgMetrics,
    type WorkspaceStats,
  } from '$lib/features/dashboard'
  import {
    providerAvailabilityBadgeVariant,
    providerAvailabilityLabel,
  } from '$lib/features/providers'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { formatCount, formatCurrency } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Bot, Coins, FolderOpen, Settings, Ticket } from '@lucide/svelte'
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
        <div class="flex items-center gap-1">
          <h1 class="text-foreground text-2xl font-semibold">
            {i18nStore.t('organizations.heading')}
          </h1>
          <Button variant="ghost" size="icon" href="/admin" class="text-muted-foreground size-7">
            <Settings class="size-4" />
            <span class="sr-only">
              {i18nStore.t('organizations.actions.instanceSettings')}
            </span>
          </Button>
        </div>
        <p class="text-muted-foreground mt-1 text-sm">
          {i18nStore.t('organizations.summary', {
            orgCount: organizations.length,
            projectCount: totalProjects,
            providerCount: providers.length,
          })}
        </p>
      </div>
      <Button onclick={() => (showCreateDialog = true)}>
        {i18nStore.t('organizations.actions.newOrganization')}
      </Button>
    </div>

    {#if organizations.length > 0}
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label={i18nStore.t('organizations.stats.runningAgents')}
          value={workspaceStats.runningAgents}
          icon={Bot}
        />
        <StatCard
          label={i18nStore.t('organizations.stats.activeTickets')}
          value={workspaceStats.activeTickets}
          icon={Ticket}
        />
        <StatCard
          label={i18nStore.t('organizations.stats.todaysSpend')}
          value={formatCurrency(workspaceStats.todayCost)}
          icon={Coins}
        />
        <StatCard
          label={i18nStore.t('organizations.stats.totalTokens')}
          value={formatCount(workspaceStats.totalTokens)}
          icon={FolderOpen}
        />
      </div>
    {/if}

    <section class="space-y-4">
      <h2 class="text-foreground text-lg font-semibold">
        {i18nStore.t('organizations.sections.organizations.title')}
      </h2>

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
                  class="text-destructive hover:text-destructive -mt-1 -mr-2 opacity-0 transition-opacity group-hover:opacity-100"
                  onclick={(event) => {
                    event.preventDefault()
                    event.stopPropagation()
                    openDelete(org)
                  }}
                >
                  {i18nStore.t('organizations.cards.archive')}
                </Button>
              </div>

              {#if metrics}
                {@const projectLabel =
                  metrics.projectCount === 1
                    ? 'organizations.cards.metrics.projectSingle'
                    : 'organizations.cards.metrics.projectPlural'}
                {@const providerLabel =
                  metrics.providerCount === 1
                    ? 'organizations.cards.metrics.providerSingle'
                    : 'organizations.cards.metrics.providerPlural'}
                <div
                  class="text-muted-foreground mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs"
                >
                  <span>{metrics.projectCount} {i18nStore.t(projectLabel)}</span>
                  <span>{metrics.providerCount} {i18nStore.t(providerLabel)}</span>
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
                <div class="text-muted-foreground mt-3 text-xs">
                  {i18nStore.t('organizations.cards.loadingMetrics')}
                </div>
              {/if}
            </a>
          {/each}
        </div>
      {:else if appStore.appContextLoading}
        <div class="text-muted-foreground text-sm">
          {i18nStore.t('organizations.loading')}
        </div>
      {:else}
        <button
          type="button"
          class="border-border hover:border-foreground/20 hover:bg-card w-full rounded-lg border border-dashed px-4 py-12 text-center transition-colors"
          onclick={() => (showCreateDialog = true)}
        >
          <p class="text-muted-foreground text-sm">
            {i18nStore.t('organizations.emptyState.noOrganizations')}
          </p>
          <p class="text-foreground mt-1 text-sm font-medium">
            {i18nStore.t('organizations.emptyState.createFirst')}
          </p>
        </button>
      {/if}
    </section>

    {#if providers.length > 0}
      <section class="space-y-4">
        <h2 class="text-foreground text-lg font-semibold">
          {i18nStore.t('organizations.sections.providers.title')}
        </h2>
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
</div>

<OrganizationCreationDialog bind:open={showCreateDialog} />

{#if deleteTarget}
  <OrganizationDeleteDialog organization={deleteTarget} bind:open={showDeleteDialog} />
{/if}
