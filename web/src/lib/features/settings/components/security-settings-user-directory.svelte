<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { UserDirectoryDetail, UserDirectoryEntry } from '$lib/api/auth'
  import {
    adminRevokeUserAuthSessions,
    getInstanceUserDetail,
    listInstanceUsers,
    transitionInstanceUserStatus,
  } from '$lib/api/auth'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { ShieldCheck, Users } from '@lucide/svelte'
  import { formatAuthAuditEventLabel, formatTimestamp } from './security-settings-human-auth.model'

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

  const selectedUser = $derived(users.find((entry) => entry.id === selectedUserId) ?? null)

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
          Instance-level <code>security.read</code> is required to browse the user directory.
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

    <div class="border-border bg-card space-y-4 rounded-lg border p-4">
      <div class="flex items-center justify-between gap-3">
        <div>
          <div class="text-sm font-semibold">Identity governance detail</div>
          <div class="text-muted-foreground text-xs">
            Issuer + subject stay canonical; mutable profile fields continue to sync onto the cached
            user.
          </div>
        </div>
        {#if selectedUser}
          <Badge variant={statusVariant(selectedUser.status)}>{selectedUser.status}</Badge>
        {/if}
      </div>

      {#if error}
        <div class="text-destructive text-sm">{error}</div>
      {/if}

      {#if !canRead}
        <div class="bg-muted/20 text-muted-foreground rounded-lg border px-4 py-3 text-sm">
          Browse access is unavailable for the current principal.
        </div>
      {:else if detailLoading}
        <div class="space-y-3">
          <div class="bg-muted h-20 animate-pulse rounded-lg"></div>
          <div class="bg-muted h-48 animate-pulse rounded-lg"></div>
        </div>
      {:else if !selectedDetail}
        <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-6 text-sm">
          Select a user to inspect identities, groups, audit history, and lifecycle controls.
        </div>
      {:else}
        <div class="space-y-4">
          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <div class="text-muted-foreground text-xs">Display name</div>
              <div class="mt-1 text-sm font-medium">
                {selectedDetail.user.displayName || 'Unnamed user'}
              </div>
            </div>
            <div>
              <div class="text-muted-foreground text-xs">Primary email</div>
              <div class="mt-1 text-sm">{selectedDetail.user.primaryEmail || 'No email'}</div>
            </div>
            <div>
              <div class="text-muted-foreground text-xs">Last login</div>
              <div class="mt-1 text-sm">{formatTimestamp(selectedDetail.user.lastLoginAt)}</div>
            </div>
            <div>
              <div class="text-muted-foreground text-xs">Active sessions</div>
              <div class="mt-1 text-sm">{selectedDetail.activeSessionCount}</div>
            </div>
          </div>

          <div class="grid gap-4 lg:grid-cols-2">
            <div class="space-y-2">
              <div class="text-sm font-medium">Identities</div>
              {#each selectedDetail.identities as identity (identity.id)}
                <div class="rounded-lg border p-3 text-xs">
                  <div class="font-medium">{identity.issuer}</div>
                  <div class="text-muted-foreground mt-1 break-all">
                    subject: {identity.subject}
                  </div>
                  <div class="text-muted-foreground mt-1">email: {identity.email || 'none'}</div>
                  <div class="text-muted-foreground mt-1">
                    synced: {formatTimestamp(identity.lastSyncedAt)} · claims v{identity.claimsVersion}
                  </div>
                </div>
              {/each}
            </div>

            <div class="space-y-2">
              <div class="text-sm font-medium">OIDC group cache</div>
              {#if selectedDetail.groups.length > 0}
                <div class="flex flex-wrap gap-2">
                  {#each selectedDetail.groups as group (group.id)}
                    <code class="bg-muted rounded px-2 py-1 text-xs">
                      {group.groupName || group.groupKey}
                    </code>
                  {/each}
                </div>
              {:else}
                <div
                  class="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-xs"
                >
                  No synchronized groups for this user.
                </div>
              {/if}
              <div class="text-muted-foreground text-xs">
                Current strategy stays cache-only: OpenASE stores synchronized OIDC groups for RBAC
                evaluation and does not ship a separate local group catalog yet.
              </div>
            </div>
          </div>

          <div class="rounded-lg border p-4">
            <div class="flex items-center gap-2 text-sm font-medium">
              <ShieldCheck class="text-muted-foreground size-4" />
              Lifecycle controls
            </div>
            <div class="text-muted-foreground mt-1 text-xs">
              Manual admin disable is supported now. Automatic upstream sync, webhook, and
              SCIM-driven deprovision hooks stay reserved for follow-up work.
            </div>

            <label class="mt-3 flex flex-col gap-1 text-xs">
              <span class="text-muted-foreground">Reason</span>
              <Input
                bind:value={statusReason}
                placeholder="Document the lifecycle reason for audit and future review"
              />
            </label>

            {#if selectedDetail.latestStatusAudit}
              <div class="text-muted-foreground mt-3 text-xs">
                Latest change: {selectedDetail.latestStatusAudit.status} by {selectedDetail
                  .latestStatusAudit.actorID || 'system'} at
                {formatTimestamp(selectedDetail.latestStatusAudit.changedAt)}.
              </div>
            {/if}

            <div class="mt-4 flex flex-wrap gap-2">
              <Button
                variant="outline"
                disabled={!canManage || actionKey !== '' || selectedDetail.user.status === 'active'}
                onclick={() => void handleTransition('active')}
              >
                Re-enable user
              </Button>
              <Button
                variant="destructive"
                disabled={!canManage ||
                  actionKey !== '' ||
                  selectedDetail.user.status === 'disabled'}
                onclick={() => void handleTransition('disabled')}
              >
                Disable and revoke sessions
              </Button>
              <Button
                variant="outline"
                disabled={!canManage || actionKey !== ''}
                onclick={() => void handleRevokeSessions()}
              >
                Revoke sessions only
              </Button>
            </div>
          </div>

          <div class="space-y-2">
            <div class="text-sm font-medium">Recent auth audit</div>
            {#if selectedDetail.recentAuditEvents.length > 0}
              <div class="space-y-2">
                {#each selectedDetail.recentAuditEvents as event (event.id)}
                  <div class="rounded-lg border p-3 text-xs">
                    <div class="flex items-start justify-between gap-3">
                      <div class="font-medium">{formatAuthAuditEventLabel(event.eventType)}</div>
                      <div class="text-muted-foreground">{formatTimestamp(event.createdAt)}</div>
                    </div>
                    <div class="text-muted-foreground mt-1">{event.message}</div>
                  </div>
                {/each}
              </div>
            {:else}
              <div class="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-xs">
                No recent auth audit events for this user.
              </div>
            {/if}
          </div>

          <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-3 text-xs">
            Unsupported today: multiple upstream identities linked to one user, manual link or
            unlink, and automatic merge across matching emails. Those cases currently fail closed
            instead of guessing.
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>
