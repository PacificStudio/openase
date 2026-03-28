<script lang="ts">
  import type { HRAdvisorRecommendation } from '$lib/api/contracts'
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'

  let {
    recommendation,
    cardClass = '',
    activationStatus,
    activationError = '',
    activating = false,
    tertiaryActionLabel = '稍后再说',
    showEvidence = true,
    onViewHarness,
    onActivate,
    onTertiaryAction,
  }: {
    recommendation: HRAdvisorRecommendation
    cardClass?: string
    activationStatus: string
    activationError?: string
    activating?: boolean
    tertiaryActionLabel?: string
    showEvidence?: boolean
    onViewHarness?: () => void
    onActivate?: () => void
    onTertiaryAction?: () => void
  } = $props()
</script>

<article class={cn('rounded-md border px-3 py-3', cardClass)}>
  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0">
      <div class="flex flex-wrap items-center gap-2">
        <h4 class="text-foreground text-sm font-medium">{recommendation.role_name}</h4>
        <span class="text-muted-foreground text-[11px]">{recommendation.role_slug}</span>
        {#if recommendation.suggested_headcount > 0}
          <Badge variant="secondary" class="text-[10px]">
            x{recommendation.suggested_headcount}
          </Badge>
        {/if}
      </div>
      <p class="text-muted-foreground mt-1 text-xs">{recommendation.summary}</p>
    </div>

    <Badge variant={recommendation.activation_ready ? 'secondary' : 'outline'}>
      {recommendation.activation_ready ? '可激活' : '已激活'}
    </Badge>
  </div>

  <div class="text-muted-foreground mt-3 flex flex-wrap gap-2 text-[11px]">
    <span>Workflow: {recommendation.suggested_workflow_name}</span>
    <span>Type: {recommendation.workflow_type}</span>
    <span>Harness: {recommendation.harness_path}</span>
  </div>

  <div class="mt-3 space-y-1">
    <p class="text-foreground text-xs">
      推荐理由:
      <span class="text-muted-foreground">{recommendation.reason}</span>
    </p>
    <p class="text-foreground text-xs">
      当前状态:
      <span class="text-muted-foreground">{activationStatus}</span>
    </p>
  </div>

  {#if showEvidence && recommendation.evidence.length > 0}
    <div class="mt-3 space-y-1">
      {#each recommendation.evidence as evidence (evidence)}
        <p class="text-muted-foreground text-xs">{evidence}</p>
      {/each}
    </div>
  {/if}

  {#if activationError}
    <div
      class="border-destructive/30 bg-destructive/10 text-destructive mt-3 rounded-md border px-3 py-2 text-xs"
    >
      {activationError}
    </div>
  {/if}

  <div class="mt-4 flex flex-wrap gap-2">
    <Button size="sm" variant="outline" onclick={() => onViewHarness?.()}>查看角色 Harness</Button>
    <Button
      size="sm"
      disabled={!recommendation.activation_ready || activating}
      onclick={() => onActivate?.()}
    >
      {activating ? '激活中…' : '一键激活'}
    </Button>
    <Button size="sm" variant="ghost" onclick={() => onTertiaryAction?.()}>
      {tertiaryActionLabel}
    </Button>
  </div>
</article>
