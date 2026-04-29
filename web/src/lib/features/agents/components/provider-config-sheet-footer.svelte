<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Button } from '$ui/button'
  import { SheetFooter } from '$ui/sheet'

  let {
    saving = false,
    deleting = false,
    onCancel,
    onSave,
    onDelete,
  }: {
    saving?: boolean
    deleting?: boolean
    onCancel?: () => void
    onSave?: () => void
    onDelete?: () => void
  } = $props()
</script>

<SheetFooter class="border-border border-t px-6 py-4">
  <Button variant="outline" onclick={onDelete} disabled={saving || deleting}>
    {deleting
      ? i18nStore.t('agents.providerConfig.actions.deleting')
      : i18nStore.t('agents.providerConfig.actions.delete')}
  </Button>
  <Button variant="outline" onclick={onCancel} disabled={saving || deleting}>
    {i18nStore.t('common.cancel')}
  </Button>
  <Button onclick={onSave} disabled={saving || deleting}>
    {saving
      ? i18nStore.t('agents.providerConfig.actions.saving')
      : i18nStore.t('agents.providerConfig.actions.save')}
  </Button>
</SheetFooter>
