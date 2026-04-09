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
      toastStore.error(formatError(caughtError, 'Failed to load organization secrets.'))
      secrets = []
    } finally {
      loading = false
    }
  }

  async function handleCreate() {
    const name = createDraft.name.trim()
    const value = createDraft.value.trim()
    if (!name) {
      toastStore.error('Secret name is required.')
      return
    }
    if (!value) {
      toastStore.error('Secret value is required.')
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
      toastStore.success('Created organization secret.')
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(formatError(caughtError, 'Failed to create organization secret.'))
    } finally {
      creating = false
    }
  }

  async function handleRotate(secret: ScopedSecretRecord, value: string) {
    try {
      await rotateOrganizationScopedSecret(organizationId, secret.id, { value })
      toastStore.success(`Rotated ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(formatError(caughtError, 'Failed to rotate organization secret.'))
      throw caughtError
    }
  }

  async function handleDisable(secret: ScopedSecretRecord) {
    try {
      await disableOrganizationScopedSecret(organizationId, secret.id)
      toastStore.success(`Disabled ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(formatError(caughtError, 'Failed to disable organization secret.'))
      throw caughtError
    }
  }

  async function handleDelete(secret: ScopedSecretRecord) {
    try {
      await deleteOrganizationScopedSecret(organizationId, secret.id)
      toastStore.success(`Deleted ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(formatError(caughtError, 'Failed to delete organization secret.'))
      throw caughtError
    }
  }
</script>

<div class="space-y-3">
  <div class="flex items-center justify-between gap-4">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Organization secrets</h3>
      <p class="text-muted-foreground mt-0.5 text-xs">
        Central secrets inherit into every project until a project overrides the same key locally.
      </p>
    </div>
    <Button variant="outline" size="sm" onclick={() => (createDialogOpen = true)}>
      Add secret
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
      <Dialog.Title>Add organization secret</Dialog.Title>
      <Dialog.Description>
        New values become the default binding for every project in this org. Values are masked after
        write and never shown again.
      </Dialog.Description>
    </Dialog.Header>

    <div class="space-y-4">
      <div class="space-y-1.5">
        <Label for="org-secret-name">Secret name</Label>
        <Input id="org-secret-name" bind:value={createDraft.name} placeholder="GH_TOKEN" />
      </div>
      <div class="space-y-1.5">
        <Label for="org-secret-value">Secret value</Label>
        <Input
          id="org-secret-value"
          type="password"
          bind:value={createDraft.value}
          placeholder="Paste the secret value"
        />
      </div>
      <div class="space-y-1.5">
        <Label for="org-secret-description"
          >Description <span class="text-muted-foreground font-normal">(optional)</span></Label
        >
        <Textarea
          id="org-secret-description"
          bind:value={createDraft.description}
          rows={3}
          placeholder="What this secret powers, who owns it, and when to rotate it."
        />
      </div>
    </div>

    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={creating}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={handleCreate} disabled={creating}>
        {creating ? 'Creating…' : 'Create secret'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
