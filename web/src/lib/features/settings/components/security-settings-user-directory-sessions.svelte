<script lang="ts">
  import type { ManagedAuthSession } from '$lib/api/auth'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { formatTimestamp } from './security-settings-human-auth.model'

  let {
    activeSessions = [],
    canManage = false,
    actionKey = '',
    onRevokeSession = (_sessionId: string) => {},
  }: {
    activeSessions?: ManagedAuthSession[]
    canManage?: boolean
    actionKey?: string
    onRevokeSession?: (sessionId: string) => void
  } = $props()
</script>

<div class="space-y-2">
  <div class="space-y-0.5">
    <div class="text-sm font-medium">
      {i18nStore.t('settings.security.userDirectory.sessions.title')}
    </div>
    <div class="text-muted-foreground text-xs leading-relaxed">
      {i18nStore.t('settings.security.userDirectory.sessions.description')}
    </div>
  </div>
  {#if activeSessions.length > 0}
    <div class="space-y-2">
      {#each activeSessions as session (session.id)}
        <div class="rounded-lg border p-3 text-xs">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div class="space-y-1">
              <div class="flex flex-wrap items-center gap-2">
                <div class="font-medium">{session.device.label}</div>
                {#if session.current}
                  <Badge variant="secondary">
                    {i18nStore.t('settings.security.userDirectory.sessions.currentAdmin')}
                  </Badge>
                {/if}
              </div>
              <div class="text-muted-foreground">
                {session.device.browser ||
                  i18nStore.t('settings.security.userDirectory.sessions.unknownBrowser')}
                {#if session.device.os}
                  {i18nStore.t('settings.security.userDirectory.sessions.onOs', {
                    os: session.device.os,
                  })}
                {/if}
                {#if session.ipSummary}
                  ·{' '}
                  {i18nStore.t('settings.security.userDirectory.sessions.ip', {
                    ip: session.ipSummary,
                  })}
                {/if}
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              disabled={!canManage || (actionKey !== '' && actionKey !== `revoke:${session.id}`)}
              onclick={() => onRevokeSession(session.id)}
            >
              {i18nStore.t('settings.security.userDirectory.sessions.actions.revoke')}
            </Button>
          </div>
          <div class="text-muted-foreground mt-3 grid gap-2 sm:grid-cols-3">
            <div>
              {i18nStore.t('settings.security.userDirectory.sessions.labels.created', {
                time: formatTimestamp(session.createdAt),
              })}
            </div>
            <div>
              {i18nStore.t('settings.security.userDirectory.sessions.labels.lastActive', {
                time: formatTimestamp(session.lastActiveAt),
              })}
            </div>
            <div>
              {i18nStore.t('settings.security.userDirectory.sessions.labels.idleExpiry', {
                time: formatTimestamp(session.idleExpiresAt),
              })}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-xs">
      {i18nStore.t('settings.security.userDirectory.sessions.messages.noSessions')}
    </div>
  {/if}
</div>
