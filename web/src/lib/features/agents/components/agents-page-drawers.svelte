<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import AgentRegistrationSheet from './agent-registration-sheet.svelte'

  let {
    registerSheetOpen = $bindable(false),
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
