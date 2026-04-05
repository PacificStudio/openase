<script lang="ts">
  import { onMount } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import type { BuiltinRole, HRAdvisorRecommendation } from '$lib/api/contracts'
  import { activateHRRecommendation, getHRAdvisor, listBuiltinRoles } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { ChevronDown, ChevronRight } from '@lucide/svelte'
  import type { HRAdvisorSnapshot } from '../types'
  import HRAdvisorHarnessDialog from './hr-advisor-harness-dialog.svelte'
  import HRAdvisorRecommendationCard from './hr-advisor-recommendation-card.svelte'
  import {
    activationStatusText,
    applyActivatedRecommendation,
    loadDeferredRecommendationKeys,
    persistDeferredRecommendationKeys as persistDeferredRecommendationKeysForProject,
    recommendationKey,
    toPrioritySectionKey,
  } from './hr-advisor-panel-state'

  const priorityDotClass: Record<string, string> = {
    high: 'bg-rose-500',
    medium: 'bg-amber-500',
    low: 'bg-sky-500',
    other: 'bg-muted-foreground',
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
  let showDeferred = $state(false)

  $effect(() => {
    void advisor
    advisorStateOverride = null
  })

  onMount(() => {
    deferredRecommendationKeys = loadDeferredRecommendationKeys(projectId)
  })

  const advisorState = $derived(advisorStateOverride ?? advisor)

  const visibleRecommendations = $derived(
    advisorState.recommendations.filter(
      (r) => !deferredRecommendationKeys.includes(recommendationKey(r)),
    ),
  )

  const deferredRecommendations = $derived(
    advisorState.recommendations.filter((r) =>
      deferredRecommendationKeys.includes(recommendationKey(r)),
    ),
  )

  const selectedHarness = $derived(
    selectedHarnessRoleSlug ? (builtinRolesBySlug[selectedHarnessRoleSlug] ?? null) : null,
  )

  function persistDeferredRecommendationKeys(nextKeys: string[]) {
    deferredRecommendationKeys = persistDeferredRecommendationKeysForProject(projectId, nextKeys)
  }

  function deferRecommendation(recommendation: HRAdvisorRecommendation) {
    const key = recommendationKey(recommendation)
    if (deferredRecommendationKeys.includes(key)) return
    persistDeferredRecommendationKeys([...deferredRecommendationKeys, key])
  }

  function restoreRecommendation(recommendation: HRAdvisorRecommendation) {
    const key = recommendationKey(recommendation)
    persistDeferredRecommendationKeys(deferredRecommendationKeys.filter((k) => k !== key))
  }

  async function openHarnessDialog(recommendation: HRAdvisorRecommendation) {
    selectedHarnessRoleSlug = recommendation.role_slug
    selectedHarnessRoleName = recommendation.role_name
    harnessError = ''
    harnessDialogOpen = true

    if (builtinRolesBySlug[recommendation.role_slug]) return

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
    if (!recommendation.activation_ready || activatingRecommendationKey === key) return

    activatingRecommendationKey = key
    activationErrors = Object.fromEntries(
      Object.entries(activationErrors).filter(([k]) => k !== key),
    )

    try {
      const payload = await activateHRRecommendation(projectId, {
        role_slug: recommendation.role_slug,
        create_bootstrap_ticket: true,
      })

      persistDeferredRecommendationKeys(deferredRecommendationKeys.filter((k) => k !== key))

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
      activationErrors = { ...activationErrors, [key]: detail }
      toastStore.error(detail)
    } finally {
      activatingRecommendationKey = null
    }
  }
</script>

<div class={cn('border-border bg-card rounded-xl border', className)}>
  <div class="flex items-center justify-between px-4 py-3">
    <div class="flex items-center gap-2">
      <h3 class="text-foreground text-sm font-medium">HR Advisor</h3>
      <Badge variant="outline" class="text-[10px]">
        {advisorState.summary.workflow_count} workflows
      </Badge>
    </div>
    {#if visibleRecommendations.length > 0}
      <span class="text-muted-foreground text-xs">
        {visibleRecommendations.filter((r) => r.activation_ready).length} ready to activate
      </span>
    {/if}
  </div>

  <div class="space-y-2 px-4 pb-4">
    {#if visibleRecommendations.length > 0}
      {#each visibleRecommendations as recommendation (recommendationKey(recommendation))}
        {@const key = recommendationKey(recommendation)}
        <HRAdvisorRecommendationCard
          {recommendation}
          priorityDotClass={priorityDotClass[toPrioritySectionKey(recommendation.priority)]}
          activationStatus={activationStatusText(recommendation)}
          activationError={activationErrors[key]}
          activating={activatingRecommendationKey === key}
          onViewHarness={() => void openHarnessDialog(recommendation)}
          onActivate={() => void activateRecommendation(recommendation)}
          onDefer={() => deferRecommendation(recommendation)}
        />
      {/each}
    {:else if deferredRecommendations.length > 0}
      <p class="text-muted-foreground py-4 text-center text-xs">
        All recommendations are deferred. Expand below to review them.
      </p>
    {:else}
      <p class="text-muted-foreground py-4 text-center text-xs">
        No staffing recommendations right now.
      </p>
    {/if}

    {#if deferredRecommendations.length > 0}
      <button
        type="button"
        class="text-muted-foreground hover:text-foreground flex w-full items-center gap-1.5 pt-1 text-xs transition-colors"
        onclick={() => (showDeferred = !showDeferred)}
      >
        {#if showDeferred}
          <ChevronDown class="size-3" />
        {:else}
          <ChevronRight class="size-3" />
        {/if}
        Deferred ({deferredRecommendations.length})
      </button>

      {#if showDeferred}
        <div class="space-y-2">
          {#each deferredRecommendations as recommendation (recommendationKey(recommendation))}
            {@const key = recommendationKey(recommendation)}
            <HRAdvisorRecommendationCard
              {recommendation}
              priorityDotClass={priorityDotClass[toPrioritySectionKey(recommendation.priority)]}
              activationStatus={activationStatusText(recommendation)}
              activationError={activationErrors[key]}
              activating={activatingRecommendationKey === key}
              deferred
              onViewHarness={() => void openHarnessDialog(recommendation)}
              onActivate={() => void activateRecommendation(recommendation)}
              onRestore={() => restoreRecommendation(recommendation)}
            />
          {/each}
        </div>
      {/if}
    {/if}
  </div>
</div>

<HRAdvisorHarnessDialog
  bind:open={harnessDialogOpen}
  harness={selectedHarness}
  roleName={selectedHarnessRoleName}
  loading={harnessLoading}
  error={harnessError}
/>
