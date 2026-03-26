<script lang="ts">
  import type { AgentOutputEntry, AgentProvider, Machine } from '$lib/api/contracts'
  import type { StreamConnectionState } from '$lib/api/sse'
  import type { AgentInstance, ProviderConfig, ProviderDraft, ProviderDraftField } from '../types'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import AgentsPageDrawers from './agents-page-drawers.svelte'
  import AgentsPagePanel from './agents-page-panel.svelte'

  let {
    activeTab = $bindable('instances'),
    registerSheetOpen = $bindable(false),
    providerConfigOpen = $bindable(false),
    outputSheetOpen = $bindable(false),
    agents,
    providers,
    loading = false,
    error = '',
    pageFeedback = '',
    pageError = '',
    runtimeActionAgentId = null,
    canRegister = false,
    registerButtonTitle,
    onOpenRegister,
    onSelectTicket,
    onViewOutput,
    onConfigureProvider,
    onPauseAgent,
    onResumeAgent,
    providerItems,
    machineItems,
    registrationDraft,
    currentOrgSlug,
    currentProjectSlug,
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
    outputStreamState = 'idle',
    onProviderDraftChange,
    onProviderSave,
    onOutputOpenChange,
  }: {
    activeTab?: string
    registerSheetOpen?: boolean
    providerConfigOpen?: boolean
    outputSheetOpen?: boolean
    agents: AgentInstance[]
    providers: ProviderConfig[]
    loading?: boolean
    error?: string
    pageFeedback?: string
    pageError?: string
    runtimeActionAgentId?: string | null
    canRegister?: boolean
    registerButtonTitle?: string
    onOpenRegister?: () => void
    onSelectTicket?: (ticketId: string) => void
    onViewOutput?: (agentId: string) => void
    onConfigureProvider?: (provider: ProviderConfig) => void
    onPauseAgent?: (agentId: string) => void
    onResumeAgent?: (agentId: string) => void
    providerItems: AgentProvider[]
    machineItems: Machine[]
    registrationDraft: AgentRegistrationDraft
    currentOrgSlug?: string
    currentProjectSlug?: string
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
    outputStreamState?: StreamConnectionState
    onProviderDraftChange?: (field: ProviderDraftField, value: string) => void
    onProviderSave?: () => void
    onOutputOpenChange?: (open: boolean) => void
  } = $props()
</script>

<div class="space-y-4">
  <AgentsPagePanel
    bind:activeTab
    {agents}
    {providers}
    {loading}
    {error}
    {pageFeedback}
    {pageError}
    {runtimeActionAgentId}
    {canRegister}
    {registerButtonTitle}
    {onOpenRegister}
    {onSelectTicket}
    {onViewOutput}
    {onConfigureProvider}
    {onPauseAgent}
    {onResumeAgent}
  />
</div>

<AgentsPageDrawers
  bind:registerSheetOpen
  bind:providerConfigOpen
  bind:outputSheetOpen
  {providerItems}
  {machineItems}
  {registrationDraft}
  {currentOrgSlug}
  {currentProjectSlug}
  {registerSaving}
  {registerError}
  {registerFeedback}
  {onRegistrationDraftChange}
  {onRegisterAgent}
  {onRegisterOpenChange}
  {selectedProvider}
  {providerDraft}
  {providerSaving}
  {providerFeedback}
  {providerError}
  {selectedOutputAgent}
  {outputEntries}
  {outputLoading}
  {outputError}
  {outputStreamState}
  {onProviderDraftChange}
  {onProviderSave}
  {onOutputOpenChange}
/>
