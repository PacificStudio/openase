<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import { Button } from '$ui/button'
  import type { AgentOutputEntry, AgentProvider, AgentStepEntry, Machine } from '$lib/api/contracts'
  import type { StreamConnectionState } from '$lib/api/sse'
  import { Plus } from '@lucide/svelte'
  import type {
    AgentInstance,
    AgentRunInstance,
    ProviderConfig,
    ProviderDraft,
    ProviderDraftField,
  } from '../types'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import AgentsPageDrawers from './agents-page-drawers.svelte'
  import AgentsPagePanel from './agents-page-panel.svelte'

  let {
    activeTab = $bindable('runtime'),
    registerSheetOpen = $bindable(false),
    providerConfigOpen = $bindable(false),
    outputSheetOpen = $bindable(false),
    agents,
    agentRuns,
    providers,
    loading = false,
    error = '',
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
    onRegistrationDraftChange,
    onRegisterAgent,
    onRegisterOpenChange,
    selectedProvider,
    providerDraft,
    providerSaving = false,
    selectedOutputAgent,
    outputEntries,
    outputSteps,
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
    agentRuns: AgentRunInstance[]
    providers: ProviderConfig[]
    loading?: boolean
    error?: string
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
    onRegistrationDraftChange?: (field: AgentRegistrationDraftField, value: string) => void
    onRegisterAgent?: () => void
    onRegisterOpenChange?: (open: boolean) => void
    selectedProvider: ProviderConfig | null
    providerDraft: ProviderDraft
    providerSaving?: boolean
    selectedOutputAgent: AgentInstance | null
    outputEntries: AgentOutputEntry[]
    outputSteps: AgentStepEntry[]
    outputLoading?: boolean
    outputError?: string
    outputStreamState?: StreamConnectionState
    onProviderDraftChange?: (field: ProviderDraftField, value: string) => void
    onProviderSave?: () => void
    onOutputOpenChange?: (open: boolean) => void
  } = $props()
</script>

{#snippet actions()}
  <Button
    size="sm"
    onclick={() => onOpenRegister?.()}
    disabled={!canRegister}
    title={registerButtonTitle}
  >
    <Plus class="size-3.5" />
    Register Agent
  </Button>
{/snippet}

<PageScaffold
  title="Agents"
  description="Manage runtime sessions, definitions, and provider bindings."
  {actions}
>
  <div class="space-y-4">
    <AgentsPagePanel
      bind:activeTab
      {agents}
      {agentRuns}
      {providers}
      {loading}
      {error}
      {runtimeActionAgentId}
      {onSelectTicket}
      {onViewOutput}
      {onConfigureProvider}
      {onPauseAgent}
      {onResumeAgent}
    />
  </div>
</PageScaffold>

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
  {onRegistrationDraftChange}
  {onRegisterAgent}
  {onRegisterOpenChange}
  {selectedProvider}
  {providerDraft}
  {providerSaving}
  {selectedOutputAgent}
  {outputEntries}
  {outputSteps}
  {outputLoading}
  {outputError}
  {outputStreamState}
  {onProviderDraftChange}
  {onProviderSave}
  {onOutputOpenChange}
/>
