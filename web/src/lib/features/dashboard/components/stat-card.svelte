<script lang="ts">
  import { cn } from '$lib/utils'
  import type { Component } from 'svelte'

  let {
    label,
    value,
    icon: Icon,
    trend,
    class: className = '',
  }: {
    label: string
    value: string | number
    icon?: Component
    trend?: { value: number; positive: boolean }
    class?: string
  } = $props()
</script>

<div class={cn('border-border bg-card rounded-md border p-4', className)}>
  <div class="flex items-center justify-between">
    <span class="text-muted-foreground text-xs">{label}</span>
    {#if Icon}
      <Icon class="text-muted-foreground size-4" />
    {/if}
  </div>
  <div class="text-foreground mt-2 text-2xl font-semibold">{value}</div>
  {#if trend}
    <div class={cn('mt-1 text-xs', trend.positive ? 'text-success' : 'text-destructive')}>
      {trend.positive ? '+' : ''}{trend.value}% from last week
    </div>
  {/if}
</div>
