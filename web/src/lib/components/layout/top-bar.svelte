<script lang="ts">
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import {
    ChevronDown,
    Search,
    Plus,
    Settings,
    LogOut,
    Moon,
  } from '@lucide/svelte'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import * as Avatar from '$ui/avatar'

  let {
    orgName = 'My Org',
    projectName = '',
    sseStatus = 'live' as 'idle' | 'connecting' | 'live' | 'retrying',
    searchEnabled = false,
    newTicketEnabled = false,
    onToggleTheme,
    onNewTicket,
    onOpenSearch,
  }: {
    orgName?: string
    projectName?: string
    sseStatus?: 'idle' | 'connecting' | 'live' | 'retrying'
    searchEnabled?: boolean
    newTicketEnabled?: boolean
    onToggleTheme?: () => void
    onNewTicket?: () => void
    onOpenSearch?: () => void
  } = $props()
</script>

<header class="flex h-12 shrink-0 items-center gap-2 border-b border-border bg-background px-4">
  <!-- Logo -->
  <a href="/" class="mr-1 flex items-center gap-1.5 text-sm font-semibold text-foreground">
    <span class="font-bold text-primary">OpenASE</span>
  </a>

  <Separator orientation="vertical" class="mx-1 h-5" />

  <!-- Org Switcher -->
  <Button variant="ghost" size="sm" class="gap-1 text-xs text-muted-foreground">
    {orgName}
    <ChevronDown class="size-3" />
  </Button>

  {#if projectName}
    <span class="text-muted-foreground/50">/</span>
    <Button variant="ghost" size="sm" class="gap-1 text-xs font-medium text-foreground">
      {projectName}
      <ChevronDown class="size-3" />
    </Button>
  {/if}

  <!-- Spacer -->
  <div class="flex-1"></div>

  <!-- Search Trigger -->
  <Button
    variant="outline"
    size="sm"
    class="hidden w-[200px] justify-start gap-2 text-muted-foreground sm:flex"
    disabled={!searchEnabled}
    title={searchEnabled ? 'Open search' : 'Search is not available in the current API-backed shell'}
    onclick={onOpenSearch}
  >
    <Search class="size-3.5" />
    <span class="text-xs">Search...</span>
    <kbd class="ml-auto rounded bg-muted px-1.5 py-0.5 font-mono text-[10px]">⌘K</kbd>
  </Button>

  <Separator orientation="vertical" class="mx-1 h-5" />

  <!-- New Ticket -->
  <Button
    size="sm"
    class="gap-1.5"
    disabled={!newTicketEnabled}
    title={newTicketEnabled ? 'Create ticket' : 'Ticket creation is not exposed by the current API'}
    onclick={onNewTicket}
  >
    <Plus class="size-3.5" />
    <span class="hidden text-xs sm:inline">New Ticket</span>
  </Button>

  <!-- SSE Status Indicator -->
  <div class="flex items-center gap-1.5 text-xs text-muted-foreground" title="SSE: {sseStatus}">
    {#if sseStatus === 'live'}
      <span class="size-1.5 rounded-full bg-success"></span>
    {:else if sseStatus === 'connecting' || sseStatus === 'retrying'}
      <span class="size-1.5 animate-pulse rounded-full bg-warning"></span>
    {:else}
      <span class="size-1.5 rounded-full bg-destructive"></span>
    {/if}
  </div>

  <!-- User Menu -->
  <DropdownMenu.Root>
    <DropdownMenu.Trigger>
      {#snippet child({ props })}
        <button {...props} class="rounded-full">
          <Avatar.Root class="size-7">
            <Avatar.Fallback class="bg-primary/10 text-xs text-primary">U</Avatar.Fallback>
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
