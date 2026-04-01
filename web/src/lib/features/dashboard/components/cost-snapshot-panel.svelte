<script lang="ts">
  import { cn, formatCount, formatCurrency } from '$lib/utils'
  import { Bot, Coins, DollarSign, ReceiptText, TrendingUp } from '@lucide/svelte'
  import type { DashboardUsageLeader } from '../types'

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
    <h3 class="text-foreground text-sm font-medium">Usage Snapshot</h3>
    <Coins class="text-muted-foreground size-4" />
  </div>

  <div class="space-y-4 p-4">
    <div class="grid grid-cols-2 gap-3">
      <div class="bg-muted/40 rounded-md px-3 py-2">
        <div
          class="text-muted-foreground flex items-center gap-2 text-[11px] tracking-[0.12em] uppercase"
        >
          <DollarSign class="size-3" />
          <span>Spend Today</span>
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
          <span>Ticket Spend Total</span>
        </div>
        <p class="text-foreground mt-1 text-base font-semibold">
          {formatCurrency(ticketSpendTotal)}
        </p>
      </div>
      <div class="bg-muted/40 rounded-md px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
          Ticket Input Tokens
        </div>
        <p class="text-foreground mt-1 text-base font-semibold">{formatCount(ticketInputTokens)}</p>
      </div>
      <div class="bg-muted/40 rounded-md px-3 py-2">
        <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
          Ticket Output Tokens
        </div>
        <p class="text-foreground mt-1 text-base font-semibold">
          {formatCount(ticketOutputTokens)}
        </p>
      </div>
    </div>

    <div class="bg-border h-px"></div>

    <div class="flex items-center justify-between gap-4">
      <div>
        <div class="text-muted-foreground text-xs">Ticket-scoped tokens</div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {formatCount(totalTicketTokens)}
        </div>
      </div>
      <div class="text-right">
        <div class="text-muted-foreground text-xs">Agent lifetime tokens</div>
        <div class="text-foreground mt-1 text-lg font-semibold">
          {formatCount(agentLifetimeTokens)}
        </div>
      </div>
    </div>

    <p class="text-muted-foreground text-xs">
      Ticket counters come from ticket usage records. Agent counters are lifetime per-agent totals
      and may not reconcile 1:1.
    </p>

    <div class="bg-border h-px"></div>

    {#if topCostTicket}
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <TrendingUp class="text-muted-foreground size-3" />
          <span class="text-muted-foreground text-xs">Top cost ticket</span>
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
          <span class="text-muted-foreground text-xs">Top lifetime-token agent</span>
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
      <span>{formatCount(ticketsCreatedToday)} tickets created today</span>
      <span>{formatCount(ticketsCompletedToday)} completed today</span>
    </div>
  </div>
</div>
