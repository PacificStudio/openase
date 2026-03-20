<script lang="ts">
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import { ChevronDown, Search, Plus, Settings, LogOut, Moon } from '@lucide/svelte'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import * as Avatar from '$ui/avatar'

  let {
    orgName = 'My Org',
    projectName = '',
    sseStatus = 'live' as 'idle' | 'connecting' | 'live' | 'retrying',
    searchEnabled = false,
    newTicketEnabled = false,
    searchTitle,
    newTicketTitle,
    onToggleTheme,
    onNewTicket,
    onOpenSearch,
  }: {
    orgName?: string
    projectName?: string
    sseStatus?: 'idle' | 'connecting' | 'live' | 'retrying'
    searchEnabled?: boolean
    newTicketEnabled?: boolean
    searchTitle?: string
    newTicketTitle?: string
    onToggleTheme?: () => void
    onNewTicket?: () => void
    onOpenSearch?: () => void
  } = $props()
</script>

<header class="border-border bg-background flex h-12 shrink-0 items-center gap-2 border-b px-4">
  <!-- Logo -->
  <a href="/" class="text-foreground mr-1 flex items-center gap-1.5 text-sm font-semibold">
    <span class="text-primary font-bold">OpenASE</span>
  </a>

  <Separator orientation="vertical" class="mx-1 h-5" />

  <!-- Org Switcher -->
  <Button variant="ghost" size="sm" class="text-muted-foreground gap-1 text-xs">
    {orgName}
    <ChevronDown class="size-3" />
  </Button>

  {#if projectName}
    <span class="text-muted-foreground/50">/</span>
    <Button variant="ghost" size="sm" class="text-foreground gap-1 text-xs font-medium">
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
    class="text-muted-foreground hidden w-[200px] justify-start gap-2 sm:flex"
    disabled={!searchEnabled}
    title={searchEnabled ? 'Open search' : (searchTitle ?? 'Search is not available.')}
    onclick={onOpenSearch}
  >
    <Search class="size-3.5" />
    <span class="text-xs">Search...</span>
    <kbd class="bg-muted ml-auto rounded px-1.5 py-0.5 font-mono text-[10px]">⌘K</kbd>
  </Button>

  <Separator orientation="vertical" class="mx-1 h-5" />

  <!-- New Ticket -->
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

  <!-- SSE Status Indicator -->
  <div class="text-muted-foreground flex items-center gap-1.5 text-xs" title="SSE: {sseStatus}">
    {#if sseStatus === 'live'}
      <span class="bg-success size-1.5 rounded-full"></span>
    {:else if sseStatus === 'connecting' || sseStatus === 'retrying'}
      <span class="bg-warning size-1.5 animate-pulse rounded-full"></span>
    {:else}
      <span class="bg-destructive size-1.5 rounded-full"></span>
    {/if}
  </div>

  <!-- User Menu -->
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
