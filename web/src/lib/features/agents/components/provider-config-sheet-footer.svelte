<script lang="ts">
  import { Button } from '$ui/button'
  import { SheetFooter } from '$ui/sheet'

  let {
    saving = false,
    onCancel,
    onSave,
  }: {
    saving?: boolean
    onCancel?: () => void
    onSave?: () => void
  } = $props()
</script>

<SheetFooter class="border-border border-t px-6 py-4">
  <Button variant="outline" onclick={onCancel} disabled={saving}>Cancel</Button>
  <Button onclick={onSave} disabled={saving}>
    {saving ? 'Saving…' : 'Save changes'}
  </Button>
</SheetFooter>
