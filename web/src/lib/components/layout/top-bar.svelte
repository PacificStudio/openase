<script lang="ts">
  import { goto, preloadCode, preloadData } from '$app/navigation'
  import type { Organization, Project } from '$lib/api/contracts'
  import { organizationPath, projectPath, type ProjectSection } from '$lib/stores/app-context'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import * as Tooltip from '$ui/tooltip'
  import { ChevronDown, Search, Plus, Settings, LogOut, Moon, Check } from '@lucide/svelte'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import * as Avatar from '$ui/avatar'

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
  } = $props()

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

  function handleOrgSelect(orgId: string) {
    return goto(organizationPath(orgId))
  }

  function handleProjectSelect(projectId: string) {
    if (!currentOrgId) {
      return Promise.resolve()
    }

    const section = currentProjectId ? currentSection : 'dashboard'
    return goto(projectPath(currentOrgId, projectId, section))
  }

  function handleOpenSearchClick() {
    onOpenSearch?.()
  }

  function warmRoute(href: string) {
    void preloadCode(href)
    void preloadData(href)
  }

  const userInitials = $derived.by(() => {
    const source = userDisplayName.trim() || userPrimaryEmail.trim()
    if (!source) {
      return 'U'
    }
    const parts = source.split(/\s+/).filter((value) => value !== '')
    if (parts.length === 1) {
      return parts[0].slice(0, 2).toUpperCase()
    }
    return `${parts[0]?.[0] ?? ''}${parts[1]?.[0] ?? ''}`.toUpperCase()
  })
</script>

