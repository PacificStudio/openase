<script lang="ts">
  import { onMount } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import type { BuiltinRole, HRAdvisorRecommendation } from '$lib/api/contracts'
  import { activateHRRecommendation, getHRAdvisor, listBuiltinRoles } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import type { HRAdvisorSnapshot } from '../types'
  import HRAdvisorHarnessDialog from './hr-advisor-harness-dialog.svelte'
  import HRAdvisorRecommendationCard from './hr-advisor-recommendation-card.svelte'
  import {
    activationStatusText,
    applyActivatedRecommendation,
    loadDeferredRecommendationKeys,
    persistDeferredRecommendationKeys as persistDeferredRecommendationKeysForProject,
    prioritySectionsMeta,
    recommendationKey,
    toPrioritySectionKey,
  } from './hr-advisor-panel-state'

  type PrioritySection = (typeof prioritySectionsMeta)[number] & {
    recommendations: HRAdvisorRecommendation[]
  }

  let {
    projectId,
    advisor,
    class: className = '',
  }: {
    projectId: string
    advisor: HRAdvisorSnapshot
    class?: string
  } = $props()

  let advisorStateOverride = $state<HRAdvisorSnapshot | null>(null)
  let deferredRecommendationKeys = $state<string[]>([])
  let activatingRecommendationKey = $state<string | null>(null)
  let activationErrors = $state<Record<string, string>>({})
  let builtinRolesBySlug = $state<Record<string, BuiltinRole>>({})
  let harnessDialogOpen = $state(false)
  let selectedHarnessRoleSlug = $state('')
  let selectedHarnessRoleName = $state('')
  let harnessLoading = $state(false)
  let harnessError = $state('')

  $effect(() => {
    void advisor
    advisorStateOverride = null
  })

  onMount(() => {
    deferredRecommendationKeys = loadDeferredRecommendationKeys(projectId)
  })

  const advisorState = $derived(advisorStateOverride ?? advisor)

  const staffingEntries = $derived(
    Object.entries(advisorState.staffing)
      .filter(([, count]) => count > 0)
      .sort((left, right) => right[1] - left[1]),
  )

  const visibleRecommendations = $derived(
    advisorState.recommendations.filter(
      (recommendation) => !deferredRecommendationKeys.includes(recommendationKey(recommendation)),
    ),
  )

  const deferredRecommendations = $derived(
    advisorState.recommendations.filter((recommendation) =>
      deferredRecommendationKeys.includes(recommendationKey(recommendation)),
    ),
  )

  const prioritySections = $derived(
    prioritySectionsMeta
      .map((section) => ({
        ...section,
        recommendations: visibleRecommendations.filter(
          (recommendation) => toPrioritySectionKey(recommendation.priority) === section.key,
        ),
      }))
      .filter((section) => section.recommendations.length > 0) as PrioritySection[],
  )

  const selectedHarness = $derived(
    selectedHarnessRoleSlug ? (builtinRolesBySlug[selectedHarnessRoleSlug] ?? null) : null,
  )

  function persistDeferredRecommendationKeys(nextKeys: string[]) {
    deferredRecommendationKeys = persistDeferredRecommendationKeysForProject(projectId, nextKeys)
  }

  function deferRecommendation(recommendation: HRAdvisorRecommendation) {
    const key = recommendationKey(recommendation)
    if (deferredRecommendationKeys.includes(key)) {
      return
    }

    persistDeferredRecommendationKeys([...deferredRecommendationKeys, key])
  }

  function restoreRecommendation(recommendation: HRAdvisorRecommendation) {
    const key = recommendationKey(recommendation)
    persistDeferredRecommendationKeys(
      deferredRecommendationKeys.filter((existingKey) => existingKey !== key),
    )
  }

  async function openHarnessDialog(recommendation: HRAdvisorRecommendation) {
    selectedHarnessRoleSlug = recommendation.role_slug
    selectedHarnessRoleName = recommendation.role_name
    harnessError = ''
    harnessDialogOpen = true

    if (builtinRolesBySlug[recommendation.role_slug]) {
      return
    }

    harnessLoading = true

    try {
      const payload = await listBuiltinRoles()
      builtinRolesBySlug = Object.fromEntries(payload.roles.map((role) => [role.slug, role]))

      if (!payload.roles.find((role) => role.slug === recommendation.role_slug)) {
        harnessError = 'Role harness template is unavailable.'
      }
    } catch (caughtError) {
      harnessError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load role harness.'
    } finally {
      harnessLoading = false
    }
  }

  async function activateRecommendation(recommendation: HRAdvisorRecommendation) {
    const key = recommendationKey(recommendation)
    if (!recommendation.activation_ready || activatingRecommendationKey === key) {
      return
    }

    activatingRecommendationKey = key
    activationErrors = Object.fromEntries(
      Object.entries(activationErrors).filter(([entryKey]) => entryKey !== key),
    )

    try {
      const payload = await activateHRRecommendation(projectId, {
        role_slug: recommendation.role_slug,
        create_bootstrap_ticket: true,
      })

      persistDeferredRecommendationKeys(
        deferredRecommendationKeys.filter((entryKey) => entryKey !== key),
      )

      advisorStateOverride = applyActivatedRecommendation(
        advisorState,
        recommendation,
        payload.workflow.name || recommendation.suggested_workflow_name,
      )

      const bootstrapTicketIdentifier = payload.bootstrap_ticket.ticket?.identifier
      toastStore.success(
        bootstrapTicketIdentifier
          ? `Activated ${recommendation.role_name} and created ${bootstrapTicketIdentifier}.`
          : `Activated ${recommendation.role_name}.`,
      )

      try {
        const refreshedAdvisor = await getHRAdvisor(projectId)
        advisorStateOverride = {
          summary: refreshedAdvisor.summary,
          staffing: refreshedAdvisor.staffing,
          recommendations: refreshedAdvisor.recommendations,
        }
      } catch {
        // Keep the optimistic card state if a follow-up refresh fails.
      }
    } catch (caughtError) {
      const detail =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to activate recommendation.'

      activationErrors = {
        ...activationErrors,
        [key]: detail,
      }
      toastStore.error(detail)
    } finally {
      activatingRecommendationKey = null
    }
  }
