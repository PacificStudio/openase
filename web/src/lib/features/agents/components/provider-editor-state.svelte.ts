import { ApiError } from '$lib/api/client'
import type { AgentProvider } from '$lib/api/contracts'
import { updateProvider } from '$lib/api/openase'
import { createEmptyProviderDraft, parseProviderDraft, providerToDraft } from '../model'
import type { ProviderConfig, ProviderDraft, ProviderDraftField } from '../types'

export function createProviderEditorState() {
  let selectedProviderId = $state<string | null>(null)
  let draft = $state<ProviderDraft>(createEmptyProviderDraft())
  let saving = $state(false)
  let feedback = $state('')
  let error = $state('')

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
    get feedback() {
      return feedback
    },
    get error() {
      return error
    },
    open(provider: ProviderConfig) {
      selectedProviderId = provider.id
      draft = providerToDraft(provider)
      saving = false
      feedback = ''
      error = ''
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
        error = 'Select a provider to configure.'
        return
      }

      const parsed = parseProviderDraft(draft)
      if (!parsed.ok) {
        error = parsed.error
        feedback = ''
        return
      }

      saving = true
      feedback = ''
      error = ''

      try {
        const payload = await updateProvider(provider.id, parsed.value)
        if (!payload.provider) {
          error =
            'Provider updated, but the latest provider data could not be refreshed. Please reload the page.'
          return
        }

        onApplied(payload.provider)
        feedback = 'Provider updated.'
      } catch (caughtError) {
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to save provider.'
      } finally {
        saving = false
      }
    },
    reset() {
      selectedProviderId = null
      draft = createEmptyProviderDraft()
      saving = false
      feedback = ''
      error = ''
    },
    clearMessages() {
      feedback = ''
      error = ''
      saving = false
    },
  }
}
