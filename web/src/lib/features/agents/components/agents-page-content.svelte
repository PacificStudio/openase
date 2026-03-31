<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import { Button } from '$ui/button'
  import type { AgentOutputEntry, AgentProvider, AgentStepEntry } from '$lib/api/contracts'
  import type { StreamConnectionState } from '$lib/api/sse'
  import { Plus } from '@lucide/svelte'
  import type { AgentInstance, AgentRunInstance } from '../types'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import AgentsPageDrawers from './agents-page-drawers.svelte'
  import AgentsPagePanel from './agents-page-panel.svelte'

  let {
    registerSheetOpen = $bindable(false),
    outputSheetOpen = $bindable(false),
    agents,
    agentRuns,
    loading = false,
    error = '',
    runtimeActionAgentId = null,
    canRegister = false,
    registerButtonTitle,
    onOpenRegister,
    onSelectAgent,
    onSelectTicket,
    onViewOutput,
    onPauseAgent,
    onResumeAgent,
    providerItems,
    registrationDraft,
    currentOrgSlug,
    currentProjectSlug,
    registerSaving = false,
    onRegistrationDraftChange,
    onRegisterAgent,
    onRegisterOpenChange,
    selectedOutputAgent,
    outputEntries,
    outputSteps,
    outputLoading = false,
    outputError = '',
    outputStreamState = 'idle',
    onOutputOpenChange,
  }: {
    registerSheetOpen?: boolean
    outputSheetOpen?: boolean
    agents: AgentInstance[]
    agentRuns: AgentRunInstance[]
    loading?: boolean
    error?: string
    runtimeActionAgentId?: string | null
    canRegister?: boolean
    registerButtonTitle?: string
    onOpenRegister?: () => void
    onSelectAgent?: (agentId: string) => void
    onSelectTicket?: (ticketId: string) => void
    onViewOutput?: (agentId: string) => void
    onPauseAgent?: (agentId: string) => void
    onResumeAgent?: (agentId: string) => void
    providerItems: AgentProvider[]
    registrationDraft: AgentRegistrationDraft
    currentOrgSlug?: string
    currentProjectSlug?: string
    registerSaving?: boolean
    onRegistrationDraftChange?: (field: AgentRegistrationDraftField, value: string) => void
    onRegisterAgent?: () => void
    onRegisterOpenChange?: (open: boolean) => void
    selectedOutputAgent: AgentInstance | null
    outputEntries: AgentOutputEntry[]
    outputSteps: AgentStepEntry[]
    outputLoading?: boolean
    outputError?: string
    outputStreamState?: StreamConnectionState
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
  description="Manage agent definitions and monitor their runs."
  {actions}
>
  <AgentsPagePanel
    {agents}
    {agentRuns}
    {loading}
    {error}
    {runtimeActionAgentId}
    {onSelectAgent}
    {onSelectTicket}
    {onViewOutput}
    {onPauseAgent}
    {onResumeAgent}
  />
</PageScaffold>

<AgentsPageDrawers
  bind:registerSheetOpen
  bind:outputSheetOpen
  {providerItems}
  {registrationDraft}
  {currentOrgSlug}
  {currentProjectSlug}
  {registerSaving}
  {onRegistrationDraftChange}
  {onRegisterAgent}
  {onRegisterOpenChange}
  {selectedOutputAgent}
  {outputEntries}
  {outputSteps}
  {outputLoading}
  {outputError}
  {outputStreamState}
  {onOutputOpenChange}
/>
