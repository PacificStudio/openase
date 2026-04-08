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
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { cn } from '$lib/utils'
  import type { Snippet } from 'svelte'

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

  const adminTabs = $derived([
    { label: 'Members', href: `${organizationPath(organizationId)}/admin/members` },
    { label: 'Invitations', href: `${organizationPath(organizationId)}/admin/invitations` },
    { label: 'Roles', href: `${organizationPath(organizationId)}/admin/roles` },
    { label: 'Settings', href: `${organizationPath(organizationId)}/admin/settings` },
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
        error =
          caughtError instanceof ApiError
            ? caughtError.detail
            : 'Failed to load organization admin diagnostics.'
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
  title="Organization admin"
  description="Members, invitations, roles, and organization settings."
>
  <div class="space-y-6">
    <div class="grid gap-4 lg:grid-cols-4">
      <div class="rounded-3xl border bg-white p-4 shadow-sm">
        <div class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Members</div>
        <div class="mt-3 text-3xl font-semibold">{memberStats.active}</div>
        <div class="text-muted-foreground mt-1 text-sm">
          {memberStats.invited} pending invites · {memberStats.suspended} suspended
        </div>
      </div>
      <div class="rounded-3xl border bg-white p-4 shadow-sm">
        <div class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Projects</div>
        <div class="mt-3 text-3xl font-semibold">{summary?.project_count ?? 0}</div>
        <div class="text-muted-foreground mt-1 text-sm">
          {summary?.active_project_count ?? 0} active descendants inherit org decisions
        </div>
      </div>
      <div class="rounded-3xl border bg-white p-4 shadow-sm">
        <div class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Org access</div>
        <div class="mt-3 text-lg font-semibold">
          {permissions?.roles?.length ? permissions.roles.join(', ') : 'No org roles'}
        </div>
        <div class="text-muted-foreground mt-1 text-sm">
          {permissions?.groups?.length ?? 0} synced groups currently contribute to effective access
        </div>
      </div>
      <div class="rounded-3xl border bg-white p-4 shadow-sm">
        <div class="text-muted-foreground text-xs tracking-[0.22em] uppercase">7d diagnostics</div>
        <div class="mt-3 text-3xl font-semibold">{tokenSummary?.total_tokens ?? 0}</div>
        <div class="text-muted-foreground mt-1 text-sm">
          Avg {tokenSummary?.avg_daily_tokens ?? 0} tokens/day across this organization
        </div>
      </div>
    </div>

    <div class="flex flex-wrap gap-2 rounded-3xl border bg-white p-2 shadow-sm">
      {#each adminTabs as tab (tab.href)}
        <a
          href={tab.href}
          class={cn(
            'rounded-2xl px-4 py-2 text-sm font-medium transition-colors',
            currentPath === tab.href
              ? 'bg-slate-950 text-white'
              : 'text-muted-foreground hover:bg-slate-100 hover:text-slate-950',
          )}
        >
          {tab.label}
        </a>
      {/each}
      <a
        href={organizationPath(organizationId)}
        class="text-muted-foreground rounded-2xl px-4 py-2 text-sm font-medium transition-colors hover:bg-slate-100 hover:text-slate-950"
      >
        Back to org dashboard
      </a>
    </div>

    {#if error}
      <div
        class="border-destructive/30 bg-destructive/5 text-destructive rounded-2xl border px-4 py-3 text-sm"
      >
        {error}
      </div>
    {:else if loading}
      <div class="text-muted-foreground rounded-2xl border border-dashed px-4 py-8 text-sm">
        Loading organization admin diagnostics…
      </div>
    {/if}

    {@render children()}
  </div>
</PageScaffold>
