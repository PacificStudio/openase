import { ApiError } from '$lib/api/client'
import type { AgentProvider } from '$lib/api/contracts'
import { updateProvider } from '$lib/api/openase'
import { toastStore } from '$lib/stores/toast.svelte'
import { createEmptyProviderDraft, parseProviderDraft, providerToDraft } from '../provider-draft'
import type { ProviderConfig, ProviderDraft, ProviderDraftField } from '../types'

export function createProviderEditorState() {
  let selectedProviderId = $state<string | null>(null)
  let draft = $state<ProviderDraft>(createEmptyProviderDraft())
  let saving = $state(false)

  return {
    get selectedProviderId() {
      return selectedProviderId
    },
    get draft() {
      return draft
    },
    get saving() {
      return saving
    },
    open(provider: ProviderConfig) {
      selectedProviderId = provider.id
      draft = providerToDraft(provider)
      saving = false
    },
    updateField(field: ProviderDraftField, value: string) {
      draft = {
        ...draft,
        [field]: value,
      }
    },
    async save(
      provider: ProviderConfig | null,
      onApplied: (updatedProvider: AgentProvider) => void,
    ) {
      if (!provider) {
        toastStore.error('Select a provider to configure.')
        return
      }

      const parsed = parseProviderDraft(draft)
      if (!parsed.ok) {
        toastStore.error(parsed.error)
        return
      }

      saving = true

      try {
        const payload = await updateProvider(provider.id, parsed.value)
        if (!payload.provider) {
          toastStore.warning(
            'Provider updated, but the latest provider data could not be refreshed. Please reload the page.',
          )
          return
        }

        onApplied(payload.provider)
        toastStore.success('Provider updated.')
      } catch (caughtError) {
        toastStore.error(
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to save provider.',
        )
      } finally {
        saving = false
      }
    },
    reset() {
      selectedProviderId = null
      draft = createEmptyProviderDraft()
      saving = false
    },
  }
}