<header class="border-border bg-background flex h-12 shrink-0 items-center gap-2 border-b px-4">
  <a href="/orgs" class="text-foreground mr-1 flex items-center gap-1.5 text-sm font-semibold">
    <span class="text-primary font-bold">OpenASE</span>
  </a>

  <Separator orientation="vertical" class="mx-1 h-5" />

  <DropdownMenu.Root>
    <DropdownMenu.Trigger>
      {#snippet child({ props })}
        <Button
          {...props}
          variant="ghost"
          size="sm"
          class="text-muted-foreground gap-1 text-xs"
          disabled={organizations.length === 0}
        >
          {orgName}
          <ChevronDown class="size-3" />
        </Button>
      {/snippet}
    </DropdownMenu.Trigger>
    <DropdownMenu.Content class="w-56">
      <DropdownMenu.Label class="flex items-center justify-between">
        <span>Organizations</span>
        {#if onCreateOrg}
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground hover:bg-accent -mr-1 flex size-5 items-center justify-center rounded"
            onclick={(e) => {
              e.stopPropagation()
              onCreateOrg()
            }}
            title="Create organization"
          >
            <Plus class="size-3.5" />
          </button>
        {/if}
      </DropdownMenu.Label>
      {#if organizations.length > 0}
        {#each organizations as organization (organization.id)}
          <DropdownMenu.Item
            onclick={() => void handleOrgSelect(organization.id)}
            onpointerenter={() => warmRoute(organizationPath(organization.id))}
          >
            <span class="flex w-full items-center gap-2">
              {#if organization.id === currentOrgId}
                <Check class="size-4" />
              {:else}
                <span class="size-4"></span>
              {/if}
              <span class={organization.id === currentOrgId ? 'font-medium' : ''}>
                {organization.name}
              </span>
            </span>
          </DropdownMenu.Item>
        {/each}
      {:else}
        <DropdownMenu.Item disabled>No organizations available</DropdownMenu.Item>
      {/if}
    </DropdownMenu.Content>
  </DropdownMenu.Root>

  {#if currentOrgId && (projectName || projects.length > 0)}
    <span class="text-muted-foreground/50">/</span>
    {#if projectHealth}
      <Tooltip.Root>
        <Tooltip.Trigger>
          {#snippet child({ props })}
            <span
              {...props}
              class={cn('size-2 shrink-0 cursor-default rounded-full', healthDotClass)}
            ></span>
          {/snippet}
        </Tooltip.Trigger>
        <Tooltip.Content side="bottom" class="max-w-64 text-xs">
          {projectHealthLabel ||
            (projectHealth === 'healthy'
              ? 'All systems healthy'
              : projectHealth === 'degraded'
                ? 'Project has warnings'
                : 'Project has critical issues')}
        </Tooltip.Content>
      </Tooltip.Root>
    {/if}
    <DropdownMenu.Root>
      <DropdownMenu.Trigger>
        {#snippet child({ props })}
          <Button {...props} variant="ghost" size="sm" class="text-foreground gap-1 text-xs">
            <span class={projectName ? 'font-medium' : 'text-muted-foreground'}>
              {projectName || 'Select project'}
            </span>
            <ChevronDown class="size-3" />
          </Button>
        {/snippet}
      </DropdownMenu.Trigger>
      <DropdownMenu.Content class="w-64">
        <DropdownMenu.Label class="flex items-center justify-between">
          <span>Projects</span>
          {#if onCreateProject}
            <button
              type="button"
              class="text-muted-foreground hover:text-foreground hover:bg-accent -mr-1 flex size-5 items-center justify-center rounded"
              onclick={(e) => {
                e.stopPropagation()
                onCreateProject()
              }}
              title="Create project"
            >
              <Plus class="size-3.5" />
            </button>
          {/if}
        </DropdownMenu.Label>
        {#if projects.length > 0}
          {#each projects as project (project.id)}
            <DropdownMenu.Item
              onclick={() => void handleProjectSelect(project.id)}
              onpointerenter={() => {
                if (!currentOrgId) return
                warmRoute(projectPath(currentOrgId, project.id, currentSection))
              }}
            >
              <span class="flex w-full items-center gap-2">
                {#if project.id === currentProjectId}
                  <Check class="size-4" />
                {:else}
                  <span class="size-4"></span>
                {/if}
                <span class={project.id === currentProjectId ? 'font-medium' : ''}>
                  {project.name}
                </span>
              </span>
            </DropdownMenu.Item>
          {/each}
        {:else}
          <DropdownMenu.Item disabled>No projects available</DropdownMenu.Item>
        {/if}
      </DropdownMenu.Content>
    </DropdownMenu.Root>
  {/if}

  <div class="flex-1"></div>

  {#if searchEnabled}
    <Button
      variant="outline"
      size="sm"
      class="text-muted-foreground hidden w-[200px] justify-start gap-2 sm:flex"
      onclick={handleOpenSearchClick}
    >
      <Search class="size-3.5" />
      <span class="text-xs">Search...</span>
      <kbd class="bg-muted ml-auto rounded px-1.5 py-0.5 font-mono text-[10px]">⌘K</kbd>
    </Button>

    <Separator orientation="vertical" class="mx-1 h-5" />
  {/if}

  <Button
    size="sm"
    class="gap-1.5"
    disabled={!newTicketEnabled}
    title={newTicketEnabled
      ? 'Create ticket'
      : (newTicketTitle ?? 'Ticket creation is not available.')}
    onclick={onNewTicket}
  >
    <Plus class="size-3.5" />
    <span class="hidden text-xs sm:inline">New Ticket</span>
  </Button>

  <div class="text-muted-foreground flex items-center gap-1.5 text-xs" title="SSE: {sseStatus}">
    {#if sseStatus === 'live'}
      <span class="bg-success size-1.5 rounded-full"></span>
    {:else if sseStatus === 'connecting' || sseStatus === 'retrying'}
      <span class="bg-warning size-1.5 animate-pulse rounded-full"></span>
    {:else}
      <span class="bg-destructive size-1.5 rounded-full"></span>
    {/if}
  </div>

  <DropdownMenu.Root>
    <DropdownMenu.Trigger>
      {#snippet child({ props })}
        <button {...props} class="rounded-full">
          <Avatar.Root class="size-7">
            {#if userAvatarURL}
              <Avatar.Image
                src={userAvatarURL}
                alt={userDisplayName || userPrimaryEmail || 'User'}
              />
            {/if}
            <Avatar.Fallback class="bg-primary/10 text-primary text-xs"
              >{userInitials}</Avatar.Fallback
            >
          </Avatar.Root>
        </button>
      {/snippet}
    </DropdownMenu.Trigger>
    <DropdownMenu.Content align="end" class="w-48">
      {#if userDisplayName || userPrimaryEmail}
        <DropdownMenu.Label class="space-y-0.5">
          {#if userDisplayName}
            <div class="text-foreground text-sm font-medium">{userDisplayName}</div>
          {/if}
          {#if userPrimaryEmail}
            <div class="text-muted-foreground text-xs">{userPrimaryEmail}</div>
          {/if}
        </DropdownMenu.Label>
        <DropdownMenu.Separator />
      {/if}
      <DropdownMenu.Item onclick={onToggleTheme}>
        <Moon class="mr-2 size-4" />
        Toggle Theme
      </DropdownMenu.Item>
      <DropdownMenu.Item
        onclick={onOpenSettings}
        onpointerenter={() => {
          if (settingsHref) {
            warmRoute(settingsHref)
          }
        }}
        disabled={!settingsEnabled}
      >
        <Settings class="mr-2 size-4" />
        Settings
      </DropdownMenu.Item>
      <DropdownMenu.Separator />
      <DropdownMenu.Item onclick={onLogout} disabled={logoutPending}>
        <LogOut class="mr-2 size-4" />
        {logoutPending ? 'Logging out…' : 'Logout'}
      </DropdownMenu.Item>
    </DropdownMenu.Content>
  </DropdownMenu.Root>
</header>
