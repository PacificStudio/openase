<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import { archiveOrganization } from '$lib/api/openase'
  import { i18nStore } from '$lib/i18n/store.svelte'
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

  const t = i18nStore.t

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
          first.reason instanceof ApiError && 'detail' in first.reason && first.reason.detail
            ? t('catalog.organization.bulkArchive.errors.partialWithDetails', {
                count: failures.length,
                message: first.reason.detail,
              })
            : t('catalog.organization.bulkArchive.errors.partial', {
                count: failures.length,
              }),
        )
      }

      onClear?.()
      await invalidateAll()
    } catch {
      toastStore.error(t('catalog.organization.bulkArchive.errors.generic'))
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
      {t('catalog.organization.bulkArchive.selected', { count: selectedCount })}
    </span>
    <div class="flex items-center gap-2">
      <Button variant="ghost" size="sm" onclick={onClear}>
        {t('catalog.organization.bulkArchive.actions.cancel')}
      </Button>
      <Button variant="destructive" size="sm" disabled={archiving} onclick={handleArchive}>
        {archiving
          ? t('catalog.organization.bulkArchive.actions.archiving')
          : t('catalog.organization.bulkArchive.actions.archive', {
              count: selectedCount,
            })}
      </Button>
    </div>
  </div>
{/if}
