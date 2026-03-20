<script lang="ts">
  import { Rocket, Waypoints } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
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
  } from '$lib/features/workspace/mappers'
  import type { HRAdvisorPayload, OnboardingSummary, Project } from '$lib/features/workspace/types'

  let {
    project = null,
    workflowCount = 0,
    statusCount = 0,
    ticketCount = 0,
    onboardingSummary,
    hrAdvisor = null,
    hrAdvisorBusy = false,
    hrAdvisorError = '',
  }: {
    project?: Project | null
    workflowCount?: number
    statusCount?: number
    ticketCount?: number
    onboardingSummary: OnboardingSummary
    hrAdvisor?: HRAdvisorPayload | null
    hrAdvisorBusy?: boolean
    hrAdvisorError?: string
  } = $props()
</script>

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <CardTitle class="flex items-center gap-2">
          <Rocket class="size-4" />
          <span>Project health</span>
        </CardTitle>
        <CardDescription>
          {project
            ? `${project.name} is the active workspace.`
            : 'Select a project to unlock board and workflow telemetry.'}
        </CardDescription>
      </div>
      <div class="flex flex-wrap gap-2">
        <Badge variant={onboardingSummary.complete ? 'secondary' : 'outline'}>
          {onboardingSummary.progressPercent}% ready
        </Badge>
        <Badge variant="outline">{workflowCount} workflows</Badge>
        <Badge variant="outline">{ticketCount} tickets</Badge>
      </div>
    </div>
  </CardHeader>

  <CardContent class="space-y-5">
    <div class="grid gap-3 sm:grid-cols-4">
      <div class="border-border/70 bg-background/60 rounded-3xl border px-4 py-4">
        <p class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
          Board lanes
        </p>
        <p class="mt-2 text-2xl font-semibold tracking-[-0.05em]">{statusCount}</p>
      </div>
      {#each onboardingSummary.stats.slice(0, 3) as stat}
        <div class="border-border/70 bg-background/60 rounded-3xl border px-4 py-4">
          <p class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
            {stat.label}
          </p>
          <p class="mt-2 text-2xl font-semibold tracking-[-0.05em]">{stat.value}</p>
        </div>
      {/each}
    </div>

    <div
      class="rounded-[1.75rem] border border-amber-500/20 bg-[linear-gradient(135deg,rgba(245,158,11,0.15),rgba(255,255,255,0.94)_42%,rgba(16,185,129,0.1))] px-5 py-5"
    >
      <div class="flex items-center gap-2 text-sm font-medium text-slate-800">
        <Waypoints class="size-4" />
        <span>{onboardingSummary.title}</span>
      </div>
      <p class="mt-3 text-sm leading-6 text-slate-700">{onboardingSummary.description}</p>
      <div class="mt-4 h-2 overflow-hidden rounded-full bg-white/75">
        <div
          class={`h-full rounded-full ${onboardingSummary.complete ? 'bg-emerald-600' : 'bg-amber-500'}`}
          style={`width: ${onboardingSummary.progressPercent}%;`}
        ></div>
      </div>
      <p class="mt-3 text-sm font-medium text-slate-800">{onboardingSummary.actionLabel}</p>
    </div>

    {#if hrAdvisorBusy}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border px-4 py-4 text-sm"
      >
        Loading staffing signal…
      </div>
    {:else if hrAdvisorError}
      <div
        class="text-destructive border-destructive/25 bg-destructive/10 rounded-3xl border px-4 py-4 text-sm"
      >
        {hrAdvisorError}
      </div>
    {:else if hrAdvisor}
      <div class="grid gap-4 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
        <div class="border-border/70 bg-background/60 space-y-3 rounded-3xl border px-4 py-4">
          <p class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
            Recommended staffing
          </p>
          {#if staffingEntries(hrAdvisor.staffing).length === 0}
            <p class="text-muted-foreground text-sm">No explicit staffing recommendations yet.</p>
          {:else}
            <div class="flex flex-wrap gap-2">
              {#each staffingEntries(hrAdvisor.staffing) as entry}
                <Badge variant="outline">{entry.label}: {entry.value}</Badge>
              {/each}
            </div>
          {/if}
        </div>

        <div class="grid gap-3">
          {#if hrAdvisor.recommendations.length === 0}
            <div
              class="text-muted-foreground border-border/70 bg-background/60 rounded-3xl border px-4 py-4 text-sm"
            >
              No role recommendations for this project yet.
            </div>
          {:else}
            {#each hrAdvisor.recommendations.slice(0, 2) as recommendation}
              <div
                class={`rounded-3xl border px-4 py-4 ${hrAdvisorPriorityCardClass(recommendation.priority)}`}
              >
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold">{recommendation.role_name}</p>
                    <p class="mt-1 text-sm leading-6 opacity-80">{recommendation.summary}</p>
                  </div>
                  <Badge class={hrAdvisorPriorityBadgeClass(recommendation.priority)}
                    >{recommendation.priority}</Badge
                  >
                </div>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    {/if}
  </CardContent>
</Card>
