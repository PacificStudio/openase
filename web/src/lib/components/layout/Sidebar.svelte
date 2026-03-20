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
  import type { Organization, Project } from '$lib/features/workspace/types'

  let {
    organizations = [],
    projects = [],
    selectedOrgId = '',
    selectedProjectId = '',
    workflowCount = 0,
    ticketCount = 0,
    onSelectOrganization,
    onSelectProject,
  }: {
    organizations?: Organization[]
    projects?: Project[]
    selectedOrgId?: string
    selectedProjectId?: string
    workflowCount?: number
    ticketCount?: number
    onSelectOrganization?: (organization: Organization) => void
    onSelectProject?: (project: Project) => void
  } = $props()
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
      <span>Navigation pattern</span>
    </CardTitle>
    <CardDescription>
      This sidebar is reusable across future project, settings, and detail pages.
    </CardDescription>
  </CardHeader>
</Card>
