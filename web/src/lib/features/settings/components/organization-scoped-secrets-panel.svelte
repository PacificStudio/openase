<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { ScopedSecretRecord } from '$lib/api/contracts'
  import {
    createOrganizationScopedSecret,
    deleteOrganizationScopedSecret,
    disableOrganizationScopedSecret,
    listOrganizationScopedSecrets,
    rotateOrganizationScopedSecret,
  } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import OrganizationScopedSecretsInventoryCard from './organization-scoped-secrets-inventory-card.svelte'

  let { organizationId }: { organizationId: string } = $props()

  let loading = $state(false)
  let creating = $state(false)
  let createDialogOpen = $state(false)
  let secrets = $state<ScopedSecretRecord[]>([])
  let createDraft = $state({ name: '', description: '', value: '' })

  $effect(() => {
    if (!organizationId) {
      secrets = []
      return
    }
    void loadSecrets()
  })

  function formatError(caughtError: unknown, fallback: string) {
    return caughtError instanceof ApiError ? caughtError.detail : fallback
  }

  async function loadSecrets() {
    loading = true
    try {
      const payload = await listOrganizationScopedSecrets(organizationId)
      secrets = payload.secrets ?? []
    } catch (caughtError) {
      toastStore.error(
        formatError(caughtError, i18nStore.t('settings.organizationSecrets.panel.errors.load')),
      )
      secrets = []
    } finally {
      loading = false
    }
  }

  async function handleCreate() {
    const name = createDraft.name.trim()
    const value = createDraft.value.trim()
    if (!name) {
      toastStore.error(
        i18nStore.t('settings.organizationSecrets.panel.errors.secretNameRequired'),
      )
      return
    }
    if (!value) {
      toastStore.error(
        i18nStore.t('settings.organizationSecrets.panel.errors.secretValueRequired'),
      )
      return
    }

    creating = true
    try {
      await createOrganizationScopedSecret(organizationId, {
        name,
        description: createDraft.description.trim(),
        value,
      })
      createDraft = { name: '', description: '', value: '' }
      createDialogOpen = false
      toastStore.success(
        i18nStore.t('settings.organizationSecrets.panel.messages.created'),
      )
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(
        formatError(caughtError, i18nStore.t('settings.organizationSecrets.panel.errors.create')),
      )
    } finally {
      creating = false
    }
  }

  async function handleRotate(secret: ScopedSecretRecord, value: string) {
    try {
      await rotateOrganizationScopedSecret(organizationId, secret.id, { value })
      toastStore.success(
        i18nStore.t('settings.organizationSecrets.panel.messages.rotated', {
          secret: secret.name,
        }),
      )
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(
        formatError(caughtError, i18nStore.t('settings.organizationSecrets.panel.errors.rotate')),
      )
      throw caughtError
    }
  }

  async function handleDisable(secret: ScopedSecretRecord) {
    try {
      await disableOrganizationScopedSecret(organizationId, secret.id)
      toastStore.success(
        i18nStore.t('settings.organizationSecrets.panel.messages.disabled', {
          secret: secret.name,
        }),
      )
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(
        formatError(caughtError, i18nStore.t('settings.organizationSecrets.panel.errors.disable')),
      )
      throw caughtError
    }
  }

  async function handleDelete(secret: ScopedSecretRecord) {
    try {
      await deleteOrganizationScopedSecret(organizationId, secret.id)
      toastStore.success(
        i18nStore.t('settings.organizationSecrets.panel.messages.deleted', {
          secret: secret.name,
        }),
      )
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(
        formatError(caughtError, i18nStore.t('settings.organizationSecrets.panel.errors.delete')),
      )
      throw caughtError
    }
  }
</script>

<div class="space-y-3">
  <div class="flex items-center justify-between gap-4">
    <div>
      <h3 class="text-foreground text-sm font-semibold">
        {i18nStore.t('settings.organizationSecrets.panel.heading')}
      </h3>
      <p class="text-muted-foreground mt-0.5 text-xs">
        {i18nStore.t('settings.organizationSecrets.panel.description')}
      </p>
    </div>
    <Button variant="outline" size="sm" onclick={() => (createDialogOpen = true)}>
      {i18nStore.t('settings.organizationSecrets.panel.buttons.addSecret')}
    </Button>
  </div>

  <OrganizationScopedSecretsInventoryCard
    {loading}
    {secrets}
    onRotate={handleRotate}
    onDisable={handleDisable}
    onDelete={handleDelete}
  />
</div>

<!-- Create secret dialog -->
<Dialog.Root bind:open={createDialogOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>
        {i18nStore.t('settings.organizationSecrets.panel.dialogs.title')}
      </Dialog.Title>
      <Dialog.Description>
        {i18nStore.t('settings.organizationSecrets.panel.dialogs.description')}
      </Dialog.Description>
    </Dialog.Header>

    <div class="space-y-4">
      <div class="space-y-1.5">
        <Label for="org-secret-name">
          {i18nStore.t('settings.organizationSecrets.panel.labels.secretName')}
        </Label>
        <Input
          id="org-secret-name"
          bind:value={createDraft.name}
          placeholder={i18nStore.t('settings.organizationSecrets.panel.placeholders.secretName')}
        />
      </div>
      <div class="space-y-1.5">
        <Label for="org-secret-value">
          {i18nStore.t('settings.organizationSecrets.panel.labels.secretValue')}
        </Label>
        <Input
          id="org-secret-value"
          type="password"
          bind:value={createDraft.value}
          placeholder={i18nStore.t('settings.organizationSecrets.panel.placeholders.secretValue')}
        />
      </div>
      <div class="space-y-1.5">
        <Label for="org-secret-description">
          {i18nStore.t('settings.organizationSecrets.panel.labels.description')}
          <span class="text-muted-foreground font-normal">
            {i18nStore.t('settings.organizationSecrets.panel.labels.optional')}
          </span>
        </Label>
        <Textarea
          id="org-secret-description"
          bind:value={createDraft.description}
          rows={3}
          placeholder={i18nStore.t('settings.organizationSecrets.panel.placeholders.description')}
        />
      </div>
    </div>

    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={creating}>
            {i18nStore.t('settings.organizationSecrets.panel.buttons.cancel')}
          </Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={handleCreate} disabled={creating}>
        {creating
          ? i18nStore.t('settings.organizationSecrets.panel.buttons.creating')
          : i18nStore.t('settings.organizationSecrets.panel.buttons.createSecret')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
