<script lang="ts">
  import { Button } from '$ui/button'
  import { LoaderCircle, RefreshCw, Trash2 } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    configured = false,
    anyBusy = false,
    actionKey = '',
    onRetest,
    onDelete,
  }: {
    configured?: boolean
    anyBusy?: boolean
    actionKey?: string
    onRetest?: () => void
    onDelete?: () => void
  } = $props()
  const t = i18nStore.t
</script>

{#if configured}
  <div class="flex items-center gap-1">
    <Button
      variant="ghost"
      size="icon"
      class="size-7"
      onclick={onRetest}
      disabled={anyBusy}
      title={t('orgAdmin.credentials.actions.retest')}
    >
      {#if actionKey === 'retest'}
        <LoaderCircle class="size-3.5 animate-spin" />
      {:else}
        <RefreshCw class="size-3.5" />
      {/if}
    </Button>
    <Button
      variant="ghost"
      size="icon"
      class="text-destructive hover:text-destructive size-7"
      onclick={onDelete}
      disabled={anyBusy}
      title={t('orgAdmin.credentials.actions.delete')}
    >
      {#if actionKey === 'delete'}
        <LoaderCircle class="size-3.5 animate-spin" />
      {:else}
        <Trash2 class="size-3.5" />
      {/if}
    </Button>
  </div>
{/if}
