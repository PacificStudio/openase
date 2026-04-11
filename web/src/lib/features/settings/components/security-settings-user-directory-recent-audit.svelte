<script lang="ts">
  import type { UserDirectoryDetail } from '$lib/api/auth'
  import * as Collapsible from '$ui/collapsible'
  import { ChevronDown } from '@lucide/svelte'
  import {
    authAuditEventDotClass,
    formatAuthAuditEventLabel,
    formatAuthAuditEventSeverity,
    formatTimestamp,
  } from './security-settings-human-auth.model'

  let { events }: { events: UserDirectoryDetail['recentAuditEvents'] } = $props()
  let open = $state(false)
</script>

<div class="border-border bg-card overflow-hidden rounded-lg border">
  <Collapsible.Root bind:open>
    <Collapsible.Trigger>
      {#snippet child({ props })}
        <button
          {...props}
          type="button"
          class="hover:bg-muted/40 flex w-full items-center justify-between gap-3 px-4 py-3 text-left transition-colors"
        >
          <div class="flex min-w-0 items-center gap-2">
            <span class="text-sm font-semibold">Recent auth audit</span>
            {#if events.length > 0}
              <span class="text-muted-foreground text-xs font-normal">{events.length}</span>
            {/if}
          </div>
          <ChevronDown
            class="text-muted-foreground size-4 shrink-0 transition-transform {open
              ? 'rotate-180'
              : ''}"
          />
        </button>
      {/snippet}
    </Collapsible.Trigger>
    <Collapsible.Content>
      <div class="border-border/60 border-t">
        {#if events.length > 0}
          <ul class="divide-border/40 max-h-72 divide-y overflow-y-auto">
            {#each events as event (event.id)}
              {@const dotClass =
                authAuditEventDotClass[formatAuthAuditEventSeverity(event.eventType)]}
              <li
                class="grid grid-cols-[auto_minmax(0,7.5rem)_minmax(0,9rem)_minmax(0,1fr)] items-center gap-x-3 px-4 py-1.5 text-[11px]"
              >
                <span class="size-1.5 rounded-full {dotClass}"></span>
                <span class="text-foreground truncate font-medium">
                  {formatAuthAuditEventLabel(event.eventType)}
                </span>
                <span class="text-muted-foreground truncate tabular-nums">
                  {formatTimestamp(event.createdAt)}
                </span>
                <span class="text-muted-foreground truncate">{event.message}</span>
              </li>
            {/each}
          </ul>
        {:else}
          <div class="text-muted-foreground px-4 py-3 text-xs">
            No recent auth audit events for this user.
          </div>
        {/if}
      </div>
    </Collapsible.Content>
  </Collapsible.Root>
</div>
