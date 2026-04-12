<script lang="ts">
  import type { UserDirectoryDetail } from '$lib/api/auth'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { ShieldCheck } from '@lucide/svelte'
  import { formatTimestamp } from './security-settings-human-auth.model'
  import SecuritySettingsUserDirectoryRecentAudit from './security-settings-user-directory-recent-audit.svelte'
  import SecuritySettingsUserDirectorySessions from './security-settings-user-directory-sessions.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

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
      return i18nStore.t('settings.security.userDirectory.groupSummary.none')
    }
    const latest = [...selectedDetail.groups].sort((left, right) =>
      right.lastSyncedAt.localeCompare(left.lastSyncedAt),
    )[0]
    return i18nStore.t('settings.security.userDirectory.groupSummary.details', {
      count: selectedDetail.groups.length,
      lastSync: formatTimestamp(latest?.lastSyncedAt),
    })
  }

  function sessionDiagnostics() {
    if (!selectedDetail) {
      return []
    }
    const diagnostics: string[] = []
    if (selectedDetail.user.status === 'disabled' && selectedDetail.activeSessionCount > 0) {
      diagnostics.push(
        i18nStore.t('settings.security.userDirectory.diagnostics.disabledActiveSessions'),
      )
    }
    if (selectedDetail.activeSessionCount >= 3) {
      diagnostics.push(
        i18nStore.t('settings.security.userDirectory.diagnostics.concurrentSessions'),
      )
    }
    if (!selectedDetail.user.lastLoginAt && selectedDetail.activeSessionCount > 0) {
      diagnostics.push(
        i18nStore.t('settings.security.userDirectory.diagnostics.missingLastLogin'),
      )
    }
    return diagnostics
  }
</script>

