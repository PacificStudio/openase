<script lang="ts">
  import type { AuthAuditEvent } from '$lib/api/auth'
  import { Clock3 } from '@lucide/svelte'
  import { formatAuthAuditEventLabel, formatTimestamp } from './security-settings-human-auth.model'

  let { auditEvents }: { auditEvents: AuthAuditEvent[] } = $props()
</script>

<div class="border-border bg-card rounded-lg border p-4">
  <div class="flex items-center gap-2">
    <Clock3 class="text-muted-foreground size-4" />
    <h5 class="text-sm font-semibold">Auth audit timeline</h5>
  </div>

  {#if auditEvents.length > 0}
    <div class="mt-4 space-y-4">
      {#each auditEvents as event (event.id)}
        <div class="border-border border-l-2 pl-3">
          <div class="text-sm font-medium">{formatAuthAuditEventLabel(event.eventType)}</div>
          <div class="text-muted-foreground mt-1 text-xs">{event.message}</div>
          <div class="text-muted-foreground mt-1 text-[11px]">
            {formatTimestamp(event.createdAt)}
            {#if event.actorID}
              · {event.actorID}
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="text-muted-foreground mt-4 text-sm">
      Auth audit events appear here after sign-in, logout, expiry, and revoke actions.
    </div>
  {/if}
</div>
