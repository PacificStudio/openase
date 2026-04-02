<script lang="ts">
  import { OrganizationCreationDialog, ProjectCreationDialog } from '$lib/features/catalog-creation'
  import { GlobalSearchDialog } from '$lib/features/search'
  import { NewTicketDialog } from '$lib/features/tickets'
  import { TicketDrawer } from '$lib/features/ticket-detail'
  import { appStore } from '$lib/stores/app.svelte'
  import type { Organization, Project } from '$lib/api/contracts'
  import type { ProjectSection } from '$lib/stores/app-context'

  let {
    currentOrg = null,
    currentProject = null,
    currentSection = 'dashboard' as ProjectSection,
    currentTicketId = null,
    searchOpen = $bindable(false),
    createOrgOpen = $bindable(false),
    createProjectOpen = $bindable(false),
    newTicketEnabled = false,
    onToggleTheme,
    onNewTicket,
    onOpenProjectAssistant,
  }: {
    currentOrg?: Organization | null
    currentProject?: Project | null
    currentSection?: ProjectSection
    currentTicketId?: string | null
    searchOpen?: boolean
    createOrgOpen?: boolean
    createProjectOpen?: boolean
    newTicketEnabled?: boolean
    onToggleTheme?: () => void
    onNewTicket?: () => void
    onOpenProjectAssistant?: (initialPrompt?: string) => void
  } = $props()
</script>

<TicketDrawer
  projectId={currentProject?.id}
  ticketId={currentTicketId}
  open={appStore.rightPanelOpen}
  onOpenChange={(open) => {
    if (!open) appStore.closeRightPanel()
  }}
/>

<NewTicketDialog />
<OrganizationCreationDialog bind:open={createOrgOpen} />

<ProjectCreationDialog
  orgId={currentOrg?.id ?? ''}
  providers={appStore.providers ?? []}
  defaultProviderId={currentProject?.default_agent_provider_id ?? null}
  bind:open={createProjectOpen}
/>

<GlobalSearchDialog
  bind:open={searchOpen}
  organizations={appStore.organizations}
  projects={appStore.projects}
  {currentOrg}
  {currentProject}
  {currentSection}
  {newTicketEnabled}
  {onToggleTheme}
  {onNewTicket}
  {onOpenProjectAssistant}
  onOpenTicket={(ticketId) => appStore.openRightPanel({ type: 'ticket', id: ticketId })}
/>
