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
  let creating = $state(false)
  let secrets = $state<ScopedSecretRecord[]>([])
  let createDraft = $state({ name: '', description: '', value: '' })

  const inventory = $derived(buildProjectSecretInventory(secrets))

  $effect(() => {
    if (!projectId) {
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
      const payload = await listProjectScopedSecrets(projectId)
      secrets = payload.secrets ?? []
    } catch (caughtError) {
      toastStore.error(formatError(caughtError, 'Failed to load project secrets.'))
      secrets = []
    } finally {
      loading = false
    }
  }

  function primeOverride(secret: ScopedSecretRecord) {
    createDraft.name = secret.name
    createDraft.description = secret.description
    toastStore.info('Override form pre-filled — scroll up to set the value and save.')
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
      toastStore.error(formatError(caughtError, 'Failed to create project secret.'))
    } finally {
      creating = false
    }
  }

  async function handleRotate(secret: ScopedSecretRecord, value: string) {
    try {
      await rotateProjectScopedSecret(projectId, secret.id, { value })
      toastStore.success(`Rotated ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(formatError(caughtError, 'Failed to rotate project secret.'))
      throw caughtError
    }
  }

  async function handleDisable(secret: ScopedSecretRecord) {
    try {
      await disableProjectScopedSecret(projectId, secret.id)
      toastStore.success(`Disabled ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(formatError(caughtError, 'Failed to disable project secret.'))
      throw caughtError
    }
  }

  async function handleDelete(secret: ScopedSecretRecord) {
    try {
      await deleteProjectScopedSecret(projectId, secret.id)
      toastStore.success(`Deleted ${secret.name}.`)
      await loadSecrets()
    } catch (caughtError) {
      toastStore.error(formatError(caughtError, 'Failed to delete project secret.'))
      throw caughtError
    }
  }
</script>

<div class="space-y-5">
  <ProjectScopedSecretsOverviewCard
    effectiveCount={inventory.effective.length}
    projectOverrideCount={inventory.projectOverrides.length}
    organizationSecretCount={inventory.organizationSecrets.length}
    {organizationId}
    {creating}
    bind:name={createDraft.name}
    bind:description={createDraft.description}
    bind:value={createDraft.value}
    onCreate={handleCreate}
  />

  <ProjectScopedSecretsEffectiveCard
    {loading}
    effectiveSecrets={inventory.effective}
    organizationSecrets={inventory.organizationSecrets}
  />

  <ProjectScopedSecretsOverridesCard
    {loading}
    projectOverrides={inventory.projectOverrides}
    organizationSecrets={inventory.organizationSecrets}
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
