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
  let error = $state('')
  let actionKey = $state('')
  let secrets = $state<ScopedSecretRecord[]>([])
  let createDraft = $state({
    name: '',
    description: '',
    value: '',
  })
  let rotateDrafts = $state<Record<string, string>>({})
  let openRotateId = $state('')

  $effect(() => {
    if (!organizationId) {
      secrets = []
      error = ''
      return
    }
    void loadSecrets()
  })

  function formatError(caughtError: unknown, fallback: string) {
    return caughtError instanceof ApiError ? caughtError.detail : fallback
  }

  async function loadSecrets() {
    loading = true
    error = ''
    try {
      const payload = await listOrganizationScopedSecrets(organizationId)
      secrets = payload.secrets ?? []
    } catch (caughtError) {
      error = formatError(caughtError, 'Failed to load organization secrets.')
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

    actionKey = 'create'
    error = ''
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
      const message = formatError(caughtError, 'Failed to create organization secret.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }

  async function handleRotate(secret: ScopedSecretRecord) {
    const value = (rotateDrafts[secret.id] ?? '').trim()
    if (!value) {
      toastStore.error('A new secret value is required to rotate.')
      return
    }

    actionKey = `rotate:${secret.id}`
    error = ''
    try {
      await rotateOrganizationScopedSecret(organizationId, secret.id, { value })
      rotateDrafts = { ...rotateDrafts, [secret.id]: '' }
      openRotateId = ''
      toastStore.success(`Rotated ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to rotate organization secret.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }

  async function handleDisable(secret: ScopedSecretRecord) {
    if (
      !window.confirm(
        `Disable ${secret.name}? Projects will fall back to lower-precedence secrets when available.`,
      )
    ) {
      return
    }

    actionKey = `disable:${secret.id}`
    error = ''
    try {
      await disableOrganizationScopedSecret(organizationId, secret.id)
      toastStore.success(`Disabled ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to disable organization secret.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }

  async function handleDelete(secret: ScopedSecretRecord) {
    if (
      !window.confirm(
        `Delete ${secret.name}? This removes the org default and its binding from every inheriting project.`,
      )
    ) {
      return
    }

    actionKey = `delete:${secret.id}`
    error = ''
    try {
      await deleteOrganizationScopedSecret(organizationId, secret.id)
      toastStore.success(`Deleted ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to delete organization secret.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }

  function toggleRotate(secretId: string) {
    openRotateId = openRotateId === secretId ? '' : secretId
  }

  function updateRotateDraft(secretId: string, value: string) {
    rotateDrafts = {
      ...rotateDrafts,
      [secretId]: value,
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
            <Button onclick={handleCreate} disabled={actionKey === 'create'}>
              {actionKey === 'create' ? 'Creating…' : 'Create org secret'}
            </Button>
          </div>
        </div>
      </div>
    </Card.Content>
  </Card.Root>

  {#if error}
    <div class="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
      {error}
    </div>
  {/if}

  <OrganizationScopedSecretsInventoryCard
    {loading}
    {secrets}
    {actionKey}
    {openRotateId}
    {rotateDrafts}
    onToggleRotate={toggleRotate}
    onRotateInput={updateRotateDraft}
    onRotate={handleRotate}
    onDisable={handleDisable}
    onDelete={handleDelete}
  />
</div>
