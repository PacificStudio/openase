<script lang="ts">
  import type { OrgGitHubCredentialResponse } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { i18nStore } from '$lib/i18n/store.svelte'

  type Credential = OrgGitHubCredentialResponse['credential']

  let {
    credential,
    saveDialogOpen,
    deleteDialogOpen,
    tokenDraft,
    anyBusy,
    actionKey,
    onSaveOpenChange,
    onDeleteOpenChange,
    onTokenDraftChange,
    onSave,
    onDelete,
  }: {
    credential: Credential | null
    saveDialogOpen: boolean
    deleteDialogOpen: boolean
    tokenDraft: string
    anyBusy: boolean
    actionKey: string
    onSaveOpenChange: (open: boolean) => void
    onDeleteOpenChange: (open: boolean) => void
    onTokenDraftChange: (value: string) => void
    onSave: () => void
    onDelete: () => void
  } = $props()
  const t = i18nStore.t
</script>

<Dialog.Root open={saveDialogOpen} onOpenChange={onSaveOpenChange}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>
        {credential?.configured
          ? t('orgAdmin.credentials.dialog.title.rotate')
          : t('orgAdmin.credentials.dialog.title.save')}
      </Dialog.Title>
      <Dialog.Description>
        {#if credential?.configured}
          {t('orgAdmin.credentials.dialog.description.rotate')}
        {:else}
          {t('orgAdmin.credentials.dialog.description.save.prefix')}
          <code>ghu_xxx</code>
          {i18nStore.locale === 'zh' ? ' 或 ' : ' or '}
          <code>github_pat_xxx</code>
          {t('orgAdmin.credentials.dialog.description.save.suffix')}
        {/if}
      </Dialog.Description>
    </Dialog.Header>

    <div class="space-y-1.5">
      <Label for="credentials-token">
        {t('orgAdmin.credentials.dialog.label.token')}
      </Label>
      <Input
        id="credentials-token"
        type="password"
        value={tokenDraft}
        placeholder={t('orgAdmin.credentials.dialog.placeholders.token')}
        disabled={anyBusy}
        oninput={(event) => onTokenDraftChange(event.currentTarget.value)}
        onkeydown={(event) => {
          if (event.key === 'Enter') onSave()
        }}
      />
    </div>

    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={anyBusy}>
            {t('orgAdmin.credentials.dialog.actions.cancel')}
          </Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={onSave} disabled={anyBusy || !tokenDraft.trim()}>
        {actionKey === 'save'
          ? t('orgAdmin.credentials.dialog.actions.saving')
          : credential?.configured
            ? t('orgAdmin.credentials.dialog.actions.rotate')
            : t('orgAdmin.credentials.dialog.actions.save')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root open={deleteDialogOpen} onOpenChange={onDeleteOpenChange}>
  <Dialog.Content class="sm:max-w-sm">
    <Dialog.Header>
      <Dialog.Title>
        {t('orgAdmin.credentials.dialog.deleteTitle')}
      </Dialog.Title>
      <Dialog.Description>
        {t('orgAdmin.credentials.dialog.deleteDescription')}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={anyBusy}>
            {t('orgAdmin.credentials.dialog.actions.cancel')}
          </Button>
        {/snippet}
      </Dialog.Close>
      <Button variant="destructive" onclick={onDelete} disabled={anyBusy}>
        {actionKey === 'delete'
          ? t('orgAdmin.credentials.dialog.actions.deleting')
          : t('orgAdmin.credentials.dialog.actions.delete')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
