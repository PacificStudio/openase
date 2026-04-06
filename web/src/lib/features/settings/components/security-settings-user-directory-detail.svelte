<script lang="ts">
  import type { UserDirectoryDetail } from '$lib/api/auth'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { ShieldCheck } from '@lucide/svelte'
  import { formatAuthAuditEventLabel, formatTimestamp } from './security-settings-human-auth.model'

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
  } = $props()

  function statusVariant(status: string) {
    return status === 'disabled' ? 'destructive' : 'secondary'
  }
</script>

<div class="border-border bg-card space-y-4 rounded-lg border p-4">
  <div class="flex items-center justify-between gap-3">
    <div>
      <div class="text-sm font-semibold">Identity governance detail</div>
      <div class="text-muted-foreground text-xs">
        Issuer + subject stay canonical; mutable profile fields continue to sync onto the cached
        user.
      </div>
    </div>
    {#if selectedUserStatus}
      <Badge variant={statusVariant(selectedUserStatus)}>{selectedUserStatus}</Badge>
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
              <div class="text-muted-foreground mt-1 break-all">subject: {identity.subject}</div>
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
            <div class="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-xs">
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
          Manual admin disable is supported now. Automatic upstream sync, webhook, and SCIM-driven
          deprovision hooks stay reserved for follow-up work.
        </div>

        <label class="mt-3 flex flex-col gap-1 text-xs">
          <span class="text-muted-foreground">Reason</span>
          <Input
            value={statusReason}
            placeholder="Document the lifecycle reason for audit and future review"
            oninput={(event) =>
              onStatusReasonInput((event.currentTarget as HTMLInputElement).value)}
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
        Unsupported today: multiple upstream identities linked to one user, manual link or unlink,
        and automatic merge across matching emails. Those cases currently fail closed instead of
        guessing.
      </div>
    </div>
  {/if}
</div>
