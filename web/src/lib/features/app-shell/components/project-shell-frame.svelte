<script lang="ts">
  import type { Organization, Project } from '$lib/api/contracts'
  import Sidebar from '$lib/components/layout/sidebar.svelte'
  import TopBar from '$lib/components/layout/top-bar.svelte'
  import type { ProjectAIFocus } from '$lib/features/chat'
  import { workspaceBrowserPortal, ProjectConversationWorkspaceBrowser } from '$lib/features/chat'
  import type { ProjectSection } from '$lib/stores/app-context'
  import { appStore } from '$lib/stores/app.svelte'
  import { viewport } from '$lib/stores/viewport.svelte'
  import { cn } from '$lib/utils'
  import type { Snippet } from 'svelte'
  import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from '$ui/sheet'
  import ProjectShellProjectAssistant from './project-shell-project-assistant.svelte'
  import ProjectShellOverlays from './project-shell-overlays.svelte'

  let {
    children,
    currentPath,
    currentSection,
    currentOrg,
    currentProject,
    organizations = [],
    projects = [],
    adminEnabled = false,
    agentCount = 0,
    sseStatus = 'live' as 'idle' | 'connecting' | 'live' | 'retrying',
    sidebarCollapsed = false,
    searchOpen = $bindable(false),
    createOrgOpen = $bindable(false),
    createProjectOpen = $bindable(false),
    projectAssistantOpen = $bindable(false),
    projectAssistantPrompt = '',
    assistantFocus = null,
    assistantWidth = $bindable(380),
    resizing = $bindable(false),
    currentTicketId = null,
    projectHealth = null,
    projectHealthLabel = '',
    isNewTicketEnabled = false,
    settingsEnabled = false,
    settingsHref = '',
    userDisplayName = '',
    userPrimaryEmail = '',
    userAvatarURL = '',
    logoutPending = false,
    onToggleTheme,
    onNewTicket,
    onOpenSearch,
    onCreateOrg,
    onCreateProject,
    onOpenSettings,
    onLogout,
    onToggleSidebar,
    onOpenProjectAssistant,
    onCloseProjectAssistant = () => {},
  }: {
    children: Snippet
    currentPath: string
    currentSection: ProjectSection
    currentOrg: Organization | null
    currentProject: Project | null
    organizations?: Organization[]
    projects?: Project[]
    adminEnabled?: boolean
    agentCount?: number
    sseStatus?: 'idle' | 'connecting' | 'live' | 'retrying'
    sidebarCollapsed?: boolean
    searchOpen?: boolean
    createOrgOpen?: boolean
    createProjectOpen?: boolean
    projectAssistantOpen?: boolean
    projectAssistantPrompt?: string
    assistantFocus?: ProjectAIFocus | null
    assistantWidth?: number
    resizing?: boolean
    currentTicketId?: string | null
    projectHealth?: 'healthy' | 'degraded' | 'critical' | null
    projectHealthLabel?: string
    isNewTicketEnabled?: boolean
    settingsEnabled?: boolean
    settingsHref?: string
    userDisplayName?: string
    userPrimaryEmail?: string
    userAvatarURL?: string
    logoutPending?: boolean
    onToggleTheme?: () => void
    onNewTicket?: () => void
    onOpenSearch?: () => void
    onCreateOrg?: () => void
    onCreateProject?: () => void
    onOpenSettings?: () => void
    onLogout?: () => void
    onToggleSidebar?: () => void
    onOpenProjectAssistant?: (initialPrompt?: string) => void
    onCloseProjectAssistant?: () => void
  } = $props()

  const isMobile = $derived(viewport.isMobile)

  function handleMobileNavClose() {
    appStore.closeMobileSidebar()
  }
</script>

