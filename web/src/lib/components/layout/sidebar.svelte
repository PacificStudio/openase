<script lang="ts">
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Badge } from '$ui/badge'
  import { Separator } from '$ui/separator'
  import * as Tooltip from '$ui/tooltip'
  import {
    LayoutDashboard,
    Bot,
    TicketCheck,
    Activity,
    Settings,
    KanbanSquare,
    Workflow,
    ChevronsLeft,
    ChevronsRight,
  } from '@lucide/svelte'
  import type { Component } from 'svelte'

  type NavItem = {
    label: string
    href: string
    icon: Component
    badge?: string | number
    active?: boolean
  }

  let {
    collapsed = false,
    currentPath = '/',
    projectSelected = false,
    projectName = '',
    projectHealth = 'healthy' as 'healthy' | 'degraded' | 'critical',
    agentCount = 0,
    onToggleCollapse,
  }: {
    collapsed?: boolean
    currentPath?: string
    projectSelected?: boolean
    projectName?: string
    projectHealth?: 'healthy' | 'degraded' | 'critical'
    agentCount?: number
    onToggleCollapse?: () => void
  } = $props()

  const globalNav: NavItem[] = $derived([
    { label: 'Dashboard', href: '/', icon: LayoutDashboard, active: currentPath === '/' },
  ])

  const projectNav: NavItem[] = $derived([
    {
      label: 'Board',
      href: '/board',
      icon: KanbanSquare,
      active: currentPath.startsWith('/board'),
    },
    {
      label: 'Tickets',
      href: '/tickets',
      icon: TicketCheck,
      active: currentPath.startsWith('/tickets'),
    },
    {
      label: 'Agents',
      href: '/agents',
      icon: Bot,
      badge: agentCount || undefined,
      active: currentPath.startsWith('/agents'),
    },
    {
      label: 'Activity',
      href: '/activity',
      icon: Activity,
      active: currentPath.startsWith('/activity'),
    },
    {
      label: 'Workflows',
      href: '/workflows',
      icon: Workflow,
      active: currentPath.startsWith('/workflows'),
    },
    {
      label: 'Settings',
      href: '/settings',
      icon: Settings,
      active: currentPath.startsWith('/settings'),
    },
  ])

  const healthColor = $derived(
    projectHealth === 'healthy'
      ? 'bg-success'
      : projectHealth === 'degraded'
        ? 'bg-warning'
        : 'bg-destructive',
  )
</script>

<nav class="flex h-full flex-col overflow-hidden">
  <!-- Sidebar content with scroll -->
  <div class="flex-1 overflow-y-auto px-2 py-3">
    <!-- Global Nav -->
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
                  class={cn(
                    'flex h-8 w-full items-center justify-center rounded-md text-sm transition-colors',
                    item.active
                      ? 'border-primary bg-sidebar-accent text-sidebar-foreground border-l-2'
                      : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-foreground',
                  )}
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
            class={cn(
              'flex h-8 items-center gap-2.5 rounded-md px-2.5 text-sm transition-colors',
              item.active
                ? 'border-primary bg-sidebar-accent text-sidebar-foreground border-l-2'
                : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-foreground',
            )}
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

    <!-- Project Section -->
    {#if projectSelected}
      <Separator class="my-3" />

      <!-- Project Header -->
      {#if !collapsed}
        <div class="mb-2 flex items-center gap-2 px-2.5">
          <span class={cn('size-2 shrink-0 rounded-full', healthColor)}></span>
          <span class="text-sidebar-foreground truncate text-xs font-medium">{projectName}</span>
        </div>
      {:else}
        <div class="mb-2 flex justify-center">
          <span class={cn('size-2 rounded-full', healthColor)}></span>
        </div>
      {/if}

      <!-- Project Nav -->
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
                    class={cn(
                      'flex h-8 w-full items-center justify-center rounded-md text-sm transition-colors',
                      item.active
                        ? 'border-primary bg-sidebar-accent text-sidebar-foreground border-l-2'
                        : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-foreground',
                    )}
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
              class={cn(
                'flex h-8 items-center gap-2.5 rounded-md px-2.5 text-sm transition-colors',
                item.active
                  ? 'border-primary bg-sidebar-accent text-sidebar-foreground border-l-2'
                  : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-foreground',
              )}
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

  <!-- Collapse Toggle -->
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
