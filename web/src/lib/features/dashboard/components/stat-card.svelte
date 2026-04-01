<script lang="ts">
  import { cn } from '$lib/utils'
  import { Skeleton } from '$ui/skeleton'
  import type { Component } from 'svelte'

  let {
    label,
    value,
    icon: Icon,
    trend,
    loading = false,
    class: className = '',
  }: {
    label: string
    value: string | number
    icon?: Component
    trend?: { value: number; positive: boolean }
    loading?: boolean
    class?: string
  } = $props()
</script>

<div class={cn('border-border bg-card hover-lift rounded-md border p-4', className)}>
  <div class="flex items-center justify-between">
    <span class="text-muted-foreground text-xs">{label}</span>
    {#if Icon}
      <Icon class="text-muted-foreground size-4" />
    {/if}
  </div>
  {#if loading}
    <Skeleton class="mt-2 h-8 w-16" />
  {:else}
    {#key value}
      <div class="text-foreground animate-count-pop mt-2 text-2xl font-semibold">{value}</div>
    {/key}
  {/if}
  {#if trend && !loading}
    <div class={cn('mt-1 text-xs', trend.positive ? 'text-success' : 'text-destructive')}>
      {trend.positive ? '+' : ''}{trend.value}% from last week
    </div>
  {/if}
</div>
