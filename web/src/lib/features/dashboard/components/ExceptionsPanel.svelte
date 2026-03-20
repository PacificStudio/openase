<script lang="ts">
  import { TriangleAlert } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'

  let {
    boardError = '',
    dashboardError = '',
    hrAdvisorError = '',
    stalledAgentCount = 0,
    pendingMutationCount = 0,
  }: {
    boardError?: string
    dashboardError?: string
    hrAdvisorError?: string
    stalledAgentCount?: number
    pendingMutationCount?: number
  } = $props()

  const issues = $derived(
    [boardError, dashboardError, hrAdvisorError]
      .map((message, index) =>
        message
          ? {
              label: ['Board', 'Activity', 'Staffing'][index] ?? 'System',
              message,
            }
          : null,
      )
      .filter(Boolean) as Array<{ label: string; message: string }>,
  )
</script>

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <div class="flex items-center justify-between gap-3">
      <div>
        <CardTitle class="flex items-center gap-2">
          <TriangleAlert class="size-4" />
          <span>Exceptions</span>
        </CardTitle>
        <CardDescription
          >Surface stale agents, mutation backlog, and failing streams.</CardDescription
        >
      </div>
      <Badge variant={issues.length > 0 || stalledAgentCount > 0 ? 'destructive' : 'secondary'}>
        {issues.length + stalledAgentCount}
      </Badge>
    </div>
  </CardHeader>

  <CardContent class="space-y-3">
    <div class="grid gap-3 sm:grid-cols-2">
      <div class="border-border/70 bg-background/60 rounded-3xl border px-4 py-4">
        <p class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
          Stalled agents
        </p>
        <p class="mt-2 text-2xl font-semibold tracking-[-0.05em]">{stalledAgentCount}</p>
      </div>
      <div class="border-border/70 bg-background/60 rounded-3xl border px-4 py-4">
        <p class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
          Pending board mutations
        </p>
        <p class="mt-2 text-2xl font-semibold tracking-[-0.05em]">{pendingMutationCount}</p>
      </div>
    </div>

    {#if issues.length === 0 && stalledAgentCount === 0}
      <div
        class="rounded-3xl border border-emerald-500/25 bg-emerald-500/10 px-4 py-4 text-sm text-emerald-900"
      >
        No active exceptions in the current control plane slice.
      </div>
    {:else}
      {#each issues as issue}
        <div
          class="rounded-3xl border border-rose-500/25 bg-rose-500/10 px-4 py-4 text-sm text-rose-900"
        >
          <p class="font-medium tracking-[0.18em] uppercase">{issue.label}</p>
          <p class="mt-1 leading-6">{issue.message}</p>
        </div>
      {/each}
      {#if stalledAgentCount > 0}
        <div
          class="rounded-3xl border border-amber-500/25 bg-amber-500/10 px-4 py-4 text-sm text-amber-900"
        >
          {stalledAgentCount} agent{stalledAgentCount === 1 ? '' : 's'} exceeded the healthy heartbeat
          window.
        </div>
      {/if}
    {/if}
  </CardContent>
</Card>
