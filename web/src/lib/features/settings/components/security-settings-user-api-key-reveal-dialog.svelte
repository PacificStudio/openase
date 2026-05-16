<script lang="ts">
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    open = $bindable(false),
    token,
    name,
    onCopy,
  }: {
    open?: boolean
    token: string
    name: string
    onCopy: () => void
  } = $props()
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-xl">
    <Dialog.Header>
      <Dialog.Title>{i18nStore.t('settings.security.userApiKeys.reveal.title')}</Dialog.Title>
      <Dialog.Description>
        {i18nStore.t('settings.security.userApiKeys.reveal.description', { name })}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Body class="space-y-3">
      <div class="bg-muted rounded-lg p-3 font-mono text-xs break-all">{token}</div>
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('settings.security.userApiKeys.reveal.warning')}
      </p>
    </Dialog.Body>
    <Dialog.Footer>
      <Button variant="outline" onclick={onCopy}
        >{i18nStore.t('settings.security.userApiKeys.buttons.copyToken')}</Button
      >
      <Dialog.Close>
        {#snippet child({ props })}
          <Button {...props}>{i18nStore.t('settings.security.userApiKeys.buttons.done')}</Button>
        {/snippet}
      </Dialog.Close>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
