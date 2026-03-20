<script lang="ts">
  import { cn } from '$lib/utils'
  import type { BoardColumn, BoardTicket } from '../types'
  import BoardColumnComponent from './board-column.svelte'

  let {
    columns,
    class: className = '',
    onticketclick,
  }: {
    columns: BoardColumn[]
    class?: string
    onticketclick?: (ticket: BoardTicket) => void
  } = $props()
</script>

<div class={cn('flex-1 overflow-x-auto', className)}>
  <div class="flex h-full gap-3 p-1">
    {#each columns as column (column.id)}
      <BoardColumnComponent {column} {onticketclick} />
    {/each}
  </div>
</div>
