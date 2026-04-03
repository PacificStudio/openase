<script lang="ts">
  import type { HRAdvisorRecommendation } from '$lib/api/contracts'
  import {
    normalizeWorkflowFamily,
    workflowFamilyColors,
    workflowFamilyIcons,
  } from '$lib/features/workflows'
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { ChevronDown, ChevronRight, Clock, Ellipsis, Eye, Play, Undo2, Zap } from '@lucide/svelte'

  let {
    recommendation,
    priorityDotClass = '',
    activationStatus,
    activationError = '',
    activating = false,
    deferred = false,
    onViewHarness,
    onActivate,
    onDefer,
    onRestore,
  }: {
    recommendation: HRAdvisorRecommendation
    priorityDotClass?: string
    activationStatus: string
    activationError?: string
    activating?: boolean
    deferred?: boolean
    onViewHarness?: () => void
    onActivate?: () => void
    onDefer?: () => void
    onRestore?: () => void
  } = $props()

  let expanded = $state(false)
  const workflowFamily = $derived(normalizeWorkflowFamily(recommendation.workflow_family ?? ''))
</script>

<article
  class={cn(
    'border-border/60 bg-card/60 rounded-xl border transition-colors',
    deferred && 'opacity-60',
  )}
>
  <!-- Compact row -->
  <button
    type="button"
    class="flex w-full items-center gap-3 px-4 py-3 text-left"
    onclick={() => (expanded = !expanded)}
  >
    <span class={cn('size-2 shrink-0 rounded-full', priorityDotClass)}></span>
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <span class="text-foreground text-sm font-medium">{recommendation.role_name}</span>
        {#if recommendation.suggested_headcount > 1}
          <Badge variant="secondary" class="text-[10px]">
            x{recommendation.suggested_headcount}
          </Badge>
        {/if}
        {#if !recommendation.activation_ready}
          <Badge variant="outline" class="text-[10px] text-emerald-600 dark:text-emerald-400">
            已激活
          </Badge>
        {/if}
      </div>
      <p class="text-muted-foreground mt-0.5 truncate text-xs">{recommendation.reason}</p>
    </div>

    {#if recommendation.activation_ready && !deferred}
      <Button
        size="sm"
        class="shrink-0"
        disabled={activating}
        onclick={(e) => {
          e.stopPropagation()
          onActivate?.()
        }}
      >
        <Zap class="mr-1 size-3" />
        {activating ? '激活中…' : '激活'}
      </Button>
    {/if}

    <DropdownMenu.Root>
      <DropdownMenu.Trigger>
        {#snippet child({ props })}
          <Button variant="ghost" size="icon-xs" {...props} onclick={(e) => e.stopPropagation()}>
            <Ellipsis class="size-3.5" />
            <span class="sr-only">More actions</span>
          </Button>
        {/snippet}
      </DropdownMenu.Trigger>
      <DropdownMenu.Content align="end" class="w-44">
        <DropdownMenu.Item onclick={() => onViewHarness?.()}>
          <Eye class="mr-2 size-3.5" />
          查看 Harness
        </DropdownMenu.Item>
        {#if recommendation.activation_ready && deferred}
          <DropdownMenu.Item onclick={() => onActivate?.()}>
            <Play class="mr-2 size-3.5" />
            激活
          </DropdownMenu.Item>
        {/if}
        <DropdownMenu.Separator />
        {#if deferred}
          <DropdownMenu.Item onclick={() => onRestore?.()}>
            <Undo2 class="mr-2 size-3.5" />
            重新显示
          </DropdownMenu.Item>
        {:else}
          <DropdownMenu.Item onclick={() => onDefer?.()}>
            <Clock class="mr-2 size-3.5" />
            稍后再说
          </DropdownMenu.Item>
        {/if}
      </DropdownMenu.Content>
    </DropdownMenu.Root>

    {#if expanded}
      <ChevronDown class="text-muted-foreground size-4 shrink-0" />
    {:else}
      <ChevronRight class="text-muted-foreground size-4 shrink-0" />
    {/if}
  </button>

  <!-- Expandable detail -->
  {#if expanded}
    <div class="border-border/60 space-y-3 border-t px-4 py-3">
      <p class="text-muted-foreground text-xs">{recommendation.summary}</p>

      <div class="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-[11px]">
        <span
          class={cn(
            'inline-flex items-center gap-1 rounded-full border px-2 py-0.5',
            workflowFamilyColors[workflowFamily],
          )}
        >
          <span>{workflowFamilyIcons[workflowFamily]}</span>
          <span>{recommendation.workflow_type}</span>
          <span class="opacity-80">/{recommendation.workflow_family}</span>
        </span>
        <span>{recommendation.suggested_workflow_name}</span>
        <span>{recommendation.suggested_workflow_type}</span>
        <span class="truncate">{recommendation.harness_path}</span>
      </div>

      <div class="text-xs">
        <span class="text-foreground font-medium">当前状态:</span>
        <span class="text-muted-foreground ml-1">{activationStatus}</span>
      </div>

      {#if recommendation.evidence.length > 0}
        <ul class="text-muted-foreground list-inside list-disc space-y-0.5 text-xs">
          {#each recommendation.evidence as evidence (evidence)}
            <li>{evidence}</li>
          {/each}
        </ul>
      {/if}

      {#if activationError}
        <div
          class="border-destructive/30 bg-destructive/10 text-destructive rounded-md border px-3 py-2 text-xs"
        >
          {activationError}
        </div>
      {/if}
    </div>
  {/if}
</article>
