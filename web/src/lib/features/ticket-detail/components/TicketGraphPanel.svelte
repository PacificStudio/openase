<script lang="ts">
  import { Clock3, TriangleAlert } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import { formatTimestamp } from '$lib/features/workspace'
  import type { TicketDetailPayload } from '../types'

  let {
    detail,
  }: {
    detail: TicketDetailPayload
  } = $props()

  const timelineEntries = $derived.by(() => {
    const activityEntries = detail.activity.map((item) => ({
      id: item.id,
      label: item.event_type,
      detail: item.message || 'No activity message provided.',
      createdAt: item.created_at,
    }))

    return [
      {
        id: `${detail.ticket.id}-created`,
        label: 'ticket.created',
        detail: `${detail.ticket.identifier} opened by ${detail.ticket.created_by}`,
        createdAt: detail.ticket.created_at,
      },
      ...activityEntries,
    ].sort((left, right) => Date.parse(left.createdAt) - Date.parse(right.createdAt))
  })
</script>

<Card class="border-border/80 bg-background/85 backdrop-blur">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <TriangleAlert class="size-4" />
      <span>Dependencies</span>
    </CardTitle>
    <CardDescription>
      Parent links and dependencies stay visible without leaving the detail page.
    </CardDescription>
  </CardHeader>

  <CardContent class="text-muted-foreground space-y-3 text-sm">
    <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
      <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Parent</p>
      <p class="text-foreground mt-2 text-sm">
        {detail.ticket.parent
          ? `${detail.ticket.parent.identifier} · ${detail.ticket.parent.title}`
          : 'No parent ticket'}
      </p>
    </div>

    <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
      <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Children</p>
      {#if detail.ticket.children.length === 0}
        <p class="mt-2">No child tickets linked yet.</p>
      {:else}
        <div class="mt-2 flex flex-wrap gap-2">
          {#each detail.ticket.children as child}
            <Badge variant="outline">{child.identifier}</Badge>
          {/each}
        </div>
      {/if}
    </div>

    <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
      <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Blocks / Depends on</p>
      {#if detail.ticket.dependencies.length === 0}
        <p class="mt-2">No dependency edges recorded yet.</p>
      {:else}
        <div class="mt-2 space-y-2">
          {#each detail.ticket.dependencies as dependency}
            <div class="flex items-center justify-between gap-3 rounded-2xl border border-white/50 px-3 py-2">
              <span class="font-medium">
                {dependency.target.identifier} · {dependency.target.title}
              </span>
              <Badge variant="outline">{dependency.type}</Badge>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
      <div class="flex items-center gap-2">
        <Clock3 class="size-3.5" />
        <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Activity timeline</p>
      </div>
      <div class="mt-4 space-y-3">
        {#each timelineEntries.slice(-6).reverse() as item}
          <div class="flex gap-3">
            <span class="mt-1 size-2 rounded-full bg-emerald-500/70"></span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <p class="text-foreground text-sm font-semibold">{item.label}</p>
                <span class="text-muted-foreground text-xs">{formatTimestamp(item.createdAt)}</span>
              </div>
              <p class="text-muted-foreground mt-1 text-sm leading-6">{item.detail}</p>
            </div>
          </div>
        {/each}
      </div>
    </div>
  </CardContent>
</Card>
