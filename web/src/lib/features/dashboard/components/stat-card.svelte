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
    trend?: { value: number, positive: boolean }
    class?: string
  } = $props()
</script>

<div class={cn('rounded-md border border-border bg-card p-4', className)}>
  <div class="flex items-center justify-between">
    <span class="text-xs text-muted-foreground">{label}</span>
    {#if Icon}
      <Icon class="size-4 text-muted-foreground" />
    {/if}
  </div>
  <div class="mt-2 text-2xl font-semibold text-foreground">{value}</div>
  {#if trend}
    <div class={cn('mt-1 text-xs', trend.positive ? 'text-success' : 'text-destructive')}>
      {trend.positive ? '+' : ''}{trend.value}% from last week
    </div>
  {/if}
</div>
