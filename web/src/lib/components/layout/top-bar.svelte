<script lang="ts">
  import { goto } from '$app/navigation'
  import type { Organization, Project } from '$lib/api/contracts'
  import { organizationPath, projectPath, type ProjectSection } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
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
    sseStatus = 'live' as 'idle' | 'connecting' | 'live' | 'retrying',
    searchEnabled = false,
    newTicketEnabled = false,
    newTicketTitle,
    onToggleTheme,
    onNewTicket,
    onOpenSearch,
  }: {
    organizations?: Organization[]
    projects?: Project[]
    currentOrgId?: string | null
    currentProjectId?: string | null
    currentSection?: ProjectSection
    orgName?: string
    projectName?: string
    sseStatus?: 'idle' | 'connecting' | 'live' | 'retrying'
    searchEnabled?: boolean
    newTicketEnabled?: boolean
    newTicketTitle?: string
    onToggleTheme?: () => void
    onNewTicket?: () => void
    onOpenSearch?: () => void
  } = $props()

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
      <DropdownMenu.Label>Organizations</DropdownMenu.Label>
      {#if organizations.length > 0}
        {#each organizations as organization (organization.id)}
          <DropdownMenu.Item onclick={() => void handleOrgSelect(organization.id)}>
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
        <DropdownMenu.Label>Projects</DropdownMenu.Label>
        {#if projects.length > 0}
          {#each projects as project (project.id)}
            <DropdownMenu.Item onclick={() => void handleProjectSelect(project.id)}>
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
            <Avatar.Fallback class="bg-primary/10 text-primary text-xs">U</Avatar.Fallback>
          </Avatar.Root>
        </button>
      {/snippet}
    </DropdownMenu.Trigger>
    <DropdownMenu.Content align="end" class="w-48">
      <DropdownMenu.Item onclick={onToggleTheme}>
        <Moon class="mr-2 size-4" />
        Toggle Theme
      </DropdownMenu.Item>
      <DropdownMenu.Item>
        <Settings class="mr-2 size-4" />
        Settings
      </DropdownMenu.Item>
      <DropdownMenu.Separator />
      <DropdownMenu.Item>
        <LogOut class="mr-2 size-4" />
        Logout
      </DropdownMenu.Item>
    </DropdownMenu.Content>
  </DropdownMenu.Root>
</header>
