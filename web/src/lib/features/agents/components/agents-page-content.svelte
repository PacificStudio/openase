<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { PageScaffold } from '$lib/components/layout'
  import { Button } from '$ui/button'
  import type { AgentProvider } from '$lib/api/contracts'
  import { Plus } from '@lucide/svelte'
  import type { AgentInstance, AgentRunInstance } from '../types'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import AgentsPageDrawers from './agents-page-drawers.svelte'
  import AgentsPagePanel from './agents-page-panel.svelte'

  let {
    registerSheetOpen = $bindable(false),
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
    onInterruptAgent,
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
  }: {
    registerSheetOpen?: boolean
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
    onInterruptAgent?: (agentId: string) => void
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
    {i18nStore.t('agents.registerAgent')}
  </Button>
{/snippet}

<PageScaffold
  title={i18nStore.t('agents.pageTitle')}
  description={i18nStore.t('agents.pageDescription')}
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
    {onInterruptAgent}
    {onPauseAgent}
    {onResumeAgent}
  />
</PageScaffold>

<AgentsPageDrawers
  bind:registerSheetOpen
  {providerItems}
  {registrationDraft}
  {currentOrgSlug}
  {currentProjectSlug}
  {registerSaving}
  {onRegistrationDraftChange}
  {onRegisterAgent}
  {onRegisterOpenChange}
/>
