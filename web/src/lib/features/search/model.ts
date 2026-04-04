import type { Agent, Organization, Project, Ticket, Workflow } from '$lib/api/contracts'
import {
  organizationPath,
  projectPath,
  projectSections,
  type ProjectSection,
} from '$lib/stores/app-context'
import { buildProjectAssistantCommand } from './command-items'
import type { SearchItem, SearchItemAction, SearchItemGroup, SearchItemKind } from './types'
import { searchItemGroupOrder } from './types'

type BuildSearchIndexInput = {
  organizations: Organization[]
  projects: Project[]
  currentOrg: Organization | null
  currentProject: Project | null
  currentSection: ProjectSection
  tickets: Ticket[]
  workflows: Workflow[]
  agents: Agent[]
  newTicketEnabled: boolean
}

const sectionLabels: Record<ProjectSection, string> = {
  dashboard: 'Dashboard',
  tickets: 'Tickets',
  agents: 'Agents',
  machines: 'Machines',
  updates: 'Updates',
  activity: 'Activity',
  workflows: 'Workflows',
  skills: 'Skills',
  'scheduled-jobs': 'Scheduled Jobs',
  settings: 'Settings',
}

export function buildSearchIndex(input: BuildSearchIndexInput): SearchItem[] {
  const items: SearchItem[] = []

  items.push(...buildCommandItems(input))
  items.push(...buildPageItems(input))
  items.push(...buildProjectItems(input))
  items.push(...buildOrganizationItems(input.organizations))
  items.push(...buildTicketItems(input))
  items.push(...buildWorkflowItems(input))
  items.push(...buildAgentItems(input))

  return items
}

export function groupSearchItems(items: SearchItem[]) {
  return searchItemGroupOrder
    .map((group) => ({
      heading: group,
      items: items.filter((item) => item.group === group),
    }))
    .filter((group) => group.items.length > 0)
}

function buildCommandItems({
  currentProject,
  newTicketEnabled,
}: BuildSearchIndexInput): SearchItem[] {
  const items: SearchItem[] = [
    createSearchItem({
      id: 'command-toggle-theme',
      group: 'Commands',
      kind: 'command',
      title: 'Toggle Theme',
      subtitle: 'Switch between light and dark workspace themes.',
      badge: 'Command',
      action: { kind: 'toggle_theme' },
      keywords: ['theme appearance color mode'],
    }),
  ]

  if (currentProject && newTicketEnabled) {
    items.unshift(
      createSearchItem({
        id: 'command-new-ticket',
        group: 'Commands',
        kind: 'command',
        title: 'New Ticket',
        subtitle: `Create a new ticket in ${currentProject.name}.`,
        badge: 'Command',
        action: { kind: 'new_ticket' },
        keywords: ['create ticket issue work item'],
      }),
    )
  }

  if (currentProject) {
    items.unshift(buildProjectAssistantCommand(currentProject))
  }

  return items
}

function buildPageItems({
  currentOrg,
  currentProject,
  currentSection,
}: BuildSearchIndexInput): SearchItem[] {
  if (!currentOrg || !currentProject) {
    return []
  }

  return projectSections.map((section) =>
    createSearchItem({
      id: `page-${section}`,
      group: 'Pages',
      kind: 'page',
      title: sectionLabel(section),
      subtitle: `Open ${sectionLabel(section)} for ${currentProject.name}.`,
      badge: section === currentSection ? 'Current' : 'Page',
      action: { kind: 'navigate', href: projectPath(currentOrg.id, currentProject.id, section) },
      keywords: [section, currentProject.name, currentOrg.name],
    }),
  )
}

function buildProjectItems({ currentOrg, projects }: BuildSearchIndexInput): SearchItem[] {
  if (!currentOrg) {
    return []
  }

  return projects.map((project) =>
    createSearchItem({
      id: `project-${project.id}`,
      group: 'Projects',
      kind: 'project',
      title: project.name,
      subtitle: project.description || `Open ${project.name} dashboard.`,
      badge: 'Project',
      action: { kind: 'navigate', href: projectPath(currentOrg.id, project.id) },
      keywords: [project.slug, project.status, currentOrg.name],
    }),
  )
}

