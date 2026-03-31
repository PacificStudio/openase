<script lang="ts">
  import type { AgentOutputEntry, AgentProvider, AgentStepEntry } from '$lib/api/contracts'
  import type { StreamConnectionState } from '$lib/api/sse'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import type { AgentInstance } from '../types'
  import AgentOutputSheet from './agent-output-sheet.svelte'
  import AgentRegistrationSheet from './agent-registration-sheet.svelte'

  let {
    registerSheetOpen = $bindable(false),
    outputSheetOpen = $bindable(false),
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

<AgentOutputSheet
  bind:open={outputSheetOpen}
  agent={selectedOutputAgent}
  entries={outputEntries}
  steps={outputSteps}
  loading={outputLoading}
  error={outputError}
  streamState={outputStreamState}
  onOpenChange={onOutputOpenChange}
/>
