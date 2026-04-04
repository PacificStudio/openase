<script lang="ts">
  import * as Popover from '$ui/popover'
  import { formatBoardPriorityLabel, type BoardPriority } from '../priority'
  import type { BoardTicket } from '../types'
  import PriorityIcon from './priority-icon.svelte'

  type Priority = BoardPriority

  const options: Array<{ value: Priority; label: string }> = [
    { value: '', label: formatBoardPriorityLabel('') },
    { value: 'urgent', label: formatBoardPriorityLabel('urgent') },
    { value: 'high', label: formatBoardPriorityLabel('high') },
    { value: 'medium', label: formatBoardPriorityLabel('medium') },
    { value: 'low', label: formatBoardPriorityLabel('low') },
  ]

  let {
    ticket,
    disabled = false,
    onPriorityChange,
  }: {
    ticket: BoardTicket
    disabled?: boolean
    onPriorityChange?: (ticketId: string, priority: Priority) => void
  } = $props()

  let open = $state(false)

  function handleSelect(priority: Priority) {
    if (priority === ticket.priority) {
      open = false
      return
    }
    onPriorityChange?.(ticket.id, priority)
    open = false
  }
</script>

<Popover.Root bind:open>
  <Popover.Trigger
    class="hover:bg-muted inline-flex shrink-0 items-center justify-center rounded p-0.5 transition-colors"
    {disabled}
    onclick={(e: MouseEvent) => e.stopPropagation()}
    aria-label="Change priority"
  >
    <PriorityIcon priority={ticket.priority} />
  </Popover.Trigger>
  <Popover.Content
    align="start"
    class="w-36 gap-0 p-0.5"
    onclick={(e: MouseEvent) => e.stopPropagation()}
  >
    {#each options as option (option.value)}
      <button
        type="button"
        class="hover:bg-muted flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors {option.value ===
        ticket.priority
          ? 'bg-muted'
          : ''}"
        onclick={() => handleSelect(option.value)}
      >
        <PriorityIcon priority={option.value} />
        <span class="text-foreground">{option.label}</span>
      </button>
    {/each}
  </Popover.Content>
</Popover.Root>