<div class="bg-background flex h-screen flex-col overflow-hidden">
  <TopBar
    {organizations}
    {projects}
    currentOrgId={currentOrg?.id ?? null}
    currentProjectId={currentProject?.id ?? null}
    {currentSection}
    {adminEnabled}
    orgName={currentOrg?.name ?? 'No organization'}
    projectName={currentProject?.name ?? ''}
    projectHealth={currentProject ? projectHealth : null}
    {projectHealthLabel}
    {sseStatus}
    searchEnabled={organizations.length > 0}
    newTicketEnabled={isNewTicketEnabled}
    {settingsEnabled}
    {settingsHref}
    {userDisplayName}
    {userPrimaryEmail}
    {userAvatarURL}
    {logoutPending}
    {onToggleTheme}
    {onNewTicket}
    {onOpenSearch}
    {onCreateOrg}
    {onCreateProject}
    {onOpenSettings}
    {onLogout}
    onOpenMobileNav={() => appStore.openMobileSidebar()}
    {onOpenProjectAssistant}
  />

  <div class="flex flex-1 overflow-hidden">
    {#if isMobile}
      <Sheet
        bind:open={appStore.mobileSidebarOpen}
        onOpenChange={(open) => {
          if (!open) handleMobileNavClose()
        }}
      >
        <SheetContent side="left" class="w-[280px] p-0" showCloseButton={false}>
          <SheetHeader class="sr-only">
            <SheetTitle>Navigation</SheetTitle>
            <SheetDescription>Project navigation menu</SheetDescription>
          </SheetHeader>
          <Sidebar
            collapsed={false}
            mobile={true}
            {currentPath}
            currentOrgId={currentOrg?.id ?? null}
            currentProjectId={currentProject?.id ?? null}
            projectSelected={Boolean(currentProject)}
            {agentCount}
            onOpenProjectAssistant={() => {
              handleMobileNavClose()
              onOpenProjectAssistant?.()
            }}
            onToggleCollapse={handleMobileNavClose}
            onNavigate={handleMobileNavClose}
          />
        </SheetContent>
      </Sheet>
    {:else}
      <aside
        class={cn(
          'border-border bg-sidebar flex h-full flex-col border-r transition-[width] duration-200 ease-in-out',
          sidebarCollapsed ? 'w-[52px]' : 'w-[240px]',
        )}
      >
        <Sidebar
          collapsed={sidebarCollapsed}
          {currentPath}
          currentOrgId={currentOrg?.id ?? null}
          currentProjectId={currentProject?.id ?? null}
          projectSelected={Boolean(currentProject)}
          {agentCount}
          onOpenProjectAssistant={() => onOpenProjectAssistant?.()}
          onToggleCollapse={onToggleSidebar}
        />
      </aside>
    {/if}

    <main
      class={cn('relative flex min-w-0 flex-1 flex-col overflow-hidden', resizing && 'select-none')}
    >
      {@render children()}
      {#if workspaceBrowserPortal.open}
        <div class="bg-background absolute inset-0 z-10 hidden lg:flex">
          <ProjectConversationWorkspaceBrowser
            conversationId={workspaceBrowserPortal.conversationId}
            workspaceDiff={workspaceBrowserPortal.workspaceDiff}
            workspaceDiffLoading={workspaceBrowserPortal.workspaceDiffLoading}
            pendingFilePath={workspaceBrowserPortal.pendingFilePath}
            onClose={() => workspaceBrowserPortal.close()}
            onPendingFileConsumed={() => workspaceBrowserPortal.consumePendingFile()}
          />
        </div>
      {/if}
    </main>

    {#if currentOrg?.id && currentProject?.id}
      <ProjectShellProjectAssistant
        organizationId={currentOrg.id}
        projectId={currentProject.id}
        projectName={currentProject.name ?? ''}
        defaultProviderId={currentProject.default_agent_provider_id ?? null}
        focus={assistantFocus}
        open={projectAssistantOpen}
        initialPrompt={projectAssistantPrompt}
        bind:width={assistantWidth}
        bind:resizing
        onClose={onCloseProjectAssistant}
      />
    {/if}
  </div>

  <ProjectShellOverlays
    {currentOrg}
    {currentProject}
    {currentSection}
    {currentTicketId}
    bind:searchOpen
    bind:createOrgOpen
    bind:createProjectOpen
    newTicketEnabled={isNewTicketEnabled}
    {onToggleTheme}
    {onNewTicket}
    {onOpenProjectAssistant}
  />
</div>
