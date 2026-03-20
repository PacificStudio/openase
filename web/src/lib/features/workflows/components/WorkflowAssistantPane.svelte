<script lang="ts">
  import { Bot, Sparkles } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import {
    hrAdvisorPriorityBadgeClass,
    hrAdvisorPriorityCardClass,
    staffingEntries,
    type BuiltinRole,
    type HRAdvisorPayload,
  } from '$lib/features/workspace'

  let {
    builtinRoles = [],
    hrAdvisor = null,
    onLoadRecommendedRole,
  }: {
    builtinRoles?: BuiltinRole[]
    hrAdvisor?: HRAdvisorPayload | null
    onLoadRecommendedRole?: (recommendation: {
      role_slug: string
      suggested_workflow_name: string
    }) => void
  } = $props()
</script>

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <Bot class="size-4" />
      <span>AI assistant pane</span>
    </CardTitle>
    <CardDescription>
      Built-in role templates and HR advisor signals stay next to the editor instead of hiding in
      the route file.
    </CardDescription>
  </CardHeader>

  <CardContent class="space-y-4">
    <div class="flex flex-wrap gap-2">
      {#each builtinRoles.slice(0, 8) as role}
        <Badge variant="outline">{role.name}</Badge>
      {/each}
    </div>

    {#if hrAdvisor}
      <div class="grid gap-3 sm:grid-cols-3">
        {#each staffingEntries(hrAdvisor.staffing).slice(0, 3) as item}
          <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
            <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">{item.label}</p>
            <p class="mt-2 text-lg font-semibold">{item.value}</p>
          </div>
        {/each}
      </div>

      {#if hrAdvisor.recommendations.length === 0}
        <div
          class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
        >
          No workflow recommendations yet for the selected project.
        </div>
      {:else}
        <div class="space-y-3">
          {#each hrAdvisor.recommendations.slice(0, 3) as recommendation}
            <div
              class={`rounded-3xl border px-4 py-4 ${hrAdvisorPriorityCardClass(recommendation.priority)}`}
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div class="flex items-center gap-2">
                    <p class="text-sm font-semibold">{recommendation.role_name}</p>
                    <Badge class={hrAdvisorPriorityBadgeClass(recommendation.priority)}>
                      {recommendation.priority}
                    </Badge>
                  </div>
                  <p class="text-muted-foreground mt-2 text-sm">{recommendation.reason}</p>
                </div>

                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onclick={() => onLoadRecommendedRole?.(recommendation)}
                >
                  <Sparkles class="mr-2 size-4" />
                  Load
                </Button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {:else}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
      >
        Select a project to load staffing advice and role recommendations.
      </div>
    {/if}
  </CardContent>
</Card>
