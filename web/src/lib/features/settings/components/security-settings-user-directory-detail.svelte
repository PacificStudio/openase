<script lang="ts">
  import type { UserDirectoryDetail } from '$lib/api/auth'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { ShieldCheck } from '@lucide/svelte'
  import { formatTimestamp } from './security-settings-human-auth.model'
  import SecuritySettingsUserDirectoryRecentAudit from './security-settings-user-directory-recent-audit.svelte'
  import SecuritySettingsUserDirectorySessions from './security-settings-user-directory-sessions.svelte'

  let {
    canRead = false,
    canManage = false,
    detailLoading = false,
    selectedDetail = null,
    selectedUserStatus = '',
    error = '',
    actionKey = '',
    statusReason = '',
    onStatusReasonInput = (_value: string) => {},
    onTransition = (_status: 'active' | 'disabled') => {},
    onRevokeSessions = () => {},
    onRevokeSession = (_sessionId: string) => {},
  }: {
    canRead?: boolean
    canManage?: boolean
    detailLoading?: boolean
    selectedDetail?: UserDirectoryDetail | null
    selectedUserStatus?: string
    error?: string
    actionKey?: string
    statusReason?: string
    onStatusReasonInput?: (value: string) => void
    onTransition?: (status: 'active' | 'disabled') => void
    onRevokeSessions?: () => void
    onRevokeSession?: (sessionId: string) => void
  } = $props()

  function statusVariant(status: string) {
    return status === 'disabled' ? 'destructive' : 'secondary'
  }

  function groupSyncSummary() {
    if (!selectedDetail || selectedDetail.groups.length === 0) {
      return 'No synchronized groups cached.'
    }
    const latest = [...selectedDetail.groups].sort((left, right) =>
      right.lastSyncedAt.localeCompare(left.lastSyncedAt),
    )[0]
    return `${selectedDetail.groups.length} synchronized group(s) · last sync ${formatTimestamp(latest?.lastSyncedAt)}`
  }

  function sessionDiagnostics() {
    if (!selectedDetail) {
      return []
    }
    const diagnostics: string[] = []
    if (selectedDetail.user.status === 'disabled' && selectedDetail.activeSessionCount > 0) {
      diagnostics.push('Disabled user still has active sessions and should be investigated.')
    }
    if (selectedDetail.activeSessionCount >= 3) {
      diagnostics.push(
        'Multiple concurrent active sessions may indicate shared-device or stale-session risk.',
      )
    }
    if (!selectedDetail.user.lastLoginAt && selectedDetail.activeSessionCount > 0) {
      diagnostics.push('Active sessions exist without a recorded last-login timestamp.')
    }
    return diagnostics
  }
</script>

