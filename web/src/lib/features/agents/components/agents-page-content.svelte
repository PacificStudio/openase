<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import type { AgentInstance, ProviderConfig, ProviderDraft } from '../types'
  import type { AgentRegistrationDraft } from '../registration'
  import AgentsPageDrawers from './agents-page-drawers.svelte'
  import AgentsPagePanel from './agents-page-panel.svelte'

  let {
    activeTab = $bindable('instances'),
    registerSheetOpen = $bindable(false),
    providerConfigOpen = $bindable(false),
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
    onConfigureProvider,
    onPauseAgent,
    onResumeAgent,
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
    onProviderDraftChange,
    onProviderSave,
  }: {
    activeTab?: string
    registerSheetOpen?: boolean
    providerConfigOpen?: boolean
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
    onConfigureProvider?: (provider: ProviderConfig) => void
    onPauseAgent?: (agentId: string) => void
    onResumeAgent?: (agentId: string) => void
    providerItems: AgentProvider[]
    registrationDraft: AgentRegistrationDraft
    registerSaving?: boolean
    registerError?: string
    registerFeedback?: string
    onRegistrationDraftChange?: (field: string, value: string) => void
    onRegisterAgent?: () => void
    onRegisterOpenChange?: (open: boolean) => void
    selectedProvider: ProviderConfig | null
    providerDraft: ProviderDraft
    providerSaving?: boolean
    providerFeedback?: string
    providerError?: string
    onProviderDraftChange?: (field: string, value: string) => void
    onProviderSave?: () => void
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
    {onConfigureProvider}
    {onPauseAgent}
    {onResumeAgent}
  />
</div>

<AgentsPageDrawers
  bind:registerSheetOpen
  bind:providerConfigOpen
  {providerItems}
  {registrationDraft}
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
  {onProviderDraftChange}
  {onProviderSave}
/>
