<script lang="ts">
  import { formatRelativeTime } from '$lib/utils'
  import * as Select from '$ui/select'
  import type { WorkflowSummary } from '../types'

  let { workflow }: { workflow: WorkflowSummary } = $props()

  let selectedVersion = $derived(String(workflow.version))

  const selectedItem = $derived(
    workflow.history.find((item) => String(item.version) === selectedVersion),
  )
</script>

{#if workflow.history.length > 0}
  <div class="flex items-center gap-2 px-4 py-2">
    <span class="text-muted-foreground text-[10px] font-medium tracking-wide uppercase"
      >v{workflow.version}</span
    >
    <Select.Root type="single" value={selectedVersion} onValueChange={(v) => (selectedVersion = v)}>
      <Select.Trigger
        class="h-6 w-auto gap-1 border-none bg-transparent px-1.5 text-[11px] shadow-none"
      >
        {#if selectedItem}
          v{selectedItem.version}{selectedItem.version === workflow.version ? ' · current' : ''}
          <span class="text-muted-foreground">{formatRelativeTime(selectedItem.createdAt)}</span>
        {:else}
          Select version
        {/if}
      </Select.Trigger>
      <Select.Content class="max-h-48 w-56">
        {#each workflow.history as item (item.id)}
          <Select.Item value={String(item.version)}>
            <div class="flex w-full items-center gap-2">
              <span class="text-foreground font-medium">v{item.version}</span>
              {#if item.version === workflow.version}
                <span class="bg-primary/10 text-primary rounded px-1 text-[9px]">current</span>
              {/if}
              <span class="text-muted-foreground ml-auto text-[10px]"
                >{formatRelativeTime(item.createdAt)}</span
              >
            </div>
          </Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
  </div>
{/if}
