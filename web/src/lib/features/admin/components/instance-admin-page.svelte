<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { getEffectivePermissions, type EffectivePermissionsResponse } from '$lib/api/auth'
  import { PageScaffold } from '$lib/components/layout'
  import {
    SecuritySettingsHumanAuthSessions,
    SecuritySettingsUserDirectory,
  } from '$lib/features/settings'
  import { authStore } from '$lib/stores/auth.svelte'
  import { Badge } from '$ui/badge'
  import { ArrowRight, LockKeyhole, ShieldCheck, Users } from '@lucide/svelte'

  let loading = $state(false)
  let error = $state('')
  let instancePermissions = $state<EffectivePermissionsResponse | null>(null)

  const canReadDirectory = $derived(
    instancePermissions?.permissions.includes('security.read') ?? false,
  )
  const canManageDirectory = $derived(
    instancePermissions?.permissions.includes('security.manage') ?? false,
  )

  $effect(() => {
    if (authStore.authMode !== 'oidc' || !authStore.authenticated) {
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
  <div class="space-y-4">
    <div class="grid gap-4 lg:grid-cols-3">
      <div class="border-border bg-card space-y-3 rounded-lg border p-4">
        <div class="flex items-center gap-2">
          <ShieldCheck class="text-muted-foreground size-4" />
          <div class="text-sm font-semibold">Your session</div>
        </div>
        <div class="flex flex-wrap gap-2 text-xs">
          <Badge variant="secondary"
            >{authStore.authMode === 'oidc' ? 'OIDC' : 'Disabled mode'}</Badge
          >
          {#if authStore.user}
            <Badge variant="outline">{authStore.user.primaryEmail}</Badge>
          {:else}
            <Badge variant="outline">local_instance_admin:default</Badge>
          {/if}
        </div>
      </div>

      <div class="border-border bg-card space-y-3 rounded-lg border p-4">
        <div class="flex items-center gap-2">
          <LockKeyhole class="text-muted-foreground size-4" />
          <div class="text-sm font-semibold">Recovery</div>
        </div>
        <ul class="text-muted-foreground space-y-1 text-xs list-disc list-inside">
          <li>Keep at least one admin able to sign in before changing auth settings</li>
          <li>OIDC failure: set <code>auth.mode: disabled</code> to restore local admin access</li>
          <li>Emergency offboarding: disable user + revoke sessions</li>
        </ul>
      </div>

      <div class="border-border bg-card space-y-3 rounded-lg border p-4">
        <div class="flex items-center gap-2">
          <ArrowRight class="text-muted-foreground size-4" />
          <div class="text-sm font-semibold">Auth configuration</div>
        </div>
        <p class="text-muted-foreground text-xs">
          OIDC settings, bootstrap admins, and validation diagnostics.
        </p>
        <a
          class="inline-flex items-center gap-2 text-sm font-medium text-sky-700 hover:text-sky-800"
          href="/admin/auth"
        >
          Open auth configuration
          <ArrowRight class="size-4" />
        </a>
      </div>
    </div>

    {#if authStore.authMode !== 'oidc'}
      <div class="border-border bg-card flex items-center gap-3 rounded-lg border p-4">
        <Users class="text-muted-foreground size-4 shrink-0" />
        <p class="text-muted-foreground text-sm">
          Running in single-user disabled mode. Switch to OIDC for per-user governance and
          audit logs.
        </p>
      </div>
    {:else}
      <SecuritySettingsHumanAuthSessions />

      {#if loading}
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
            This page needs instance-level <code>security.read</code> to browse the directory and
            <code>security.manage</code> for lifecycle and session governance actions.
          </div>
        </div>
      {:else}
        <SecuritySettingsUserDirectory canRead={canReadDirectory} canManage={canManageDirectory} />
      {/if}
    {/if}
  </div>
</PageScaffold>
