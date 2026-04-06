<script lang="ts">
  import { preloadCode, preloadData } from '$app/navigation'
  import type { Organization, Project } from '$lib/api/contracts'
  import type { ProjectSection } from '$lib/stores/app-context'
  import { viewport } from '$lib/stores/viewport.svelte'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import { Menu } from '@lucide/svelte'
  import TopBarBreadcrumb from './top-bar-breadcrumb.svelte'
  import TopBarPrimaryActions from './top-bar-primary-actions.svelte'
  import TopBarUserMenu from './top-bar-user-menu.svelte'

  let {
    organizations = [],
    projects = [],
    currentOrgId = null,
    currentProjectId = null,
    currentSection = 'dashboard' as ProjectSection,
    orgName = 'My Org',
    projectName = '',
    projectHealth = null,
    projectHealthLabel = '',
    sseStatus = 'live' as 'idle' | 'connecting' | 'live' | 'retrying',
    searchEnabled = false,
    newTicketEnabled = false,
    newTicketTitle,
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
    onOpenMobileNav,
    onOpenProjectAssistant,
  }: {
    organizations?: Organization[]
    projects?: Project[]
    currentOrgId?: string | null
    currentProjectId?: string | null
    currentSection?: ProjectSection
    orgName?: string
    projectName?: string
    projectHealth?: 'healthy' | 'degraded' | 'critical' | null
    projectHealthLabel?: string
    sseStatus?: 'idle' | 'connecting' | 'live' | 'retrying'
    searchEnabled?: boolean
    newTicketEnabled?: boolean
    newTicketTitle?: string
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
    onOpenMobileNav?: () => void
    onOpenProjectAssistant?: (initialPrompt?: string) => void
  } = $props()

  const isMobile = $derived(viewport.isMobile)

  const healthDotClass = $derived.by(() => {
    switch (projectHealth) {
      case 'healthy':
        return 'bg-success'
      case 'degraded':
        return 'bg-warning'
      case 'critical':
        return 'bg-destructive'
      default:
        return ''
    }
  })

  function warmRoute(href: string) {
    void preloadCode(href)
    void preloadData(href)
  }

  const userInitials = $derived.by(() => {
    const source = userDisplayName.trim() || userPrimaryEmail.trim()
    if (!source) return 'U'
    const parts = source.split(/\s+/).filter((value) => value !== '')
    if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
    return `${parts[0]?.[0] ?? ''}${parts[1]?.[0] ?? ''}`.toUpperCase()
  })
</script>

<header
  class="border-border bg-background flex h-12 shrink-0 items-center gap-2 border-b px-3 md:px-4"
>
  {#if isMobile}
    <Button
      variant="ghost"
      size="icon-sm"
      class="shrink-0"
      onclick={onOpenMobileNav}
      aria-label="Open navigation"
    >
      <Menu class="size-4" />
    </Button>
  {/if}

  <a href="/orgs" class="text-foreground mr-1 flex items-center gap-1.5 text-sm font-semibold">
    <img src="/favicon.svg" alt="" class="size-5" />
    <span class={cn('text-primary font-bold', isMobile && 'sr-only')}>OpenASE</span>
  </a>

  {#if !isMobile}
    <Separator orientation="vertical" class="mx-1 h-5" />
    <TopBarBreadcrumb
      {organizations}
      {projects}
      {currentOrgId}
      {currentProjectId}
      {currentSection}
      {orgName}
      {projectName}
      {projectHealth}
      {projectHealthLabel}
      {onCreateOrg}
      {onCreateProject}
    />
  {:else if projectName}
    <span class="text-foreground min-w-0 truncate text-xs font-medium">{projectName}</span>
    {#if projectHealth}
      <span class={cn('size-1.5 shrink-0 rounded-full', healthDotClass)}></span>
    {/if}
  {/if}

  <div class="flex-1"></div>

  <TopBarPrimaryActions
    {searchEnabled}
    {newTicketEnabled}
    {newTicketTitle}
    {sseStatus}
    {onOpenSearch}
    {onNewTicket}
    {onOpenProjectAssistant}
    projectSelected={Boolean(currentProjectId)}
  />

  <TopBarUserMenu
    {userDisplayName}
    {userPrimaryEmail}
    {userAvatarURL}
    {userInitials}
    {logoutPending}
    {settingsEnabled}
    {settingsHref}
    {onToggleTheme}
    {onOpenSettings}
    onWarmSettings={warmRoute}
    {onLogout}
  />
</header>
