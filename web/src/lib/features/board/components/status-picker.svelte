<script lang="ts">
  import * as Popover from '$ui/popover'
  import type { BoardStatusOption, BoardTicket } from '../types'
  import StageIcon from './stage-icon.svelte'

  let {
    ticket,
    statuses,
    disabled = false,
    onStatusChange,
  }: {
    ticket: BoardTicket
    statuses: BoardStatusOption[]
    disabled?: boolean
    onStatusChange?: (ticketId: string, statusId: string) => void
  } = $props()

  let open = $state(false)

  function handleSelect(statusId: string) {
    if (statusId === ticket.statusId) {
      open = false
      return
    }
    onStatusChange?.(ticket.id, statusId)
    open = false
  }
</script>

<Popover.Root bind:open>
  <Popover.Trigger
    class="hover:bg-muted inline-flex shrink-0 items-center justify-center rounded p-0.5 transition-colors"
    {disabled}
    onclick={(e: MouseEvent) => e.stopPropagation()}
    aria-label="Change status"
  >
    <StageIcon stage={ticket.stage} color={ticket.statusColor} />
  </Popover.Trigger>
  <Popover.Content
    align="start"
    class="w-48 gap-0 p-0.5"
    onclick={(e: MouseEvent) => e.stopPropagation()}
  >
    {#each statuses as status (status.id)}
      <button
        type="button"
        class="hover:bg-muted flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors {status.id ===
        ticket.statusId
          ? 'bg-muted'
          : ''}"
        onclick={() => handleSelect(status.id)}
      >
        <StageIcon stage={status.stage} color={status.color} />
        <span class="text-foreground truncate">{status.name}</span>
      </button>
    {/each}
  </Popover.Content>
</Popover.Root>
