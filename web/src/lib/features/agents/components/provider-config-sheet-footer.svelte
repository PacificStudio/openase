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
    {deleting ? 'Deleting…' : 'Delete provider'}
  </Button>
  <Button variant="outline" onclick={onCancel} disabled={saving || deleting}>
    {i18nStore.t('common.cancel')}
  </Button>
  <Button onclick={onSave} disabled={saving || deleting}>
    {saving ? 'Saving…' : 'Save changes'}
  </Button>
</SheetFooter>