</script>

<div class={cn('border-border bg-card rounded-md border', className)}>
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <div>
      <h3 class="text-foreground text-sm font-medium">HR Advisor</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Inspect builtin role harnesses, activate recommended workflows, or defer them from the
        dashboard.
      </p>
    </div>
    <Badge variant="outline" class="shrink-0 text-[10px]">
      {advisorState.summary.workflow_count} workflows
    </Badge>
  </div>

  <div class="space-y-4 px-4 py-4">
    <div class="grid grid-cols-2 gap-3 text-sm">
      <div class="rounded-md border border-emerald-500/20 bg-emerald-500/5 px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Open tickets</div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {advisorState.summary.open_tickets}
        </div>
      </div>
      <div class="rounded-md border border-amber-500/20 bg-amber-500/5 px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Blocked</div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {advisorState.summary.blocked_tickets}
        </div>
      </div>
      <div class="rounded-md border border-sky-500/20 bg-sky-500/5 px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Coding</div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {advisorState.summary.coding_tickets}
        </div>
      </div>
      <div class="rounded-md border border-violet-500/20 bg-violet-500/5 px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Active agents</div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {advisorState.summary.active_agents}
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

      {#if prioritySections.length > 0}
        {#each prioritySections as section (section.key)}
          <section class="space-y-3">
            <div class="flex items-center justify-between">
              <div class="text-foreground text-sm font-medium">{section.label}</div>
              <Badge variant="outline" class="text-[10px]">
                {section.recommendations.length}
              </Badge>
            </div>

            {#each section.recommendations as recommendation (recommendationKey(recommendation))}
              {@const key = recommendationKey(recommendation)}
              <HRAdvisorRecommendationCard
                {recommendation}
                cardClass={section.accentClass}
                activationStatus={activationStatusText(recommendation)}
                activationError={activationErrors[key]}
                activating={activatingRecommendationKey === key}
                onViewHarness={() => void openHarnessDialog(recommendation)}
                onActivate={() => void activateRecommendation(recommendation)}
                onTertiaryAction={() => deferRecommendation(recommendation)}
              />
            {/each}
          </section>
        {/each}
      {:else if deferredRecommendations.length > 0}
        <p class="text-muted-foreground text-xs">
          All current recommendations are deferred. Restore any card below when you want it back in
          the active queue.
        </p>
      {:else}
        <p class="text-muted-foreground text-xs">No staffing recommendations available.</p>
      {/if}

      {#if deferredRecommendations.length > 0}
        <section class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="text-foreground text-sm font-medium">已延后</div>
            <Badge variant="secondary" class="text-[10px]">{deferredRecommendations.length}</Badge>
          </div>

          {#each deferredRecommendations as recommendation (recommendationKey(recommendation))}
            {@const key = recommendationKey(recommendation)}
            <HRAdvisorRecommendationCard
              {recommendation}
              cardClass="border-border bg-muted/20"
              activationStatus={activationStatusText(recommendation)}
              activationError={activationErrors[key]}
              activating={activatingRecommendationKey === key}
              tertiaryActionLabel="重新显示"
              showEvidence={false}
              onViewHarness={() => void openHarnessDialog(recommendation)}
              onActivate={() => void activateRecommendation(recommendation)}
              onTertiaryAction={() => restoreRecommendation(recommendation)}
            />
          {/each}
        </section>
      {/if}
    </div>
  </div>
</div>

<HRAdvisorHarnessDialog
  bind:open={harnessDialogOpen}
  harness={selectedHarness}
  roleName={selectedHarnessRoleName}
  loading={harnessLoading}
  error={harnessError}
/>
