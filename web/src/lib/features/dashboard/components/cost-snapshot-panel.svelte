<script lang="ts">
  import { cn, formatCount, formatCurrency } from '$lib/utils'
  import { Bot, Coins, DollarSign, ReceiptText, TrendingUp } from '@lucide/svelte'
  import type { DashboardUsageLeader } from '../types'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    ticketSpendToday,
    ticketSpendTotal,
    ticketInputTokens,
    ticketOutputTokens,
    agentLifetimeTokens,
    ticketsCreatedToday,
    ticketsCompletedToday,
    topCostTicket,
    topTokenAgent,
    class: className = '',
  }: {
    ticketSpendToday: number
    ticketSpendTotal: number
    ticketInputTokens: number
    ticketOutputTokens: number
    agentLifetimeTokens: number
    ticketsCreatedToday: number
    ticketsCompletedToday: number
    topCostTicket?: DashboardUsageLeader | null
    topTokenAgent?: DashboardUsageLeader | null
    class?: string
  } = $props()

  const totalTicketTokens = $derived(ticketInputTokens + ticketOutputTokens)
</script>

<div class={cn('border-border bg-card rounded-md border', className)}>
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <h3 class="text-foreground text-sm font-medium">
      {i18nStore.t('dashboard.costSnapshot.heading')}
    </h3>
    <Coins class="text-muted-foreground size-4" />
  </div>

  <div class="space-y-4 p-4">
    <div class="grid grid-cols-2 gap-3">
      <div class="bg-muted/40 rounded-md px-3 py-2">
        <div
          class="text-muted-foreground flex items-center gap-2 text-[11px] tracking-[0.12em] uppercase"
        >
          <DollarSign class="size-3" />
          <span>{i18nStore.t('dashboard.costSnapshot.stats.spendToday')}</span>
        </div>
        <p class="text-foreground mt-1 text-base font-semibold">
          {formatCurrency(ticketSpendToday)}
        </p>
      </div>
      <div class="bg-muted/40 rounded-md px-3 py-2">
        <div
          class="text-muted-foreground flex items-center gap-2 text-[11px] tracking-[0.12em] uppercase"
        >
          <ReceiptText class="size-3" />
          <span>{i18nStore.t('dashboard.costSnapshot.stats.ticketSpendTotal')}</span>
        </div>
        <p class="text-foreground mt-1 text-base font-semibold">
          {formatCurrency(ticketSpendTotal)}
        </p>
      </div>
      <div class="bg-muted/40 rounded-md px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
          {i18nStore.t('dashboard.costSnapshot.stats.ticketInputTokens')}
        </div>
        <p class="text-foreground mt-1 text-base font-semibold">{formatCount(ticketInputTokens)}</p>
      </div>
      <div class="bg-muted/40 rounded-md px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
          {i18nStore.t('dashboard.costSnapshot.stats.ticketOutputTokens')}
        </div>
        <p class="text-foreground mt-1 text-base font-semibold">
          {formatCount(ticketOutputTokens)}
        </p>
      </div>
    </div>

    <div class="bg-border h-px"></div>

    <div class="flex items-center justify-between gap-4">
      <div>
        <div class="text-muted-foreground text-xs">
          {i18nStore.t('dashboard.costSnapshot.stats.ticketScopedTokens')}
        </div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {formatCount(totalTicketTokens)}
        </div>
      </div>
      <div class="text-right">
        <div class="text-muted-foreground text-xs">
          {i18nStore.t('dashboard.costSnapshot.stats.agentLifetimeTokens')}
        </div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {formatCount(agentLifetimeTokens)}
        </div>
      </div>
    </div>

    <p class="text-muted-foreground text-xs">
      {i18nStore.t('dashboard.costSnapshot.description')}
    </p>

    <div class="bg-border h-px"></div>

    {#if topCostTicket}
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <TrendingUp class="text-muted-foreground size-3" />
          <span class="text-muted-foreground text-xs">
            {i18nStore.t('dashboard.costSnapshot.labels.topCostTicket')}
          </span>
        </div>
        <div class="text-right">
          <span class="text-foreground text-sm">{topCostTicket.name}</span>
          <span class="text-muted-foreground ml-2 text-xs">
            {formatCurrency(topCostTicket.value)}
          </span>
        </div>
      </div>
    {/if}

    {#if topTokenAgent}
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Bot class="text-muted-foreground size-3" />
          <span class="text-muted-foreground text-xs">
            {i18nStore.t('dashboard.costSnapshot.labels.topTokenAgent')}
          </span>
        </div>
        <div class="text-right">
          <span class="text-foreground text-sm">{topTokenAgent.name}</span>
          <span class="text-muted-foreground ml-2 text-xs">
            {formatCount(topTokenAgent.value)}
          </span>
        </div>
      </div>
    {/if}

    <div class="text-muted-foreground flex items-center justify-between text-xs">
      <span>
        {i18nStore.t('dashboard.costSnapshot.summary.ticketsCreated', {
          count: formatCount(ticketsCreatedToday),
        })}
      </span>
      <span>
        {i18nStore.t('dashboard.costSnapshot.summary.ticketsCompleted', {
          count: formatCount(ticketsCompletedToday),
        })}
      </span>
    </div>
  </div>
</div>
