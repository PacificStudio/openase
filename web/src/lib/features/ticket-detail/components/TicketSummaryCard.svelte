<script lang="ts">
  import { LoaderCircle } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import { formatTimestamp, ticketPriorityBadgeClass, type Project } from '$lib/features/workspace'
  import type { TicketDetailPayload } from '../types'

  let {
    detail,
    project = null,
    projectId = '',
    refreshing = false,
  }: {
    detail: TicketDetailPayload
    project?: Project | null
    projectId?: string
    refreshing?: boolean
  } = $props()
</script>

<Card class="border-border/80 bg-background/85 backdrop-blur">
  <CardHeader class="border-border/60 gap-4 border-b pb-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-3">
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="outline">{detail.ticket.identifier}</Badge>
          <Badge variant="outline">{project?.name ?? projectId}</Badge>
          <span
            class={`inline-flex rounded-full border px-2.5 py-1 text-[11px] font-medium ${ticketPriorityBadgeClass(detail.ticket.priority)}`}
          >
            {detail.ticket.priority}
          </span>
        </div>
        <div>
          <CardTitle class="text-foreground text-3xl tracking-[-0.04em]">
            {detail.ticket.title}
          </CardTitle>
          <CardDescription class="text-muted-foreground mt-2 max-w-3xl text-sm leading-6">
            {detail.ticket.description || 'No description yet.'}
          </CardDescription>
        </div>
      </div>

      {#if refreshing}
        <div
          class="border-border/70 bg-background/80 text-muted-foreground inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs"
        >
          <LoaderCircle class="size-3 animate-spin" />
          Refreshing
        </div>
      {/if}
    </div>
  </CardHeader>

  <CardContent class="grid gap-4 pt-6 md:grid-cols-2 xl:grid-cols-4">
    <div class="border-border/70 bg-background/70 rounded-3xl border p-4">
      <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Status</p>
      <p class="text-foreground mt-3 text-lg font-semibold">{detail.ticket.status_name}</p>
      <p class="text-muted-foreground mt-2 text-sm">{detail.ticket.type}</p>
    </div>
    <div class="border-border/70 bg-background/70 rounded-3xl border p-4">
      <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Linked Repos</p>
      <p class="text-foreground mt-3 text-lg font-semibold">{detail.repo_scopes.length}</p>
      <p class="text-muted-foreground mt-2 text-sm">
        {detail.hook_history.length} hook events captured
      </p>
    </div>
    <div class="border-border/70 bg-background/70 rounded-3xl border p-4">
      <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Attempts</p>
      <p class="text-foreground mt-3 text-lg font-semibold">{detail.ticket.attempt_count}</p>
      <p class="text-muted-foreground mt-2 text-sm">
        {detail.ticket.retry_paused
          ? detail.ticket.pause_reason || 'Retry paused'
          : `${detail.ticket.consecutive_errors} consecutive errors`}
      </p>
    </div>
    <div class="border-border/70 bg-background/70 rounded-3xl border p-4">
      <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Created</p>
      <p class="text-foreground mt-3 text-lg font-semibold">
        {formatTimestamp(detail.ticket.created_at)}
      </p>
      <p class="text-muted-foreground mt-2 text-sm">{detail.ticket.created_by}</p>
    </div>
  </CardContent>
</Card>
