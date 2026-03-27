<script lang="ts">
  import type { AgentOutputEntry, AgentProvider, Machine } from '$lib/api/contracts'
  import type { StreamConnectionState } from '$lib/api/sse'
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
    machineItems,
    registrationDraft,
    currentOrgSlug,
    currentProjectSlug,
    registerSaving = false,
    onRegistrationDraftChange,
    onRegisterAgent,
    onRegisterOpenChange,
    selectedProvider,
    providerDraft,
    providerSaving = false,
    selectedOutputAgent,
    outputEntries,
    outputLoading = false,
    outputError = '',
    outputStreamState = 'idle',
    onProviderDraftChange,
    onProviderSave,
    onOutputOpenChange,
  }: {
    registerSheetOpen?: boolean
    providerConfigOpen?: boolean
    outputSheetOpen?: boolean
    providerItems: AgentProvider[]
    machineItems: Machine[]
    registrationDraft: AgentRegistrationDraft
    currentOrgSlug?: string
    currentProjectSlug?: string
    registerSaving?: boolean
    onRegistrationDraftChange?: (field: AgentRegistrationDraftField, value: string) => void
    onRegisterAgent?: () => void
    onRegisterOpenChange?: (open: boolean) => void
    selectedProvider: ProviderConfig | null
    providerDraft: ProviderDraft
    providerSaving?: boolean
    selectedOutputAgent: AgentInstance | null
    outputEntries: AgentOutputEntry[]
    outputLoading?: boolean
    outputError?: string
    outputStreamState?: StreamConnectionState
    onProviderDraftChange?: (field: ProviderDraftField, value: string) => void
    onProviderSave?: () => void
    onOutputOpenChange?: (open: boolean) => void
  } = $props()
</script>

<AgentRegistrationSheet
  bind:open={registerSheetOpen}
  providers={providerItems}
  draft={registrationDraft}
  {currentOrgSlug}
  {currentProjectSlug}
  saving={registerSaving}
  onDraftChange={onRegistrationDraftChange}
  onSubmit={onRegisterAgent}
  onOpenChange={onRegisterOpenChange}
/>

<ProviderConfigSheet
  bind:open={providerConfigOpen}
  provider={selectedProvider}
  machines={machineItems}
  draft={providerDraft}
  saving={providerSaving}
  onDraftChange={onProviderDraftChange}
  onSave={onProviderSave}
/>

<AgentOutputSheet
  bind:open={outputSheetOpen}
  agent={selectedOutputAgent}
  entries={outputEntries}
  loading={outputLoading}
  error={outputError}
  streamState={outputStreamState}
  onOpenChange={onOutputOpenChange}
/>
