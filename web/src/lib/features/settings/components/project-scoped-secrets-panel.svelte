<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import {
    createProjectScopedSecret,
    deleteProjectScopedSecret,
    disableProjectScopedSecret,
    listProjectScopedSecrets,
    rotateProjectScopedSecret,
  } from '$lib/api/openase'
  import type { ScopedSecretRecord } from '$lib/api/contracts'
  import { organizationPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import ProjectScopedSecretsEffectiveCard from './project-scoped-secrets-effective-card.svelte'
  import ProjectScopedSecretsOrgDefaultsCard from './project-scoped-secrets-org-defaults-card.svelte'
  import ProjectScopedSecretsOverridesCard from './project-scoped-secrets-overrides-card.svelte'
  import { buildProjectSecretInventory } from '../scoped-secrets'

  let { projectId, organizationId }: { projectId: string; organizationId: string } = $props()

  let loading = $state(false)
  let error = $state('')
  let actionKey = $state('')
  let secrets = $state<ScopedSecretRecord[]>([])
  let createDraft = $state({ name: '', description: '', value: '' })
  let rotateDrafts = $state<Record<string, string>>({})
  let openRotateId = $state('')

  const inventory = $derived(buildProjectSecretInventory(secrets))

  $effect(() => {
    if (!projectId) {
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
      const payload = await listProjectScopedSecrets(projectId)
      secrets = payload.secrets ?? []
    } catch (caughtError) {
      error = formatError(caughtError, 'Failed to load project secrets.')
      secrets = []
    } finally {
      loading = false
    }
  }

  function primeOverride(secret: ScopedSecretRecord) {
    createDraft.name = secret.name
    createDraft.description = secret.description
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
      await createProjectScopedSecret(projectId, {
        scope: 'project',
        name,
        description: createDraft.description.trim(),
        value,
      })
      createDraft = { name: '', description: '', value: '' }
      toastStore.success('Created project override secret.')
      await loadSecrets()
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to create project secret.')
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
      await rotateProjectScopedSecret(projectId, secret.id, { value })
      rotateDrafts = { ...rotateDrafts, [secret.id]: '' }
      openRotateId = ''
      toastStore.success(`Rotated ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to rotate project secret.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }

  async function handleDisable(secret: ScopedSecretRecord) {
    if (
      !window.confirm(
        `Disable ${secret.name}? Project runtimes will fall back to lower precedence secrets when available.`,
      )
    ) {
      return
    }

    actionKey = `disable:${secret.id}`
    error = ''
    try {
      await disableProjectScopedSecret(projectId, secret.id)
      toastStore.success(`Disabled ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to disable project secret.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }

  async function handleDelete(secret: ScopedSecretRecord) {
    if (
      !window.confirm(
        `Delete ${secret.name}? This removes the project override and its default binding.`,
      )
    ) {
      return
    }

    actionKey = `delete:${secret.id}`
    error = ''
    try {
      await deleteProjectScopedSecret(projectId, secret.id)
      toastStore.success(`Deleted ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to delete project secret.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }
</script>

<div class="space-y-5">
  <Card.Root class="rounded-2xl border-slate-200">
    <Card.Header>
      <Card.Title>Scoped secrets</Card.Title>
      <Card.Description>
        Effective runtime secrets inherit org defaults until this project adds an override with the
        same binding key.
      </Card.Description>
    </Card.Header>
    <Card.Content class="space-y-4">
      <div class="grid gap-3 sm:grid-cols-3">
        <div class="rounded-2xl border border-slate-200 bg-slate-50/70 p-4">
          <div class="text-xs font-semibold tracking-[0.18em] text-slate-500 uppercase">
            Effective now
          </div>
          <div class="mt-2 text-3xl font-semibold text-slate-950">{inventory.effective.length}</div>
          <p class="mt-1 text-sm text-slate-600">What runtime resolution sees after overrides.</p>
        </div>
        <div class="rounded-2xl border border-slate-200 bg-white p-4">
          <div class="text-xs font-semibold tracking-[0.18em] text-slate-500 uppercase">
            Project overrides
          </div>
          <div class="mt-2 text-3xl font-semibold text-slate-950">
            {inventory.projectOverrides.length}
          </div>
          <p class="mt-1 text-sm text-slate-600">Project-only secrets and org overrides.</p>
        </div>
        <div class="rounded-2xl border border-slate-200 bg-white p-4">
          <div class="text-xs font-semibold tracking-[0.18em] text-slate-500 uppercase">
            Org defaults
          </div>
          <div class="mt-2 text-3xl font-semibold text-slate-950">
            {inventory.organizationSecrets.length}
          </div>
          <p class="mt-1 text-sm text-slate-600">
            Central inventory managed from org admin settings.
          </p>
        </div>
      </div>

      <div class="rounded-2xl border border-slate-200 bg-white p-4">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 class="text-sm font-semibold text-slate-950">Create project override</h3>
            <p class="mt-1 text-sm text-slate-600">
              Use the same name as an inherited org secret to override it for this project only.
            </p>
          </div>
          {#if organizationId}
            <a
              href={`${organizationPath(organizationId)}/admin/settings`}
              class="text-sm font-medium text-slate-700 underline-offset-4 hover:text-slate-950 hover:underline"
            >
              Manage org inventory
            </a>
          {/if}
        </div>

        <div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
          <div class="space-y-3">
            <div class="space-y-2">
              <Label for="project-secret-name">Secret name</Label>
              <Input
                id="project-secret-name"
                bind:value={createDraft.name}
                placeholder="OPENAI_API_KEY"
              />
            </div>
            <div class="space-y-2">
              <Label for="project-secret-value">Secret value</Label>
              <Input
                id="project-secret-value"
                type="password"
                bind:value={createDraft.value}
                placeholder="Paste the new secret value"
              />
              <p class="text-xs text-slate-500">
                The raw value is only accepted on write and never shown again.
              </p>
            </div>
          </div>

          <div class="space-y-2">
            <Label for="project-secret-description">Description</Label>
            <Textarea
              id="project-secret-description"
              bind:value={createDraft.description}
              rows={4}
              placeholder="Optional context for operators and future rotations."
            />
          </div>
        </div>

        <div class="mt-4 flex justify-end">
          <Button onclick={handleCreate} disabled={actionKey === 'create'}>
            {actionKey === 'create' ? 'Creating…' : 'Create override'}
          </Button>
        </div>
      </div>
    </Card.Content>
  </Card.Root>

  {#if error}
    <div class="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
      {error}
    </div>
  {/if}

  <ProjectScopedSecretsEffectiveCard
    {loading}
    effectiveSecrets={inventory.effective}
    organizationSecrets={inventory.organizationSecrets}
  />

  <ProjectScopedSecretsOverridesCard
    {loading}
    projectOverrides={inventory.projectOverrides}
    organizationSecrets={inventory.organizationSecrets}
    {actionKey}
    {openRotateId}
    {rotateDrafts}
    onToggleRotate={toggleRotate}
    onRotateInput={updateRotateDraft}
    onRotate={handleRotate}
    onDisable={handleDisable}
    onDelete={handleDelete}
  />

  <ProjectScopedSecretsOrgDefaultsCard
    {loading}
    organizationSecrets={inventory.organizationSecrets}
    projectOverrides={inventory.projectOverrides}
    {organizationId}
    onPrimeOverride={primeOverride}
  />
</div>
