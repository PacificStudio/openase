<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { FolderLock } from '@lucide/svelte'
</script>

<div class="border-border bg-card space-y-3 rounded-lg border p-4">
  <div class="flex items-center gap-2">
    <FolderLock class="text-muted-foreground size-4" />
    <div class="text-sm font-semibold">
      {i18nStore.t('settings.access.disabled.localBootstrapTitle')}
    </div>
  </div>
  <div class="text-muted-foreground space-y-2 text-sm leading-6">
    <p>{i18nStore.t('settings.access.disabled.messages.description')}</p>
    <p>{i18nStore.t('settings.access.disabled.messages.migration')}</p>
  </div>
</div>