{#if error}
  <div class="text-destructive rounded-lg border px-4 py-3 text-sm">{error}</div>
{/if}

{#if !canRead}
  <div class="bg-muted/20 text-muted-foreground rounded-lg border px-4 py-3 text-sm">
    Browse access is unavailable for the current principal.
  </div>
{:else if detailLoading}
  <div class="space-y-3">
    <div class="bg-muted h-24 animate-pulse rounded-lg"></div>
    <div class="bg-muted h-48 animate-pulse rounded-lg"></div>
  </div>
{:else if !selectedDetail}
  <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-6 text-sm">
    Select a user to inspect identities, groups, audit history, and lifecycle controls.
  </div>
{:else}
  <div class="space-y-4">
    <!-- Profile card -->
    <div class="border-border bg-card space-y-4 rounded-lg border p-4">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <h5 class="text-sm font-semibold">Identity governance detail</h5>
          <p class="text-muted-foreground text-xs leading-relaxed">
            Issuer + subject stay canonical; mutable profile fields continue to sync onto the cached
            user.
          </p>
        </div>
        {#if selectedUserStatus}
          <Badge variant={statusVariant(selectedUserStatus)} class="shrink-0">
            {selectedUserStatus}
          </Badge>
        {/if}
      </div>

      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <div class="text-muted-foreground text-xs">Display name</div>
          <div class="mt-1 truncate text-sm font-medium">
            {selectedDetail.user.displayName || 'Unnamed user'}
          </div>
        </div>
        <div>
          <div class="text-muted-foreground text-xs">Primary email</div>
          <div class="mt-1 truncate text-sm">
            {selectedDetail.user.primaryEmail || 'No email'}
          </div>
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
    </div>

    <!-- Access card: Identities + Groups -->
    <div class="border-border bg-card space-y-3 rounded-lg border p-4">
      <h5 class="text-sm font-semibold">Access</h5>
      <div class="grid gap-4 lg:grid-cols-2">
        <div class="space-y-2">
          <div class="text-muted-foreground text-xs font-medium uppercase">Identities</div>
          {#each selectedDetail.identities as identity (identity.id)}
            <div class="rounded-lg border p-3 text-xs">
              <div class="font-medium break-all">{identity.issuer}</div>
              <div class="text-muted-foreground mt-1 break-all">subject: {identity.subject}</div>
              <div class="text-muted-foreground mt-1 break-all">
                email: {identity.email || 'none'}
              </div>
              <div class="text-muted-foreground mt-1">
                synced: {formatTimestamp(identity.lastSyncedAt)} · claims v{identity.claimsVersion}
              </div>
            </div>
          {/each}
        </div>

        <div class="space-y-2">
          <div class="text-muted-foreground text-xs font-medium uppercase">OIDC group cache</div>
          <div class="text-muted-foreground text-xs">{groupSyncSummary()}</div>
          {#if selectedDetail.groups.length > 0}
            <div class="flex flex-wrap gap-2">
              {#each selectedDetail.groups as group (group.id)}
                <code class="bg-muted rounded px-2 py-1 text-xs">
                  {group.groupName || group.groupKey}
                </code>
              {/each}
            </div>
          {:else}
            <div class="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-xs">
              No synchronized groups for this user.
            </div>
          {/if}
          <div class="text-muted-foreground text-[11px] leading-relaxed">
            Current strategy stays cache-only: OpenASE stores synchronized OIDC groups for RBAC
            evaluation and does not ship a separate local group catalog yet.
          </div>
        </div>
      </div>
    </div>

    <!-- Lifecycle controls card -->
    <div class="border-border bg-card space-y-3 rounded-lg border p-4">
      <div class="flex items-center gap-2">
        <ShieldCheck class="text-muted-foreground size-4" />
        <h5 class="text-sm font-semibold">Lifecycle controls</h5>
      </div>
      <p class="text-muted-foreground text-xs leading-relaxed">
        Manual admin disable is supported now. Automatic upstream sync, webhook, and SCIM-driven
        deprovision hooks stay reserved for follow-up work.
      </p>

      <label class="flex flex-col gap-1 text-xs">
        <span class="text-muted-foreground">Reason</span>
        <Input
          value={statusReason}
          placeholder="Document the lifecycle reason for audit and future review"
          oninput={(event) => onStatusReasonInput((event.currentTarget as HTMLInputElement).value)}
        />
      </label>

      {#if selectedDetail.latestStatusAudit}
        <div class="text-muted-foreground text-xs">
          Latest change: {selectedDetail.latestStatusAudit.status} by {selectedDetail
            .latestStatusAudit.actorID || 'system'} at
          {formatTimestamp(selectedDetail.latestStatusAudit.changedAt)}.
        </div>
      {/if}

      <div class="flex flex-wrap gap-2 pt-1">
        <Button
          variant="outline"
          disabled={!canManage || actionKey !== '' || selectedDetail.user.status === 'active'}
          onclick={() => onTransition('active')}
        >
          Re-enable user
        </Button>
        <Button
          variant="destructive"
          disabled={!canManage || actionKey !== '' || selectedDetail.user.status === 'disabled'}
          onclick={() => onTransition('disabled')}
        >
          Disable and revoke sessions
        </Button>
        <Button
          variant="outline"
          disabled={!canManage || actionKey !== ''}
          onclick={onRevokeSessions}
        >
          Revoke all sessions
        </Button>
      </div>
    </div>

    <!-- Session activity card: Active sessions + Diagnostics -->
    <div class="border-border bg-card space-y-4 rounded-lg border p-4">
      <SecuritySettingsUserDirectorySessions
        activeSessions={selectedDetail.activeSessions}
        {canManage}
        {actionKey}
        {onRevokeSession}
      />

      <div class="space-y-2">
        <div class="text-muted-foreground text-xs font-medium uppercase">Diagnostics</div>
        {#if sessionDiagnostics().length > 0}
          <div class="space-y-2">
            {#each sessionDiagnostics() as diagnostic (diagnostic)}
              <div class="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs">
                {diagnostic}
              </div>
            {/each}
          </div>
        {:else}
          <div class="text-muted-foreground rounded-lg border border-dashed px-3 py-3 text-xs">
            No immediate session anomalies detected from the cached user and active-session state.
          </div>
        {/if}
      </div>
    </div>

    <SecuritySettingsUserDirectoryRecentAudit events={selectedDetail.recentAuditEvents} />

    <div
      class="text-muted-foreground rounded-lg border border-dashed px-4 py-3 text-[11px] leading-relaxed"
    >
      Unsupported today: multiple upstream identities linked to one user, manual link or unlink, and
      automatic merge across matching emails. Those cases currently fail closed instead of guessing.
    </div>
  </div>
{/if}
