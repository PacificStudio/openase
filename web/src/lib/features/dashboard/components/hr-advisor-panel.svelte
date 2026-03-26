<script lang="ts">
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import type { HRAdvisorSnapshot } from '../types'

  let {
    advisor,
    class: className = '',
  }: {
    advisor: HRAdvisorSnapshot
    class?: string
  } = $props()

  const staffingEntries = $derived(
    Object.entries(advisor.staffing)
      .filter(([, count]) => count > 0)
      .sort((left, right) => right[1] - left[1]),
  )
</script>

<div class={cn('border-border bg-card rounded-md border', className)}>
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <div>
      <h3 class="text-foreground text-sm font-medium">HR Advisor</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Staffing guidance derived from current ticket and workflow load.
      </p>
    </div>
    <Badge variant="outline" class="shrink-0 text-[10px]">
      {advisor.summary.workflow_count} workflows
    </Badge>
  </div>

  <div class="space-y-4 px-4 py-4">
    <div class="grid grid-cols-2 gap-3 text-sm">
      <div class="rounded-md border border-emerald-500/20 bg-emerald-500/5 px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Open tickets</div>
        <div class="text-foreground mt-1 text-lg font-semibold">{advisor.summary.open_tickets}</div>
      </div>
      <div class="rounded-md border border-amber-500/20 bg-amber-500/5 px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Blocked</div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {advisor.summary.blocked_tickets}
        </div>
      </div>
      <div class="rounded-md border border-sky-500/20 bg-sky-500/5 px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Coding</div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {advisor.summary.coding_tickets}
        </div>
      </div>
      <div class="rounded-md border border-violet-500/20 bg-violet-500/5 px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Active agents</div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {advisor.summary.active_agents}
        </div>
      </div>
    </div>

    <div class="space-y-2">
      <div class="text-foreground text-sm font-medium">Suggested staffing</div>
      {#if staffingEntries.length > 0}
        <div class="flex flex-wrap gap-2">
          {#each staffingEntries as [role, count] (role)}
            <Badge variant="secondary" class="capitalize">{role}: {count}</Badge>
          {/each}
        </div>
      {:else}
        <p class="text-muted-foreground text-xs">No additional staffing pressure detected.</p>
      {/if}
    </div>

    <div class="space-y-3">
      <div class="text-foreground text-sm font-medium">Recommendations</div>
      {#if advisor.recommendations.length > 0}
        {#each advisor.recommendations as recommendation (recommendation.role_slug + recommendation.suggested_workflow_name)}
          <div class="border-border bg-muted/20 rounded-md border px-3 py-3">
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="text-foreground text-sm font-medium">
                  {recommendation.role_name}
                  {#if recommendation.suggested_headcount > 0}
                    <span class="text-muted-foreground ml-2 text-xs">
                      x{recommendation.suggested_headcount}
                    </span>
                  {/if}
                </div>
                <div class="text-muted-foreground mt-1 text-xs">
                  {recommendation.summary}
                </div>
              </div>
              <Badge variant="outline" class="shrink-0 capitalize">
                {recommendation.priority}
              </Badge>
            </div>

            <div class="text-muted-foreground mt-3 flex flex-wrap gap-2 text-[11px]">
              <span>Workflow: {recommendation.suggested_workflow_name}</span>
              {#if recommendation.active_workflow_name}
                <span>Active: {recommendation.active_workflow_name}</span>
              {/if}
              <span>Type: {recommendation.workflow_type}</span>
            </div>

            {#if recommendation.evidence.length > 0}
              <div class="mt-3 space-y-1">
                {#each recommendation.evidence as evidence (evidence)}
                  <p class="text-muted-foreground text-xs">{evidence}</p>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      {:else}
        <p class="text-muted-foreground text-xs">No staffing recommendations available.</p>
      {/if}
    </div>
  </div>
</div>
