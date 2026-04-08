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
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import OrganizationScopedSecretsInventoryCard from './organization-scoped-secrets-inventory-card.svelte'

  let { organizationId }: { organizationId: string } = $props()

  let loading = $state(false)
  let creating = $state(false)
  let secrets = $state<ScopedSecretRecord[]>([])
  let createDraft = $state({
    name: '',
    description: '',
    value: '',
  })

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

<div class="space-y-5">
  <Card.Root class="rounded-2xl">
    <Card.Header>
      <Card.Title>Organization secrets</Card.Title>
      <Card.Description>
        Central secrets inherit into every project until a project chooses to override the same
        binding key locally.
      </Card.Description>
    </Card.Header>
    <Card.Content class="space-y-4">
      <div class="grid gap-3 md:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <div class="space-y-3 rounded-2xl border border-slate-200 bg-slate-50/70 p-4">
          <div>
            <h3 class="text-sm font-semibold text-slate-950">Create organization secret</h3>
            <p class="mt-1 text-sm text-slate-600">
              New values become the default binding for every project in this org.
            </p>
          </div>

          <div class="space-y-2">
            <Label for="org-secret-name">Secret name</Label>
            <Input id="org-secret-name" bind:value={createDraft.name} placeholder="GH_TOKEN" />
          </div>
          <div class="space-y-2">
            <Label for="org-secret-value">Secret value</Label>
            <Input
              id="org-secret-value"
              type="password"
              bind:value={createDraft.value}
              placeholder="Paste the new secret value"
            />
            <p class="text-xs text-slate-500">
              Values are masked after write and never shown again.
            </p>
          </div>
        </div>

        <div class="space-y-2 rounded-2xl border border-slate-200 bg-white p-4">
          <Label for="org-secret-description">Description</Label>
          <Textarea
            id="org-secret-description"
            bind:value={createDraft.description}
            rows={6}
            placeholder="What this secret powers, who owns it, and when to rotate it."
          />
          <div class="flex justify-end pt-2">
            <Button onclick={handleCreate} disabled={creating}>
              {creating ? 'Creating…' : 'Create org secret'}
            </Button>
          </div>
        </div>
      </div>
    </Card.Content>
  </Card.Root>

  <OrganizationScopedSecretsInventoryCard
    {loading}
    {secrets}
    onRotate={handleRotate}
    onDisable={handleDisable}
    onDelete={handleDelete}
  />
</div>
