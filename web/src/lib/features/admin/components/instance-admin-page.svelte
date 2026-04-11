<script lang="ts">
  import { onMount } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import { getEffectivePermissions, type EffectivePermissionsResponse } from '$lib/api/auth'
  import { PageScaffold } from '$lib/components/layout'
  import {
    SecuritySettingsHumanAuthSessions,
    SecuritySettingsUserDirectory,
  } from '$lib/features/settings'
  import { authStore } from '$lib/stores/auth.svelte'
  import { cn } from '$lib/utils'
  import { currentHashSelection, writeHashSelection } from '$lib/utils/hash-state'
  import { Badge } from '$ui/badge'
  import { ArrowRight, LaptopMinimal, LockKeyhole, ShieldCheck, Users } from '@lucide/svelte'
  import type { Component } from 'svelte'

  type InstanceAdminSection = 'overview' | 'sessions' | 'users'
  const sections = ['overview', 'sessions', 'users'] as const

  type NavItem = {
    key: InstanceAdminSection
    label: string
    icon: Component
  }

  const navItems: NavItem[] = [
    { key: 'overview', label: 'Overview', icon: ShieldCheck },
    { key: 'sessions', label: 'Your sessions', icon: LaptopMinimal },
    { key: 'users', label: 'User directory', icon: Users },
  ]

  let activeSection = $state<InstanceAdminSection>('overview')
  let hashSyncReady = $state(false)

  let loading = $state(false)
  let error = $state('')
  let instancePermissions = $state<EffectivePermissionsResponse | null>(null)

  const canReadDirectory = $derived(
    instancePermissions?.permissions.includes('security_setting.read') ?? false,
  )
  const canManageDirectory = $derived(
    instancePermissions?.permissions.includes('security_setting.update') ?? false,
  )
  const currentAuthMethodLabel = $derived(
    authStore.currentAuthMethod === 'local_bootstrap_link' ? 'Local bootstrap' : 'OIDC',
  )

  function handleSelect(section: InstanceAdminSection) {
    activeSection = section
  }

  function syncSectionFromHash() {
    activeSection = currentHashSelection(sections, 'overview')
  }

  onMount(() => {
    syncSectionFromHash()
    hashSyncReady = true

    const handleHashChange = () => {
      syncSectionFromHash()
    }

    window.addEventListener('hashchange', handleHashChange)

    return () => {
      window.removeEventListener('hashchange', handleHashChange)
    }
  })

  $effect(() => {
    if (!hashSyncReady) {
      return
    }

    writeHashSelection(activeSection)
  })

  $effect(() => {
    if (!authStore.usesOIDC || !authStore.authenticated) {
      loading = false
      error = ''
      instancePermissions = null
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''
      try {
        const payload = await getEffectivePermissions({})
        if (cancelled) {
          return
        }
        instancePermissions = payload
      } catch (caughtError) {
        if (cancelled) {
          return
        }
        error =
          caughtError instanceof ApiError
            ? caughtError.detail
            : 'Failed to load instance admin permissions.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })
</script>

<PageScaffold
  title="Instance Admin"
  description="Instance-scoped user directory, session governance, auth diagnostics, and break-glass recovery guidance."
>
  <div class="flex flex-col gap-6 lg:flex-row lg:gap-8">
    <nav class="flex w-full shrink-0 flex-wrap gap-1 pb-1 lg:w-[200px] lg:flex-col lg:gap-0.5">
      {#each navItems as item (item.key)}
        {@const Icon = item.icon}
        <button
          type="button"
          class={cn(
            'flex shrink-0 items-center gap-2.5 rounded-md px-3 py-2 text-sm whitespace-nowrap transition-colors lg:w-full',
            activeSection === item.key
              ? 'bg-muted text-foreground font-medium'
              : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
          )}
          onclick={() => handleSelect(item.key)}
        >
          <Icon class="size-4 shrink-0" />
          {item.label}
        </button>
      {/each}
    </nav>

    <div class="min-w-0 flex-1 space-y-4">
      {#if activeSection === 'overview'}
        <div class="border-border bg-card rounded-lg border p-4">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div class="flex min-w-0 items-start gap-2">
              <ShieldCheck class="text-muted-foreground mt-0.5 size-4 shrink-0" />
              <div class="min-w-0 space-y-1.5">
                <div class="text-sm font-semibold">Your session</div>
                <div class="flex flex-wrap items-center gap-1.5 text-xs">
                  <Badge variant="secondary">{currentAuthMethodLabel}</Badge>
                  {#if authStore.user}
                    <Badge variant="outline" class="max-w-full truncate">
                      {authStore.user.primaryEmail}
                    </Badge>
                  {:else}
                    <Badge variant="outline">local_instance_admin:default</Badge>
                  {/if}
                </div>
                <p class="text-muted-foreground text-xs">
                  OIDC settings, bootstrap admins, and validation diagnostics.
                </p>
              </div>
            </div>
            <a
              class="border-border bg-background hover:bg-accent hover:text-accent-foreground inline-flex shrink-0 items-center justify-center gap-2 rounded-md border px-3 py-2 text-sm font-medium transition-colors"
              href="/admin/auth"
            >
              Open auth configuration
              <ArrowRight class="size-4" />
            </a>
          </div>
        </div>

        <div class="border-border bg-card space-y-2 rounded-lg border p-4">
          <div class="flex items-center gap-2">
            <LockKeyhole class="text-muted-foreground size-4 shrink-0" />
            <div class="text-sm font-semibold">Recovery</div>
            <span class="text-muted-foreground text-xs">Break-glass guidance</span>
          </div>
          <ul
            class="text-muted-foreground list-outside list-disc space-y-1.5 pl-5 text-xs leading-relaxed"
          >
            <li>Keep at least one admin able to sign in before changing auth settings.</li>
            <li class="break-words">
              OIDC failure: run
              <code class="bg-muted rounded px-1 py-0.5 font-mono text-[11px] break-all">
                openase auth break-glass disable-oidc
              </code>
              , then create a fresh local bootstrap link.
            </li>
            <li>Emergency offboarding: disable user + revoke sessions.</li>
          </ul>
        </div>

        {#if authStore.usesLocalBootstrap}
          <div class="border-border bg-card flex items-start gap-3 rounded-lg border p-4">
            <Users class="text-muted-foreground mt-0.5 size-4 shrink-0" />
            <p class="text-muted-foreground text-sm">
              Local bootstrap is the active browser auth method. Switch to OIDC when you need
              per-user governance, user directory sync, and session audits.
            </p>
          </div>
        {/if}
      {:else if activeSection === 'sessions'}
        {#if authStore.usesLocalBootstrap}
          <div class="border-border bg-card flex items-start gap-3 rounded-lg border p-4">
            <LaptopMinimal class="text-muted-foreground mt-0.5 size-4 shrink-0" />
            <p class="text-muted-foreground text-sm">
              Session governance is only available when OIDC is the active auth method. Switch away
              from local bootstrap to review active devices and the audit trail.
            </p>
          </div>
        {:else}
          <SecuritySettingsHumanAuthSessions />
        {/if}
      {:else if activeSection === 'users'}
        {#if authStore.usesLocalBootstrap}
          <div class="border-border bg-card flex items-start gap-3 rounded-lg border p-4">
            <Users class="text-muted-foreground mt-0.5 size-4 shrink-0" />
            <p class="text-muted-foreground text-sm">
              The user directory is only available when OIDC is the active auth method. Switch away
              from local bootstrap to browse and deprovision cached users.
            </p>
          </div>
        {:else if loading}
          <div class="space-y-3">
            <div class="bg-muted h-24 animate-pulse rounded-lg"></div>
            <div class="bg-muted h-64 animate-pulse rounded-lg"></div>
          </div>
        {:else if error}
          <div class="text-destructive rounded-lg border px-4 py-3 text-sm">{error}</div>
        {:else if !canReadDirectory}
          <div class="border-border bg-card rounded-lg border p-4 text-sm">
            <div class="font-medium">Instance admin access required</div>
            <div class="text-muted-foreground mt-1 text-xs">
              This page needs instance-level <code>security_setting.read</code> to browse the
              directory and <code>security_setting.update</code> for lifecycle and session governance
              actions.
            </div>
          </div>
        {:else}
          <SecuritySettingsUserDirectory
            canRead={canReadDirectory}
            canManage={canManageDirectory}
          />
        {/if}
      {/if}
    </div>
  </div>
</PageScaffold>
