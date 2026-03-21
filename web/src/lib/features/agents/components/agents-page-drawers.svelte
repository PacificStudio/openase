<script lang="ts">
  import type { AgentOutputEntry, AgentProvider } from '$lib/api/contracts'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import type { AgentInstance, ProviderConfig, ProviderDraft, ProviderDraftField } from '../types'
  import AgentOutputSheet from './agent-output-sheet.svelte'
  import AgentRegistrationSheet from './agent-registration-sheet.svelte'
  import ProviderConfigSheet from './provider-config-sheet.svelte'

  let {
    registerSheetOpen = $bindable(false),
    providerConfigOpen = $bindable(false),
    outputSheetOpen = $bindable(false),
    providerItems,
    registrationDraft,
    registerSaving = false,
    registerError = '',
    registerFeedback = '',
    onRegistrationDraftChange,
    onRegisterAgent,
    onRegisterOpenChange,
    selectedProvider,
    providerDraft,
    providerSaving = false,
    providerFeedback = '',
    providerError = '',
    selectedOutputAgent,
    outputEntries,
    outputLoading = false,
    outputError = '',
    onProviderDraftChange,
    onProviderSave,
    onRefreshOutput,
  }: {
    registerSheetOpen?: boolean
    providerConfigOpen?: boolean
    outputSheetOpen?: boolean
    providerItems: AgentProvider[]
    registrationDraft: AgentRegistrationDraft
    registerSaving?: boolean
    registerError?: string
    registerFeedback?: string
    onRegistrationDraftChange?: (field: AgentRegistrationDraftField, value: string) => void
    onRegisterAgent?: () => void
    onRegisterOpenChange?: (open: boolean) => void
    selectedProvider: ProviderConfig | null
    providerDraft: ProviderDraft
    providerSaving?: boolean
    providerFeedback?: string
    providerError?: string
    selectedOutputAgent: AgentInstance | null
    outputEntries: AgentOutputEntry[]
    outputLoading?: boolean
    outputError?: string
    onProviderDraftChange?: (field: ProviderDraftField, value: string) => void
    onProviderSave?: () => void
    onRefreshOutput?: () => void
  } = $props()
</script>

<AgentRegistrationSheet
  bind:open={registerSheetOpen}
  providers={providerItems}
  draft={registrationDraft}
  saving={registerSaving}
  error={registerError}
  feedback={registerFeedback}
  onDraftChange={onRegistrationDraftChange}
  onSubmit={onRegisterAgent}
  onOpenChange={onRegisterOpenChange}
/>

<ProviderConfigSheet
  bind:open={providerConfigOpen}
  provider={selectedProvider}
  draft={providerDraft}
  saving={providerSaving}
  feedback={providerFeedback}
  error={providerError}
  onDraftChange={onProviderDraftChange}
  onSave={onProviderSave}
/>

<AgentOutputSheet
  bind:open={outputSheetOpen}
  agent={selectedOutputAgent}
  entries={outputEntries}
  loading={outputLoading}
  error={outputError}
  onRefresh={onRefreshOutput}
/>
