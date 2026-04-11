<script lang="ts">
  import type { ManagedAuthSession } from '$lib/api/auth'
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
    <div class="text-sm font-medium">Active browser sessions</div>
    <div class="text-muted-foreground text-xs leading-relaxed">
      Clear distinction: these are governable user sessions, not just the current browser.
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
                  <Badge variant="secondary">Current admin session</Badge>
                {/if}
              </div>
              <div class="text-muted-foreground">
                {session.device.browser || 'Unknown browser'}
                {#if session.device.os}
                  on {session.device.os}
                {/if}
                {#if session.ipSummary}
                  · IP {session.ipSummary}
                {/if}
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              disabled={!canManage || (actionKey !== '' && actionKey !== `revoke:${session.id}`)}
              onclick={() => onRevokeSession(session.id)}
            >
              Revoke session
            </Button>
          </div>
          <div class="text-muted-foreground mt-3 grid gap-2 sm:grid-cols-3">
            <div>Created {formatTimestamp(session.createdAt)}</div>
            <div>Last active {formatTimestamp(session.lastActiveAt)}</div>
            <div>Idle expiry {formatTimestamp(session.idleExpiresAt)}</div>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-xs">
      No active browser sessions for this user.
    </div>
  {/if}
</div>
