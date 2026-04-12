<script lang="ts">
  import { Button } from '$ui/button'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    archiving = false,
    onArchive,
  }: {
    archiving?: boolean
    onArchive?: () => void
  } = $props()
</script>

<div class="border-destructive/30 bg-destructive/5 rounded-lg border p-4">
  <div class="space-y-2">
    <h3 class="text-foreground text-sm font-medium">
      {i18nStore.t('settings.archive.heading')}
    </h3>
    <p class="text-muted-foreground text-sm">
      {i18nStore.t('settings.archive.description')}
    </p>
  </div>

  <div class="mt-4 flex justify-start">
    <Button variant="destructive" onclick={onArchive} disabled={archiving}>
      {archiving
        ? i18nStore.t('settings.archive.button.archiving')
        : i18nStore.t('settings.archive.button.archive')}
    </Button>
  </div>
</div>