{#if error}
  <div class="text-destructive rounded-lg border px-4 py-3 text-sm">{error}</div>
{/if}

{#if !canRead}
  <div class="bg-muted/20 text-muted-foreground rounded-lg border px-4 py-3 text-sm">
    {i18nStore.t('settings.security.userDirectory.messages.accessUnavailable')}
  </div>
{:else if detailLoading}
  <div class="space-y-3">
    <div class="bg-muted h-24 animate-pulse rounded-lg"></div>
    <div class="bg-muted h-48 animate-pulse rounded-lg"></div>
  </div>
{:else if !selectedDetail}
  <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-6 text-sm">
    {i18nStore.t('settings.security.userDirectory.messages.selectUser')}
  </div>
{:else}
  <div class="space-y-4">
    <!-- Profile card -->
    <div class="border-border bg-card space-y-4 rounded-lg border p-4">
      <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <h5 class="text-sm font-semibold">
              {i18nStore.t('settings.security.userDirectory.headings.identityGovernance')}
            </h5>
            <p class="text-muted-foreground text-xs leading-relaxed">
              {i18nStore.t('settings.security.userDirectory.description.identityGovernance')}
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
          <div class="text-muted-foreground text-xs">
            {i18nStore.t('settings.security.userDirectory.labels.displayName')}
          </div>
          <div class="mt-1 truncate text-sm font-medium">
            {selectedDetail.user.displayName ||
              i18nStore.t('settings.security.userDirectory.fallbacks.unnamedUser')}
          </div>
        </div>
        <div>
          <div class="text-muted-foreground text-xs">
            {i18nStore.t('settings.security.userDirectory.labels.primaryEmail')}
          </div>
          <div class="mt-1 truncate text-sm">
            {selectedDetail.user.primaryEmail ||
              i18nStore.t('settings.security.userDirectory.fallbacks.noEmail')}
          </div>
        </div>
        <div>
          <div class="text-muted-foreground text-xs">
            {i18nStore.t('settings.security.userDirectory.labels.lastLogin')}
          </div>
          <div class="mt-1 text-sm">{formatTimestamp(selectedDetail.user.lastLoginAt)}</div>
        </div>
        <div>
          <div class="text-muted-foreground text-xs">
            {i18nStore.t('settings.security.userDirectory.labels.activeSessions')}
          </div>
          <div class="mt-1 text-sm">{selectedDetail.activeSessionCount}</div>
        </div>
      </div>
    </div>

    <!-- Access card: Identities + Groups -->
    <div class="border-border bg-card space-y-3 rounded-lg border p-4">
      <h5 class="text-sm font-semibold">{i18nStore.t('settings.security.userDirectory.headings.access')}</h5>
      <div class="grid gap-4 lg:grid-cols-2">
        <div class="space-y-2">
          <div class="text-muted-foreground text-xs font-medium uppercase">
            {i18nStore.t('settings.security.userDirectory.labels.identities')}
          </div>
          {#each selectedDetail.identities as identity (identity.id)}
            <div class="rounded-lg border p-3 text-xs">
              <div class="font-medium break-all">{identity.issuer}</div>
              <div class="text-muted-foreground mt-1 break-all">
                {i18nStore.t('settings.security.userDirectory.messages.identitySubject', {
                  subject:
                    identity.subject ||
                    i18nStore.t('settings.security.userDirectory.fallbacks.noIdentitySubject'),
                })}
              </div>
              <div class="text-muted-foreground mt-1 break-all">
                {i18nStore.t('settings.security.userDirectory.messages.identityEmail', {
                  email:
                    identity.email ||
                    i18nStore.t('settings.security.userDirectory.fallbacks.noEmail'),
                })}
              </div>
              <div class="text-muted-foreground mt-1">
                {i18nStore.t('settings.security.userDirectory.messages.identitySynced', {
                  syncedAt: formatTimestamp(identity.lastSyncedAt),
                  claimsVersion: identity.claimsVersion,
                })}
              </div>
            </div>
          {/each}
        </div>

        <div class="space-y-2">
          <div class="text-muted-foreground text-xs font-medium uppercase">
            {i18nStore.t('settings.security.userDirectory.labels.oidcGroupCache')}
          </div>
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
              {i18nStore.t('settings.security.userDirectory.messages.noGroups')}
            </div>
          {/if}
          <div class="text-muted-foreground text-[11px] leading-relaxed">
            {i18nStore.t('settings.security.userDirectory.description.groupStrategy')}
          </div>
        </div>
      </div>
    </div>

    <!-- Lifecycle controls card -->
    <div class="border-border bg-card space-y-3 rounded-lg border p-4">
      <div class="flex items-center gap-2">
        <ShieldCheck class="text-muted-foreground size-4" />
        <h5 class="text-sm font-semibold">
          {i18nStore.t('settings.security.userDirectory.headings.lifecycleControls')}
        </h5>
      </div>
      <p class="text-muted-foreground text-xs leading-relaxed">
        {i18nStore.t('settings.security.userDirectory.description.lifecycleControls')}
      </p>

      <label class="flex flex-col gap-1 text-xs">
        <span class="text-muted-foreground">
          {i18nStore.t('settings.security.userDirectory.labels.reason')}
        </span>
        <Input
          value={statusReason}
          placeholder={i18nStore.t('settings.security.userDirectory.placeholders.reason')}
          oninput={(event) => onStatusReasonInput((event.currentTarget as HTMLInputElement).value)}
        />
      </label>

      {#if selectedDetail.latestStatusAudit}
        <div class="text-muted-foreground text-xs">
          {i18nStore.t('settings.security.userDirectory.messages.latestChange', {
            status: selectedDetail.latestStatusAudit.status,
            actor:
              selectedDetail.latestStatusAudit.actorID ||
              i18nStore.t('settings.security.userDirectory.fallbacks.systemActor'),
            changedAt: formatTimestamp(selectedDetail.latestStatusAudit.changedAt),
          })}
        </div>
      {/if}

      <div class="flex flex-wrap gap-2 pt-1">
        <Button
          variant="outline"
          disabled={!canManage || actionKey !== '' || selectedDetail.user.status === 'active'}
          onclick={() => onTransition('active')}
        >
          {i18nStore.t('settings.security.userDirectory.buttons.reEnable')}
        </Button>
        <Button
          variant="destructive"
          disabled={!canManage || actionKey !== '' || selectedDetail.user.status === 'disabled'}
          onclick={() => onTransition('disabled')}
        >
          {i18nStore.t('settings.security.userDirectory.buttons.disableAndRevoke')}
        </Button>
        <Button
          variant="outline"
          disabled={!canManage || actionKey !== ''}
          onclick={onRevokeSessions}
        >
          {i18nStore.t('settings.security.userDirectory.buttons.revokeAll')}
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
        <div class="text-muted-foreground text-xs font-medium uppercase">
          {i18nStore.t('settings.security.userDirectory.labels.diagnostics')}
        </div>
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
            {i18nStore.t('settings.security.userDirectory.messages.noDiagnostics')}
          </div>
        {/if}
      </div>
    </div>

    <SecuritySettingsUserDirectoryRecentAudit events={selectedDetail.recentAuditEvents} />

    <div
      class="text-muted-foreground rounded-lg border border-dashed px-4 py-3 text-[11px] leading-relaxed"
    >
      {i18nStore.t('settings.security.userDirectory.messages.unsupportedToday')}
    </div>
  </div>
{/if}
