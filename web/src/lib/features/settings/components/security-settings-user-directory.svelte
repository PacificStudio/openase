<script lang="ts">
  import { goto } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import type { UserDirectoryDetail, UserDirectoryEntry } from '$lib/api/auth'
  import {
    adminRevokeAuthSession,
    adminRevokeUserAuthSessions,
    getInstanceUserDetail,
    listInstanceUsers,
    normalizeReturnTo,
    transitionInstanceUserStatus,
  } from '$lib/api/auth'
  import { authStore } from '$lib/stores/auth.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import { Input } from '$ui/input'
  import { Users } from '@lucide/svelte'
  import { formatTimestamp } from './security-settings-human-auth.model'
  import SecuritySettingsUserDirectoryDetail from './security-settings-user-directory-detail.svelte'

  let {
    canRead = false,
    canManage = false,
  }: {
    canRead?: boolean
    canManage?: boolean
  } = $props()
  let loading = $state(false)
  let detailLoading = $state(false)
  let error = $state('')
  let actionKey = $state('')
  let searchQuery = $state('')
  let statusFilter = $state<'all' | 'active' | 'disabled'>('all')
  let users = $state<UserDirectoryEntry[]>([])
  let selectedUserId = $state('')
  let selectedDetail = $state<UserDirectoryDetail | null>(null)
  let statusReason = $state('')

  const selectedUserStatus = $derived(
    users.find((entry) => entry.id === selectedUserId)?.status ?? '',
  )
  function formatError(caughtError: unknown, fallback: string) {
    return caughtError instanceof ApiError ? caughtError.detail : fallback
  }

  async function loadUsers() {
    if (!canRead) {
      users = []
      selectedUserId = ''
      selectedDetail = null
      error = ''
      return
    }

    loading = true
    error = ''
    try {
      const nextUsers = await listInstanceUsers({
        query: searchQuery.trim() || undefined,
        status: statusFilter,
        limit: 50,
      })
      users = nextUsers
      if (!nextUsers.some((entry) => entry.id === selectedUserId)) {
        selectedUserId = nextUsers[0]?.id ?? ''
      }
    } catch (caughtError) {
      users = []
      selectedUserId = ''
      selectedDetail = null
      error = formatError(caughtError, 'Failed to load the user directory.')
    } finally {
      loading = false
    }
  }
  async function loadDetail(userId: string) {
    if (!canRead || !userId) {
      selectedDetail = null
      return
    }
    detailLoading = true
    error = ''
    try {
      selectedDetail = await getInstanceUserDetail(userId)
      statusReason = selectedDetail.latestStatusAudit?.reason ?? ''
    } catch (caughtError) {
      selectedDetail = null
      error = formatError(caughtError, 'Failed to load user detail.')
    } finally {
      detailLoading = false
    }
  }
  async function handleTransition(status: 'active' | 'disabled') {
    if (!selectedDetail || !canManage) {
      return
    }
    if (statusReason.trim() === '') {
      toastStore.error('Reason is required for user status changes.')
      return
    }

    actionKey = status
    error = ''
    try {
      const result = await transitionInstanceUserStatus(selectedDetail.user.id, {
        status,
        reason: statusReason.trim(),
      })
      toastStore.success(
        status === 'disabled'
          ? `Disabled user and revoked ${result.revokedSessionCount} session(s).`
          : 'User enabled.',
      )
      statusReason = result.latestStatusAudit?.reason ?? statusReason
      await loadUsers()
      await loadDetail(result.user.id)
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to change user status.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }
  async function handleRevokeSessions() {
    if (!selectedDetail || !canManage) {
      return
    }
    actionKey = 'revoke'
    error = ''
    try {
      const result = await adminRevokeUserAuthSessions(selectedDetail.user.id)
      if (result.current_session_revoked) {
        authStore.clear()
        await goto(
          `/login?return_to=${encodeURIComponent(normalizeReturnTo(window.location.pathname + window.location.search + window.location.hash))}`,
        )
        return
      }
      toastStore.success(`Revoked ${result.revoked_count} browser session(s).`)
      await loadDetail(selectedDetail.user.id)
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to revoke user sessions.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }
  async function handleRevokeSession(sessionId: string) {
    if (!selectedDetail || !canManage) {
      return
    }
    actionKey = `revoke:${sessionId}`
    error = ''
    try {
      const result = await adminRevokeAuthSession(sessionId)
      if (result.current_session_revoked) {
        authStore.clear()
        await goto(
          `/login?return_to=${encodeURIComponent(normalizeReturnTo(window.location.pathname + window.location.search + window.location.hash))}`,
        )
        return
      }
      toastStore.success('Session revoked.')
      await loadDetail(selectedDetail.user.id)
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to revoke session.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }

  function statusVariant(status: string) {
    return status === 'disabled' ? 'destructive' : 'secondary'
  }

  $effect(() => {
    if (!canRead) {
      users = []
      selectedUserId = ''
      selectedDetail = null
      error = ''
      return
    }
    const timer = window.setTimeout(() => {
      void loadUsers()
    }, 150)
    return () => {
      window.clearTimeout(timer)
    }
  })

  $effect(() => {
    if (!selectedUserId) {
      selectedDetail = null
      return
    }
    void loadDetail(selectedUserId)
  })
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <Users class="text-muted-foreground size-4" />
    <h4 class="text-sm font-semibold">User directory and deprovision</h4>
  </div>

  <div class="grid gap-4 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.25fr)]">
    <div class="border-border bg-card space-y-4 rounded-lg border p-4">
      <div class="flex flex-col gap-3 sm:flex-row">
        <div class="relative flex-1">
          <Input
            bind:value={searchQuery}
            placeholder="Search email, display name, issuer, or subject"
          />
        </div>
        <label class="flex min-w-[10rem] flex-col gap-1 text-xs">
          <span class="text-muted-foreground">Status</span>
          <select
            bind:value={statusFilter}
            class="border-input bg-background ring-offset-background focus-visible:ring-ring h-10 rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
          >
            <option value="all">All statuses</option>
            <option value="active">Active</option>
            <option value="disabled">Disabled</option>
          </select>
        </label>
      </div>

      {#if !canRead}
        <div class="bg-muted/20 text-muted-foreground rounded-lg border px-4 py-3 text-sm">
          Instance-level <code>security_setting.read</code> is required to browse the user directory.
        </div>
      {:else if loading}
        <div class="space-y-3">
          {#each { length: 4 } as _}
            <div class="bg-muted h-16 animate-pulse rounded-lg"></div>
          {/each}
        </div>
      {:else if users.length === 0}
        <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-6 text-sm">
          No users match the current search and filter combination.
        </div>
      {:else}
        <div class="space-y-2">
          {#each users as entry (entry.id)}
            <button
              type="button"
              class={`border-border w-full rounded-lg border p-3 text-left transition-colors ${
                entry.id === selectedUserId ? 'bg-muted/60' : 'hover:bg-muted/40'
              }`}
              onclick={() => {
                selectedUserId = entry.id
              }}
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="truncate text-sm font-medium">
                    {entry.displayName || entry.primaryEmail || entry.id}
                  </div>
                  <div class="text-muted-foreground truncate text-xs">
                    {entry.primaryEmail || entry.id}
                  </div>
                </div>
                <Badge variant={statusVariant(entry.status)}>{entry.status}</Badge>
              </div>
              <div class="text-muted-foreground mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs">
                <span>Last login: {formatTimestamp(entry.lastLoginAt)}</span>
                <span>Issuer: {entry.primaryIdentity?.issuer || 'No identity cached'}</span>
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <SecuritySettingsUserDirectoryDetail
      {canRead}
      {canManage}
      {detailLoading}
      {selectedDetail}
      {error}
      {actionKey}
      {statusReason}
      {selectedUserStatus}
      onStatusReasonInput={(value) => {
        statusReason = value
      }}
      onTransition={(status) => void handleTransition(status)}
      onRevokeSessions={() => void handleRevokeSessions()}
      onRevokeSession={(sessionId) => void handleRevokeSession(sessionId)}
    />
  </div>
</div>
