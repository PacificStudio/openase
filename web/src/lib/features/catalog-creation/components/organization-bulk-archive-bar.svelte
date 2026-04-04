<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import { archiveOrganization } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'

  let {
    selectedIds,
    onClear,
  }: {
    selectedIds: string[]
    onClear?: () => void
  } = $props()

  let archiving = $state(false)

  const selectedCount = $derived(selectedIds.length)

  async function handleArchive() {
    if (selectedCount === 0) return

    archiving = true

    try {
      const results = await Promise.allSettled(selectedIds.map((id) => archiveOrganization(id)))
      const failures = results.filter((result) => result.status === 'rejected')

      if (failures.length > 0) {
        const first = failures[0] as PromiseRejectedResult
        toastStore.error(
          first.reason instanceof ApiError
            ? `${failures.length} failed: ${first.reason.detail}`
            : `${failures.length} organization archive(s) failed.`,
        )
      }

      onClear?.()
      await invalidateAll()
    } catch {
      toastStore.error('Bulk archive failed.')
    } finally {
      archiving = false
    }
  }
</script>

{#if selectedCount > 0}
  <div
    class="bg-muted/60 border-border flex items-center justify-between rounded-lg border px-4 py-3"
  >
    <span class="text-foreground text-sm font-medium">
      {selectedCount} selected
    </span>
    <div class="flex items-center gap-2">
      <Button variant="ghost" size="sm" onclick={onClear}>Cancel</Button>
      <Button variant="destructive" size="sm" disabled={archiving} onclick={handleArchive}>
        {archiving ? 'Archiving...' : `Archive ${selectedCount}`}
      </Button>
    </div>
  </div>
{/if}
