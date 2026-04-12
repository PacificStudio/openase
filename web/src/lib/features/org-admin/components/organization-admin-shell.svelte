<script lang="ts">
  import type {
    OrganizationDashboardSummary,
    OrganizationTokenUsageSummary,
  } from '$lib/api/contracts'
  import {
    getEffectivePermissions,
    listOrganizationMemberships,
    type EffectivePermissionsResponse,
  } from '$lib/api/auth'
  import { ApiError } from '$lib/api/client'
  import { getOrganizationSummary, getOrganizationTokenUsage } from '$lib/api/openase'
  import { PageScaffold } from '$lib/components/layout'
  import { StatCard } from '$lib/features/dashboard'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { cn } from '$lib/utils'
  import type { Component, Snippet } from 'svelte'
  import { Activity, FolderOpen, KeyRound, Settings, Shield, Users } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    organizationId,
    currentPath,
    children,
  }: {
    organizationId: string
    currentPath: string
    children: Snippet
  } = $props()

  const currentOrg = $derived(appStore.currentOrg)
  let loading = $state(false)
  let error = $state('')
  let permissions = $state<EffectivePermissionsResponse | null>(null)
  let summary = $state<OrganizationDashboardSummary | null>(null)
  let tokenSummary = $state<OrganizationTokenUsageSummary | null>(null)
  let memberStats = $state({ active: 0, invited: 0, suspended: 0 })

  type NavItem = { label: string; href: string; icon: Component }

  const t = i18nStore.t

  const adminTabs = $derived<NavItem[]>([
    {
      label: t('orgAdmin.shell.tabs.members'),
      href: `${organizationPath(organizationId)}/admin/members`,
      icon: Users,
    },
    {
      label: t('orgAdmin.shell.tabs.roles'),
      href: `${organizationPath(organizationId)}/admin/roles`,
      icon: Shield,
    },
    {
      label: t('orgAdmin.shell.tabs.credentials'),
      href: `${organizationPath(organizationId)}/admin/credentials`,
      icon: KeyRound,
    },
    {
      label: t('orgAdmin.shell.tabs.settings'),
      href: `${organizationPath(organizationId)}/admin/settings`,
      icon: Settings,
    },
  ])

  function dateRange() {
    const end = new Date()
    const start = new Date(end)
    start.setUTCDate(end.getUTCDate() - 6)
    return {
      from: start.toISOString().slice(0, 10),
      to: end.toISOString().slice(0, 10),
    }
  }

  $effect(() => {
    if (!organizationId) {
      permissions = null
      summary = null
      tokenSummary = null
      memberStats = { active: 0, invited: 0, suspended: 0 }
      return
    }

    let cancelled = false
    const controller = new AbortController()

    const load = async () => {
      loading = true
      error = ''
      try {
        const [nextPermissions, nextSummary, nextMemberships, nextTokenUsage] = await Promise.all([
          getEffectivePermissions({ orgId: organizationId }),
          getOrganizationSummary(organizationId, { signal: controller.signal }),
          listOrganizationMemberships(organizationId, { signal: controller.signal }),
          getOrganizationTokenUsage(organizationId, dateRange(), { signal: controller.signal }),
        ])
        if (cancelled) {
          return
        }
        permissions = nextPermissions
        summary = nextSummary.organization ?? null
        tokenSummary = nextTokenUsage.summary ?? null
        memberStats = {
          active: nextMemberships.filter((item) => item.status === 'active').length,
          invited: nextMemberships.filter((item) => item.activeInvitation).length,
          suspended: nextMemberships.filter((item) => item.status === 'suspended').length,
        }
      } catch (caughtError) {
        if (cancelled || controller.signal.aborted) {
          return
        }
        const loadError = t('orgAdmin.shell.errors.loadDiagnostics')
        error = caughtError instanceof ApiError ? caughtError.detail : loadError
      } finally {
        if (!cancelled) {
          loading = false
        }
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
  <title>{currentOrg?.name ?? 'Organization'} admin - OpenASE</title>
</svelte:head>

<PageScaffold
  title={t('orgAdmin.shell.pageTitle')}
  description={t('orgAdmin.shell.pageDescription')}
>
  <div class="space-y-6">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label={t('orgAdmin.shell.stats.members')}
          value={memberStats.active}
          icon={Users}
          {loading}
        />
        <StatCard
          label={t('orgAdmin.shell.stats.projects')}
          value={summary?.project_count ?? 0}
          icon={FolderOpen}
          {loading}
        />
        <StatCard
          label={t('orgAdmin.shell.stats.access')}
          value={permissions?.roles?.length ? permissions.roles.join(', ') : '—'}
          icon={Shield}
          {loading}
        />
        <StatCard
          label={t('orgAdmin.shell.stats.tokenUsage')}
          value={tokenSummary?.total_tokens ?? 0}
          icon={Activity}
          {loading}
        />
      </div>

    <div class="flex flex-col gap-6 lg:flex-row lg:gap-8">
      <nav class="flex w-full shrink-0 flex-wrap gap-1 lg:w-[180px] lg:flex-col lg:gap-0.5">
        {#each adminTabs as tab (tab.href)}
          {@const Icon = tab.icon}
          <a
            href={tab.href}
            class={cn(
              'flex shrink-0 items-center gap-2.5 rounded-md px-3 py-2 text-sm whitespace-nowrap transition-colors lg:w-full',
              currentPath === tab.href
                ? 'bg-muted text-foreground font-medium'
                : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
            )}
          >
            <Icon class="size-4 shrink-0" />
            {tab.label}
          </a>
        {/each}
      </nav>

      <div class="min-w-0 flex-1">
        {#if error}
          <div
            class="border-destructive/30 bg-destructive/5 text-destructive mb-4 rounded-md border px-4 py-3 text-sm"
          >
            {error}
          </div>
        {/if}
        {@render children()}
      </div>
    </div>
  </div>
</PageScaffold>
