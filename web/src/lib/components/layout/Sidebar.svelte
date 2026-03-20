<script lang="ts">
  import { ArrowRightLeft, PanelsLeftBottom } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import OrganizationProjectSwitcher from './OrganizationProjectSwitcher.svelte'
  import type { Organization, Project } from '$lib/types/workspace-shell'

  let {
    organizations = [],
    projects = [],
    selectedOrgId = '',
    selectedProjectId = '',
    selectedPage = 'board',
    workflowCount = 0,
    ticketCount = 0,
    onSelectOrganization,
    onSelectProject,
  }: {
    organizations?: Organization[]
    projects?: Project[]
    selectedOrgId?: string
    selectedProjectId?: string
    selectedPage?: 'board' | 'workflows' | 'agents' | 'notifications'
    workflowCount?: number
    ticketCount?: number
    onSelectOrganization?: (organization: Organization) => void
    onSelectProject?: (project: Project) => void
  } = $props()

  const pages = [
    { key: 'board', label: 'Board', href: '/' },
    { key: 'workflows', label: 'Workflows', href: '/workflows' },
    { key: 'agents', label: 'Agents', href: '/agents' },
    { key: 'notifications', label: 'Notifications', href: '/notifications' },
  ] as const

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

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <PanelsLeftBottom class="size-4" />
      <span>App shell</span>
    </CardTitle>
    <CardDescription>
      Feature-first shell: switch context on the left, keep board and dashboard in the center, move
      editing flows into the drawer.
    </CardDescription>
  </CardHeader>
  <CardContent class="flex flex-wrap gap-2">
    <Badge variant="outline">{organizations.length} orgs</Badge>
    <Badge variant="outline">{projects.length} projects</Badge>
    <Badge variant="outline">{workflowCount} workflows</Badge>
    <Badge variant="outline">{ticketCount} tickets</Badge>
  </CardContent>
</Card>

<OrganizationProjectSwitcher
  {organizations}
  {projects}
  {selectedOrgId}
  {selectedProjectId}
  {onSelectOrganization}
  {onSelectProject}
/>

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <ArrowRightLeft class="size-4" />
      <span>Navigation</span>
    </CardTitle>
    <CardDescription>
      Keep the same project context while moving between board, workflow, and agent surfaces.
    </CardDescription>
  </CardHeader>
  <CardContent class="grid gap-2">
    {#each pages as page}
      <a
        href={pageHref(page.href)}
        class={`rounded-2xl border px-4 py-3 text-sm font-medium transition ${
          selectedPage === page.key
            ? 'border-foreground/30 bg-foreground text-background'
            : 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
        }`}
      >
        {page.label}
      </a>
    {/each}
  </CardContent>
</Card>
