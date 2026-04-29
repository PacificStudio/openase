<script lang="ts">
  import * as Dialog from '$ui/dialog'
  import { Button } from '$ui/button'
  import { chatT } from './i18n'

  let {
    open = false,
    fileName = '',
    saving = false,
    onDismiss,
    onDiscard,
    onSave,
  }: {
    open?: boolean
    fileName?: string
    saving?: boolean
    onDismiss: () => void
    onDiscard: () => void
    onSave: () => void
  } = $props()
</script>

<Dialog.Root
  {open}
  onOpenChange={(next) => {
    if (!next) {
      onDismiss()
    }
  }}
>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>{chatT('chat.saveChangesTitle')}</Dialog.Title>
      <Dialog.Description>
        {chatT('chat.saveChangesDescription', { fileName })}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Button variant="ghost" onclick={onDismiss} disabled={saving}>
        {chatT('common.cancel')}
      </Button>
      <Button variant="ghost" onclick={onDiscard} disabled={saving}>
        {chatT('chat.dontSave')}
      </Button>
      <Button onclick={onSave} disabled={saving}>
        {saving ? chatT('chat.saving') : chatT('chat.save')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
