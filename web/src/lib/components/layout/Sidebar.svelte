<script lang="ts">
  import { ArrowRightLeft, FolderCog, KanbanSquare, LayoutDashboard, Orbit, Radar, Ticket } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import ScrollPane from '$lib/components/layout/ScrollPane.svelte'
  import SurfacePanel from '$lib/components/layout/SurfacePanel.svelte'
  import OrganizationProjectSwitcher from './OrganizationProjectSwitcher.svelte'
  import type { Organization, Project } from '$lib/types/workspace-shell'

  let {
    organizations = [],
    projects = [],
    selectedOrgId = '',
    selectedProjectId = '',
    selectedPage = 'overview',
    workflowCount = 0,
    ticketCount = 0,
    runningAgentCount = 0,
    connectorCount = 0,
    onSelectOrganization,
    onSelectProject,
  }: {
    organizations?: Organization[]
    projects?: Project[]
    selectedOrgId?: string
    selectedProjectId?: string
    selectedPage?: 'overview' | 'board' | 'workflows' | 'agents' | 'activity' | 'settings'
    workflowCount?: number
    ticketCount?: number
    runningAgentCount?: number
    connectorCount?: number
    onSelectOrganization?: (organization: Organization) => void
    onSelectProject?: (project: Project) => void
  } = $props()

  const pages = $derived.by(() => [
    {
      key: 'overview' as const,
      label: 'Overview',
      href: '/',
      icon: LayoutDashboard,
      badge: `${ticketCount}`,
    },
    {
      key: 'board' as const,
      label: 'Board',
      href: '/board',
      icon: KanbanSquare,
      badge: `${ticketCount}`,
    },
    {
      key: 'workflows' as const,
      label: 'Workflows',
      href: '/workflows',
      icon: Orbit,
      badge: `${workflowCount}`,
    },
    {
      key: 'agents' as const,
      label: 'Agents',
      href: '/agents',
      icon: Radar,
      badge: `${runningAgentCount}`,
    },
    {
      key: 'activity' as const,
      label: 'Activity',
      href: '/activity',
      icon: Ticket,
      badge: 'live',
    },
    {
      key: 'settings' as const,
      label: 'Settings',
      href: '/settings',
      icon: FolderCog,
      badge: `${connectorCount}`,
    },
  ])

  function pageHref(path: string) {
    const params = new URLSearchParams()
    if (selectedOrgId) {
      params.set('org', selectedOrgId)
    }
    if (selectedProjectId) {
      params.set('project', selectedProjectId)
    }

    const query = params.toString()
    return query ? `${path}?${query}` : path
  }
</script>

<SurfacePanel class="h-full" bodyClass="min-h-0 px-3 py-3">
  <div class="flex h-full min-h-0 flex-col gap-4">
    <div class="space-y-3">
      <div class="rounded-[1.4rem] border border-amber-500/20 bg-[linear-gradient(140deg,rgba(245,158,11,0.12),rgba(255,255,255,0.9)_45%,rgba(20,184,166,0.08))] px-4 py-4">
        <p class="text-xs font-medium tracking-[0.24em] uppercase text-slate-600">Project shell</p>
        <p class="mt-2 text-base font-semibold text-slate-900">
          {projects.find((project) => project.id === selectedProjectId)?.name ?? 'Select a project'}
        </p>
        <div class="mt-3 flex flex-wrap gap-2">
          <Badge variant="outline">{organizations.length} orgs</Badge>
          <Badge variant="outline">{projects.length} projects</Badge>
          <Badge variant="outline">{ticketCount} tickets</Badge>
          <Badge variant="outline">{workflowCount} workflows</Badge>
        </div>
      </div>

      <div>
        <div class="mb-2 flex items-center gap-2 px-1">
          <ArrowRightLeft class="text-muted-foreground size-4" />
          <p class="text-sm font-medium">Navigation</p>
        </div>

        <nav class="grid gap-2">
          {#each pages as page}
            <a
              href={pageHref(page.href)}
              class={`rounded-2xl border px-3 py-3 text-sm transition ${
                selectedPage === page.key
                  ? 'border-foreground/25 bg-foreground text-background shadow-sm'
                  : 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
              }`}
            >
              <div class="flex items-center justify-between gap-3">
                <div class="flex min-w-0 items-center gap-3">
                  <page.icon class="size-4 shrink-0" />
                  <span class="truncate font-medium">{page.label}</span>
                </div>
                <span
                  class={`rounded-full border px-2 py-0.5 text-[11px] ${
                    selectedPage === page.key
                      ? 'border-background/20 text-background/75'
                      : 'border-border/70 text-muted-foreground'
                  }`}
                >
                  {page.badge}
                </span>
              </div>
            </a>
          {/each}
        </nav>
      </div>
    </div>

    <div class="border-border/70 min-h-0 flex-1 border-t pt-4">
      <div class="mb-2 flex items-center justify-between gap-2 px-1">
        <div>
          <p class="text-sm font-medium">Context</p>
          <p class="text-muted-foreground text-xs">Switch org and project without moving navigation.</p>
        </div>
      </div>

      <ScrollPane class="h-full">
        <OrganizationProjectSwitcher
          {organizations}
          {projects}
          {selectedOrgId}
          {selectedProjectId}
          {onSelectOrganization}
          {onSelectProject}
        />
      </ScrollPane>
    </div>
  </div>
</SurfacePanel>