function buildOrganizationItems(organizations: Organization[]): SearchItem[] {
  return organizations.map((organization) =>
    createSearchItem({
      id: `organization-${organization.id}`,
      group: 'Organizations',
      kind: 'organization',
      title: organization.name,
      subtitle: `Open ${organization.name} overview.`,
      badge: 'Org',
      action: { kind: 'navigate', href: organizationPath(organization.id) },
      keywords: [organization.slug],
    }),
  )
}

function buildTicketItems({
  currentProject,
  tickets,
  workflows,
}: BuildSearchIndexInput): SearchItem[] {
  if (!currentProject) {
    return []
  }

  const workflowNamesByID = new Map(workflows.map((workflow) => [workflow.id, workflow.name]))

  return tickets.map((ticket) =>
    createSearchItem({
      id: `ticket-${ticket.id}`,
      group: 'Tickets',
      kind: 'ticket',
      title: `${ticket.identifier} ${ticket.title}`,
      subtitle: [
        ticket.status_name,
        ticket.priority,
        workflowLabel(ticket.workflow_id, workflowNamesByID),
      ]
        .filter(Boolean)
        .join(' • '),
      badge: 'Ticket',
      action: { kind: 'open_ticket', ticketId: ticket.id },
      keywords: [
        currentProject.name,
        ticket.identifier,
        ticket.status_name,
        ticket.priority,
        ticket.type,
      ],
    }),
  )
}

function buildWorkflowItems({
  currentOrg,
  currentProject,
  workflows,
}: BuildSearchIndexInput): SearchItem[] {
  if (!currentOrg || !currentProject) {
    return []
  }

  return workflows.map((workflow) =>
    createSearchItem({
      id: `workflow-${workflow.id}`,
      group: 'Workflows',
      kind: 'workflow',
      title: workflow.name,
      subtitle: `${workflow.type} workflow in ${currentProject.name}.`,
      badge: workflow.is_active ? 'Active' : 'Workflow',
      action: {
        kind: 'navigate',
        href: projectPath(currentOrg.id, currentProject.id, 'workflows'),
      },
      keywords: [workflow.type, workflow.harness_path, currentProject.name],
    }),
  )
}

function buildAgentItems({
  currentOrg,
  currentProject,
  agents,
}: BuildSearchIndexInput): SearchItem[] {
  if (!currentOrg || !currentProject) {
    return []
  }

  return agents.map((agent) =>
    createSearchItem({
      id: `agent-${agent.id}`,
      group: 'Agents',
      kind: 'agent',
      title: agent.name,
      subtitle: [
        agent.runtime?.status ?? 'idle',
        agent.runtime?.runtime_phase ?? 'none',
        currentTicketLabel(agent.runtime?.current_ticket_id ?? null),
      ]
        .filter(Boolean)
        .join(' • '),
      badge: 'Agent',
      action: { kind: 'navigate', href: projectPath(currentOrg.id, currentProject.id, 'agents') },
      keywords: [currentProject.name, agent.runtime?.session_id ?? ''],
    }),
  )
}

function createSearchItem({
  id,
  group,
  kind,
  title,
  subtitle,
  badge,
  action,
  keywords,
}: {
  id: string
  group: SearchItemGroup
  kind: SearchItemKind
  title: string
  subtitle: string
  badge?: string
  action: SearchItemAction
  keywords: string[]
}) {
  return {
    id,
    group,
    kind,
    title,
    subtitle,
    badge,
    searchText: [title, subtitle, badge ?? '', ...keywords].filter(Boolean).join(' '),
    action,
  } satisfies SearchItem
}

function sectionLabel(section: ProjectSection) {
  return sectionLabels[section]
}

function workflowLabel(
  workflowID: string | null | undefined,
  workflowNamesByID: Map<string, string>,
) {
  if (!workflowID) {
    return 'No workflow'
  }

  return workflowNamesByID.get(workflowID) ?? 'Workflow assigned'
}

function currentTicketLabel(ticketID: string | null | undefined) {
  return ticketID ? `Ticket ${ticketID.slice(0, 8)}` : ''
}
