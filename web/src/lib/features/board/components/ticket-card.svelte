<script lang="ts">
  import { cn, formatRelativeTime, truncate } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import {
    RotateCcw,
    ShieldAlert,
    Wallet,
    GripVertical,
    Cog,
    Loader,
    CircleX,
  } from '@lucide/svelte'
  import * as Tooltip from '$ui/tooltip'
  import type { BoardStatusOption, BoardTicket } from '../types'
  import type { BoardPriority } from '../priority'
  import StatusPicker from './status-picker.svelte'
  import PriorityPicker from './priority-picker.svelte'
  import TicketCardContextMenu from './ticket-card-context-menu.svelte'
  import TicketLinkBadges from './ticket-link-badges.svelte'

  let {
    ticket,
    statuses = [],
    class: className = '',
    onclick,
    ondragstartticket,
    ondragendticket,
    onStatusChange,
    onPriorityChange,
    onArchiveTicket,
    isDragging = false,
    isPendingMove = false,
  }: {
    ticket: BoardTicket
    statuses?: BoardStatusOption[]
    class?: string
    onclick?: (ticket: BoardTicket) => void
    ondragstartticket?: (ticket: BoardTicket) => void
    ondragendticket?: () => void
    onStatusChange?: (ticketId: string, statusId: string) => void
    onPriorityChange?: (ticketId: string, priority: BoardTicket['priority']) => void
    onArchiveTicket?: (ticketId: string) => void
    isDragging?: boolean
    isPendingMove?: boolean
  } = $props()

  const workflowColors: Record<string, string> = {
    coding: 'bg-violet-500/15 text-violet-600 dark:text-violet-400',
    test: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400',
    security: 'bg-red-500/15 text-red-600 dark:text-red-400',
    review: 'bg-blue-500/15 text-blue-600 dark:text-blue-400',
    deploy: 'bg-amber-500/15 text-amber-600 dark:text-amber-400',
  }

  const anomalyConfig: Record<
    string,
    { label: string; variant: 'destructive' | 'secondary'; icon: typeof RotateCcw }
  > = {
    retry: { label: 'Retry paused', variant: 'secondary', icon: RotateCcw },
    awaiting_approval: { label: 'Needs approval', variant: 'secondary', icon: ShieldAlert },
    budget_exhausted: { label: 'Budget exhausted', variant: 'destructive', icon: Wallet },
  }

  let suppressClickUntil = 0
  let contextMenuOpen = $state(false)

  function handleClick() {
    if (Date.now() < suppressClickUntil || isDragging || contextMenuOpen) {
      return
    }
    onclick?.(ticket)
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      handleClick()
    }
    if (e.shiftKey && e.key === 'F10') {
      e.preventDefault()
      openContextMenuFromElement(e.currentTarget)
    }
    if (e.key === 'ContextMenu') {
      e.preventDefault()
      openContextMenuFromElement(e.currentTarget)
    }
  }

  function openContextMenuFromElement(target: EventTarget | null) {
    if (!(target instanceof HTMLElement) || isDragging || isPendingMove) {
      return
    }

    const rect = target.getBoundingClientRect()
    target.dispatchEvent(
      new MouseEvent('contextmenu', {
        bubbles: true,
        cancelable: true,
        clientX: rect.left + rect.width / 2,
        clientY: rect.top + rect.height / 2,
      }),
    )
  }

  function handleDragStart(event: DragEvent) {
    if (isPendingMove || contextMenuOpen) {
      event.preventDefault()
      return
    }

    suppressClickUntil = Date.now() + 250
    event.dataTransfer?.setData('application/x-openase-ticket-id', ticket.id)
    event.dataTransfer?.setData('text/plain', ticket.id)
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move'
    }
    ondragstartticket?.(ticket)
  }

  function handleDragEnd() {
    suppressClickUntil = Date.now() + 250
    ondragendticket?.()
  }

  function handleOpenDetails(t: BoardTicket) {
    onclick?.(t)
  }

  function handleContextPriorityChange(ticketId: string, priority: BoardPriority) {
    onPriorityChange?.(ticketId, priority)
  }
</script>

<TicketCardContextMenu
  {ticket}
  {statuses}
  disabled={isDragging || isPendingMove}
  {isPendingMove}
  bind:open={contextMenuOpen}
  onOpenDetails={handleOpenDetails}
  {onStatusChange}
  onPriorityChange={handleContextPriorityChange}
  onArchive={onArchiveTicket}
