<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { ScopedSecret, ScopedSecretBinding, Ticket, Workflow } from '$lib/api/contracts'
  import {
    createScopedSecretBinding,
    deleteScopedSecretBinding,
    listScopedSecretBindings,
    listScopedSecrets,
    listTickets,
    listWorkflows,
  } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'

  import SecuritySettingsSecretBindings, {
    type SecretBindingDraft,
  } from './security-settings-secret-bindings.svelte'

  const emptySecretBindingDraft = (): SecretBindingDraft => ({
    bindingKey: '',
    scope: 'workflow',
    scopeResourceId: '',
    secretId: '',
  })

  let loading = $state(false)
  let error = $state('')
  let mutationKey = $state('')
  let secrets = $state<ScopedSecret[]>([])
  let bindings = $state<ScopedSecretBinding[]>([])
  let workflows = $state<Workflow[]>([])
  let tickets = $state<Ticket[]>([])
  let draft = $state<SecretBindingDraft>(emptySecretBindingDraft())

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      loading = false
      error = ''
      mutationKey = ''
      secrets = []
      bindings = []
      workflows = []
      tickets = []
      draft = emptySecretBindingDraft()
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''
      try {
        const [secretPayload, bindingPayload, workflowPayload, ticketPayload] = await Promise.all([
          listScopedSecrets(projectId),
          listScopedSecretBindings(projectId),
          listWorkflows(projectId),
          listTickets(projectId),
        ])
        if (cancelled) return
        secrets = secretPayload.secrets
        bindings = bindingPayload.bindings
        workflows = workflowPayload.workflows
        tickets = ticketPayload.tickets
      } catch (caughtError) {
        if (cancelled) return
        error = formatError(caughtError, 'Failed to load runtime secret bindings.')
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  function formatError(caughtError: unknown, fallback: string) {
    return caughtError instanceof ApiError ? caughtError.detail : fallback
  }

  async function reload(projectId: string) {
    const [secretPayload, bindingPayload, workflowPayload, ticketPayload] = await Promise.all([
      listScopedSecrets(projectId),
      listScopedSecretBindings(projectId),
      listWorkflows(projectId),
      listTickets(projectId),
    ])
    secrets = secretPayload.secrets
    bindings = bindingPayload.bindings
    workflows = workflowPayload.workflows
    tickets = ticketPayload.tickets
  }

  async function handleCreate() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    const bindingKey = draft.bindingKey.trim()
    if (!bindingKey || !draft.secretId || !draft.scopeResourceId) {
      toastStore.error('Select a target, a secret, and a binding key first.')
      return
    }

    mutationKey = 'create'
    error = ''
    try {
      await createScopedSecretBinding(projectId, {
        secret_id: draft.secretId,
        scope: draft.scope,
        scope_resource_id: draft.scopeResourceId,
        binding_key: bindingKey,
      })
      await reload(projectId)
      draft = emptySecretBindingDraft()
      toastStore.success('Runtime secret binding created.')
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to create runtime secret binding.')
      error = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }

  async function handleDelete(bindingId: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    mutationKey = `delete:${bindingId}`
    error = ''
    try {
      await deleteScopedSecretBinding(projectId, bindingId)
      await reload(projectId)
      toastStore.success('Runtime secret binding deleted.')
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to delete runtime secret binding.')
      error = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }
</script>

<SecuritySettingsSecretBindings
  {secrets}
  {bindings}
  {workflows}
  {tickets}
  {loading}
  {error}
  {mutationKey}
  {draft}
  onDraftChange={(nextDraft) => (draft = nextDraft)}
  onCreate={() => void handleCreate()}
  onDelete={(bindingId) => void handleDelete(bindingId)}
/>
