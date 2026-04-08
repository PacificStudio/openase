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
  import { toastStore } from '$lib/stores/toast.svelte'
  import ProjectScopedSecretsEffectiveCard from './project-scoped-secrets-effective-card.svelte'
  import ProjectScopedSecretsOrgDefaultsCard from './project-scoped-secrets-org-defaults-card.svelte'
  import ProjectScopedSecretsOverviewCard from './project-scoped-secrets-overview-card.svelte'
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
  <ProjectScopedSecretsOverviewCard
    effectiveCount={inventory.effective.length}
    projectOverrideCount={inventory.projectOverrides.length}
    organizationSecretCount={inventory.organizationSecrets.length}
    {organizationId}
    {actionKey}
    bind:name={createDraft.name}
    bind:description={createDraft.description}
    bind:value={createDraft.value}
    onCreate={handleCreate}
  />

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
