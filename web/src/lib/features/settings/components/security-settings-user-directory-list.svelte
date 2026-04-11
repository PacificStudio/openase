<script lang="ts">
  import type { UserDirectoryEntry } from '$lib/api/auth'
  import { Badge } from '$ui/badge'
  import { formatTimestamp } from './security-settings-human-auth.model'

  let {
    canRead = false,
    loading = false,
    users = [],
    selectedUserId = '',
    onSelect = (_userId: string) => {},
  }: {
    canRead?: boolean
    loading?: boolean
    users?: UserDirectoryEntry[]
    selectedUserId?: string
    onSelect?: (userId: string) => void
  } = $props()

  function statusVariant(status: string) {
    return status === 'disabled' ? 'destructive' : 'secondary'
  }
</script>

{#if !canRead}
  <div class="bg-muted/20 text-muted-foreground rounded-lg border px-4 py-3 text-sm">
    Instance-level <code>security_setting.read</code> is required to browse the user directory.
  </div>
{:else if loading}
  <div class="border-border divide-border/60 divide-y overflow-hidden rounded-lg border">
    {#each { length: 4 } as _}
      <div class="px-4 py-3">
        <div class="bg-muted h-3 w-40 animate-pulse rounded"></div>
        <div class="bg-muted mt-2 h-3 w-56 animate-pulse rounded"></div>
      </div>
    {/each}
  </div>
{:else if users.length === 0}
  <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-6 text-sm">
    No users match the current search and filter combination.
  </div>
{:else}
  <div class="border-border overflow-hidden rounded-lg border">
    <div class="divide-border/60 max-h-[22rem] divide-y overflow-y-auto">
      {#each users as entry (entry.id)}
        <button
          type="button"
          class={`flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors ${
            entry.id === selectedUserId ? 'bg-primary/5' : 'hover:bg-muted/40'
          }`}
          onclick={() => onSelect(entry.id)}
        >
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="truncate text-sm font-medium">
                {entry.displayName || entry.primaryEmail || entry.id}
              </span>
              <Badge variant={statusVariant(entry.status)} class="shrink-0 px-1.5 py-0 text-[10px]">
                {entry.status}
              </Badge>
            </div>
            <div class="text-muted-foreground truncate text-xs">
              {entry.primaryEmail || entry.id}
            </div>
          </div>
          <div class="text-muted-foreground hidden shrink-0 text-right text-[11px] md:block">
            <div class="tabular-nums">{formatTimestamp(entry.lastLoginAt)}</div>
            <div class="max-w-[18rem] truncate">
              {entry.primaryIdentity?.issuer || 'No identity cached'}
            </div>
          </div>
        </button>
      {/each}
    </div>
  </div>
{/if}
