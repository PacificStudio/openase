<script lang="ts">
  import { goto } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import {
    logoutHumanSession,
    normalizeReturnTo,
    type ManagedAuthSession,
    type SessionGovernanceResponse,
  } from '$lib/api/auth'
  import {
    getSessionGovernance,
    revokeAllOtherAuthSessions,
    revokeAuthSession,
  } from '$lib/api/openase'
  import { authStore } from '$lib/stores/auth.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { LaptopMinimal, LogOut, ShieldCheck, Smartphone, TabletSmartphone } from '@lucide/svelte'
  import { formatTimestamp } from './security-settings-human-auth.model'
  import SecuritySettingsHumanAuthAuditTimeline from './security-settings-human-auth-audit-timeline.svelte'

  let governance = $state<SessionGovernanceResponse | null>(null)
  let loading = $state(false)
  let error = $state('')
  let actionKey = $state('')

  async function loadGovernanceState() {
    loading = true
    error = ''

    try {
      governance = await getSessionGovernance()
    } catch (caughtError) {
      governance = null
      error =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load session governance.'
    } finally {
      loading = false
    }
  }

  $effect(() => {
    if (!authStore.sessionGovernanceAvailable) {
      governance = null
      loading = false
      error = ''
      return
    }

    let cancelled = false

    const load = async () => {
      await loadGovernanceState()
      if (cancelled) {
        loading = false
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  function iconForSession(session: ManagedAuthSession) {
    switch (session.device.kind) {
      case 'mobile':
        return Smartphone
      case 'tablet':
        return TabletSmartphone
      default:
        return LaptopMinimal
    }
  }

  async function handleCurrentSessionLogout() {
    const key = 'session:current'
    actionKey = key
    error = ''

    try {
      await logoutHumanSession()
    } catch (caughtError) {
      if (!(caughtError instanceof ApiError) || caughtError.status !== 401) {
        const message = caughtError instanceof ApiError ? caughtError.detail : 'Failed to log out.'
        error = message
        toastStore.error(message)
        actionKey = ''
        return
      }
    }

    authStore.clear()
    actionKey = ''
    await goto(
      `/login?return_to=${encodeURIComponent(normalizeReturnTo(window.location.pathname + window.location.search + window.location.hash))}`,
    )
  }

  async function handleSessionAction(session: ManagedAuthSession) {
    if (session.current) {
      await handleCurrentSessionLogout()
      return
    }

    const key = `session:${session.id}`
    actionKey = key
    error = ''

    try {
      await revokeAuthSession(session.id)
      await loadGovernanceState()
      toastStore.success('Session revoked.')
    } catch (caughtError) {
      const message =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to revoke session.'
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }

  async function handleRevokeOthers() {
    actionKey = 'session:others'
    error = ''

    try {
      const payload = await revokeAllOtherAuthSessions()
      await loadGovernanceState()
      toastStore.success(
        payload.revoked_count > 0
          ? `Revoked ${payload.revoked_count} other session(s).`
          : 'No other active sessions to revoke.',
      )
    } catch (caughtError) {
      const message =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to revoke other sessions.'
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <ShieldCheck class="text-muted-foreground size-4" />
    <h4 class="text-sm font-semibold">Session governance</h4>
  </div>

  <p class="text-muted-foreground text-xs">
    Review active devices, revoke stale sessions, and inspect the auth audit trail for browser
    access.
  </p>

  {#if loading}
    <div class="space-y-3">
      {#each { length: 2 } as _}
        <div class="border-border bg-card rounded-lg border p-4">
          <div class="bg-muted h-4 w-40 animate-pulse rounded"></div>
          <div class="mt-3 grid gap-2 sm:grid-cols-3">
            <div class="bg-muted h-3 animate-pulse rounded"></div>
            <div class="bg-muted h-3 animate-pulse rounded"></div>
            <div class="bg-muted h-3 animate-pulse rounded"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else if error}
    <div class="text-destructive text-sm">{error}</div>
  {:else if governance}
    <div class="border-border bg-card rounded-lg border p-4">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div class="space-y-1">
          <div class="text-sm font-medium">Current protection boundary</div>
          <div class="text-muted-foreground text-xs">{governance.stepUp.summary}</div>
        </div>
        <button
          class="border-border bg-background hover:bg-accent hover:text-accent-foreground inline-flex items-center justify-center rounded-md border px-3 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          onclick={handleRevokeOthers}
          disabled={actionKey !== '' ||
            governance.sessions.filter((session) => !session.current).length === 0}
        >
          Revoke other sessions
        </button>
      </div>
    </div>

    <div class="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
      <div class="space-y-3">
        {#if governance.sessions.length > 0}
          {#each governance.sessions as session (session.id)}
            {@const SessionIcon = iconForSession(session)}
            <div class="border-border bg-card rounded-lg border p-4">
              <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div class="flex items-start gap-3">
                  <div
                    class="bg-muted text-muted-foreground flex size-9 items-center justify-center rounded-lg"
                  >
                    <SessionIcon class="size-4" />
                  </div>
                  <div class="space-y-1">
                    <div class="flex flex-wrap items-center gap-2">
                      <div class="text-sm font-medium">{session.device.label}</div>
                      {#if session.current}
                        <span
                          class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-700"
                        >
                          Current
                        </span>
                      {/if}
                    </div>
                    <div class="text-muted-foreground text-xs">
                      {session.device.browser || 'Unknown browser'}
                      {#if session.device.os}
                        on {session.device.os}
                      {/if}
                    </div>
                  </div>
                </div>

                <button
                  class="border-border bg-background hover:bg-accent hover:text-accent-foreground inline-flex items-center justify-center gap-2 rounded-md border px-3 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50"
                  onclick={() => handleSessionAction(session)}
                  disabled={actionKey !== '' &&
                    actionKey !== `session:${session.id}` &&
                    actionKey !== 'session:current'}
                >
                  <LogOut class="size-4" />
                  {session.current ? 'Log out' : 'Revoke'}
                </button>
              </div>

              <div class="mt-4 grid gap-3 text-xs sm:grid-cols-3">
                <div>
                  <div class="text-muted-foreground">Created</div>
                  <div class="mt-1 font-medium">{formatTimestamp(session.createdAt)}</div>
                </div>
                <div>
                  <div class="text-muted-foreground">Last active</div>
                  <div class="mt-1 font-medium">{formatTimestamp(session.lastActiveAt)}</div>
                </div>
                <div>
                  <div class="text-muted-foreground">Expires</div>
                  <div class="mt-1 font-medium">{formatTimestamp(session.idleExpiresAt)}</div>
                </div>
              </div>
            </div>
          {/each}
        {:else}
          <div class="border-border bg-card text-muted-foreground rounded-lg border p-4 text-sm">
            No active browser sessions found.
          </div>
        {/if}
      </div>

      <SecuritySettingsHumanAuthAuditTimeline auditEvents={governance.auditEvents} />
    </div>
  {/if}
</div>
