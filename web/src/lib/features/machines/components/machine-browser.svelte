<script lang="ts">
  import { Input } from '$ui/input'
  import { Search } from '@lucide/svelte'
  import MachineList from './machine-list.svelte'
  import type { MachineItem } from '../types'

  let {
    searchQuery = '',
    machines,
    selectedId = '',
    emptyMessage = 'No machines match the current filter.',
    onSearchChange,
    onSelect,
  }: {
    searchQuery?: string
    machines: MachineItem[]
    selectedId?: string
    emptyMessage?: string
    onSearchChange?: (value: string) => void
    onSelect?: (machineId: string) => void
  } = $props()
</script>

<section class="space-y-3">
  <div class="relative">
    <Search class="text-muted-foreground absolute top-2.5 left-2.5 size-3.5" />
    <Input
      value={searchQuery}
      class="h-9 pl-8 text-sm"
      placeholder="Search machines..."
      oninput={(event) => onSearchChange?.((event.currentTarget as HTMLInputElement).value)}
    />
  </div>

  <MachineList {machines} {selectedId} {onSelect} {emptyMessage} />
</section>
