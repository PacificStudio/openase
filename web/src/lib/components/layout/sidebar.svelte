<script lang="ts">
  import { preloadCode } from '$app/navigation'
  import { buildGlobalNav, buildProjectNav, type SidebarNavItem } from './sidebar-nav'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Badge } from '$ui/badge'
  import { Separator } from '$ui/separator'
  import * as Tooltip from '$ui/tooltip'
  import { Bot, ChevronsLeft, ChevronsRight } from '@lucide/svelte'

  let {
    collapsed = false,
    currentPath = '/',
    currentOrgId = null,
    currentProjectId = null,
    projectSelected = false,
    projectName = '',
    projectHealth = 'healthy' as 'healthy' | 'degraded' | 'critical',
    agentCount = 0,
    onOpenProjectAssistant,
    onToggleCollapse,
  }: {
    collapsed?: boolean
    currentPath?: string
    currentOrgId?: string | null
    currentProjectId?: string | null
    projectSelected?: boolean
    projectName?: string
    projectHealth?: 'healthy' | 'degraded' | 'critical'
    agentCount?: number
    onOpenProjectAssistant?: () => void
    onToggleCollapse?: () => void
  } = $props()

  const globalNav: SidebarNavItem[] = $derived(buildGlobalNav(currentPath, currentOrgId))
  const projectNav: SidebarNavItem[] = $derived(
    buildProjectNav({ currentPath, currentOrgId, currentProjectId, agentCount }),
  )

  const healthColor = $derived(
    projectHealth === 'healthy'
      ? 'bg-success'
      : projectHealth === 'degraded'
        ? 'bg-warning'
        : 'bg-destructive',
  )

  function warmRoute(href: string) {
    void preloadCode(href)
  }
</script>

<nav class="flex h-full flex-col overflow-hidden">
  <div class="flex-1 overflow-y-auto px-2 py-3">
    <div class="space-y-0.5">
      {#each globalNav as item}
        {@const Icon = item.icon}
        {#if collapsed}
          <Tooltip.Root>
            <Tooltip.Trigger>
              {#snippet child({ props })}
                <a
                  href={item.href}
                  {...props}
                  data-sveltekit-preload-code="hover"
                  class={cn(
                    'flex h-8 w-full items-center justify-center rounded-md text-sm transition-colors',
                    item.active
                      ? 'border-primary bg-sidebar-accent text-sidebar-foreground border-l-2'
                      : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-foreground',
                  )}
                  onpointerenter={() => warmRoute(item.href)}
                >
                  <Icon class="size-4" />
                </a>
              {/snippet}
            </Tooltip.Trigger>
            <Tooltip.Content side="right" class="text-xs">
              {item.label}
            </Tooltip.Content>
          </Tooltip.Root>
        {:else}
          <a
            href={item.href}
            data-sveltekit-preload-code="hover"
            class={cn(
              'flex h-8 items-center gap-2.5 rounded-md px-2.5 text-sm transition-colors',
              item.active
                ? 'border-primary bg-sidebar-accent text-sidebar-foreground border-l-2'
                : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-foreground',
            )}
            onpointerenter={() => warmRoute(item.href)}
          >
            <Icon class="size-4 shrink-0" />
            <span class="truncate">{item.label}</span>
            {#if item.badge}
              <Badge
                variant="secondary"
                class="ml-auto h-5 min-w-5 justify-center px-1 text-[10px]"
              >
                {item.badge}
              </Badge>
            {/if}
          </a>
        {/if}
      {/each}
    </div>
    {#if projectSelected}
      <Separator class="my-3" />
      {#if !collapsed}
        <div class="mb-2 flex items-center gap-2 px-2.5">
          <span class={cn('size-2 shrink-0 rounded-full', healthColor)}></span>
          <span class="text-sidebar-foreground truncate text-xs font-medium">{projectName}</span>
        </div>
        <div class="mb-3 px-2.5">
          <Button variant="outline" size="sm" class="w-full" onclick={onOpenProjectAssistant}>
            <Bot class="size-4" />
            Ask AI
          </Button>
        </div>
      {:else}
        <div class="mb-2 flex flex-col items-center gap-2">
          <span class={cn('size-2 rounded-full', healthColor)}></span>
          <Tooltip.Root>
            <Tooltip.Trigger>
              {#snippet child({ props })}
                <Button
                  variant="ghost"
                  size="icon-sm"
                  {...props}
                  onclick={onOpenProjectAssistant}
                  aria-label="Ask AI"
                >
                  <Bot class="size-4" />
                </Button>
              {/snippet}
            </Tooltip.Trigger>
            <Tooltip.Content side="right" class="text-xs">Ask AI</Tooltip.Content>
          </Tooltip.Root>
        </div>
      {/if}
      <div class="space-y-0.5">
        {#each projectNav as item}
          {@const Icon = item.icon}
          {#if collapsed}
            <Tooltip.Root>
              <Tooltip.Trigger>
                {#snippet child({ props })}
                  <a
                    href={item.href}
                    {...props}
                    data-sveltekit-preload-code="hover"
                    class={cn(
                      'flex h-8 w-full items-center justify-center rounded-md text-sm transition-colors',
                      item.active
                        ? 'border-primary bg-sidebar-accent text-sidebar-foreground border-l-2'
                        : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-foreground',
                    )}
                    onpointerenter={() => warmRoute(item.href)}
                  >
                    <Icon class="size-4" />
                  </a>
                {/snippet}
              </Tooltip.Trigger>
              <Tooltip.Content side="right" class="text-xs">
                <span>{item.label}</span>
                {#if item.badge}
                  <Badge
                    variant="secondary"
                    class="ml-1.5 h-4 min-w-4 justify-center px-1 text-[10px]"
                  >
                    {item.badge}
                  </Badge>
                {/if}
              </Tooltip.Content>
            </Tooltip.Root>
          {:else}
            <a
              href={item.href}
              data-sveltekit-preload-code="hover"
              class={cn(
                'flex h-8 items-center gap-2.5 rounded-md px-2.5 text-sm transition-colors',
                item.active
                  ? 'border-primary bg-sidebar-accent text-sidebar-foreground border-l-2'
                  : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-foreground',
              )}
              onpointerenter={() => warmRoute(item.href)}
            >
              <Icon class="size-4 shrink-0" />
              <span class="truncate">{item.label}</span>
              {#if item.badge}
                <Badge
                  variant="secondary"
                  class="ml-auto h-5 min-w-5 justify-center px-1 text-[10px]"
                >
                  {item.badge}
                </Badge>
              {/if}
            </a>
          {/if}
        {/each}
      </div>
    {/if}
  </div>
  <div class="border-border shrink-0 border-t p-2">
    <Button
      variant="ghost"
      size="sm"
      class={cn('w-full', collapsed ? 'justify-center px-0' : 'justify-start')}
      onclick={onToggleCollapse}
    >
      {#if collapsed}
        <ChevronsRight class="size-4" />
      {:else}
        <ChevronsLeft class="mr-2 size-4" />
        <span class="text-muted-foreground text-xs">Collapse</span>
      {/if}
    </Button>
  </div>
</nav>