>
  <button
    type="button"
    draggable={!isPendingMove && !contextMenuOpen}
    class={cn(
      'border-border bg-card w-full shrink-0 rounded-md border p-2.5 text-left',
      'cursor-grab active:cursor-grabbing',
      'hover:border-border/80 hover:bg-accent/50 transition-colors',
      'focus-visible:ring-ring focus-visible:ring-2 focus-visible:outline-none',
      isDragging && 'border-primary/60 bg-primary/5 opacity-70 shadow-sm',
      isPendingMove && 'cursor-progress opacity-80',
      className,
    )}
    aria-grabbed={isDragging}
    disabled={isPendingMove}
    onclick={handleClick}
    onkeydown={handleKeydown}
    ondragstart={handleDragStart}
    ondragend={handleDragEnd}
  >
    <div class="flex items-start gap-2">
      <div class="relative mt-0.5 shrink-0">
        <StatusPicker {ticket} {statuses} disabled={isPendingMove} {onStatusChange} />
        {#if ticket.isBlocked}
          <svg
            viewBox="0 0 16 16"
            class="absolute right-0 bottom-1 size-2.5 text-red-500"
            role="img"
            aria-label="Blocked"
          >
            <polygon points="5,1 11,1 15,5 15,11 11,15 5,15 1,11 1,5" fill="currentColor" />
            <rect x="4" y="7" width="8" height="2" rx="0.5" fill="white" />
          </svg>
        {/if}
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-1.5">
          <span class="text-muted-foreground text-xs font-medium">{ticket.identifier}</span>
          {#if isPendingMove}
            <span class="text-muted-foreground text-[10px]">Moving…</span>
          {/if}
          <div
            class="text-muted-foreground/50 ml-auto flex items-center"
            title="Right-click for actions"
          >
            <GripVertical class="text-muted-foreground/50 size-3.5 shrink-0" />
          </div>
        </div>
        <p class="text-foreground mt-0.5 text-sm leading-snug font-medium">
          {truncate(ticket.title, 60)}
        </p>
      </div>
    </div>

    <div class="mt-2 flex flex-wrap items-center gap-1.5">
      <PriorityPicker {ticket} disabled={isPendingMove} {onPriorityChange} />
      <TicketLinkBadges links={ticket.externalLinks} pullRequestURLs={ticket.pullRequestURLs} />

      {#if ticket.workflowType}
        <span
          class={cn(
            'inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium',
            workflowColors[ticket.workflowType] ?? 'bg-muted text-muted-foreground',
          )}
        >
          {ticket.workflowType}
        </span>
      {/if}

      {#if ticket.anomaly}
        {@const config = anomalyConfig[ticket.anomaly]}
        {#if config}
          <Badge variant={config.variant} class="h-4 gap-1 text-[10px]">
            <config.icon class="size-2.5" />
            {config.label}
          </Badge>
        {/if}
      {/if}

      {#if ticket.runtimePhase === 'executing'}
        <span class="inline-flex items-center text-emerald-500" title="Executing">
          <Cog class="size-3 animate-spin" />
        </span>
      {:else if ticket.runtimePhase === 'ready'}
        <span class="inline-flex items-center text-emerald-500" title="Ready">
          <Cog class="size-3" />
        </span>
      {:else if ticket.runtimePhase === 'launching'}
        <span class="inline-flex items-center text-amber-500" title="Launching">
          <Loader class="size-3 animate-spin [animation-duration:2s]" />
        </span>
      {:else if ticket.runtimePhase === 'failed'}
        {#if ticket.lastError}
          <Tooltip.Provider>
            <Tooltip.Root>
              <Tooltip.Trigger class="inline-flex items-center text-red-500">
                <CircleX class="size-3" />
              </Tooltip.Trigger>
              <Tooltip.Portal>
                <Tooltip.Content
                  side="top"
                  class="bg-popover text-popover-foreground max-w-64 rounded-md border px-3 py-2 text-xs shadow-md"
                >
                  {ticket.lastError}
                </Tooltip.Content>
              </Tooltip.Portal>
            </Tooltip.Root>
          </Tooltip.Provider>
        {:else}
          <span class="inline-flex items-center text-red-500" title="Failed">
            <CircleX class="size-3" />
          </span>
        {/if}
      {/if}
    </div>

    <div class="text-muted-foreground/70 mt-1.5 text-[10px]">
      {formatRelativeTime(ticket.updatedAt)}
    </div>
  </button>
</TicketCardContextMenu>
