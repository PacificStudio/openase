import { organizationPath, projectPath } from '$lib/stores/app-context'
import {
  Activity,
  Bot,
  CalendarClock,
  LayoutDashboard,
  MessageSquare,
  Server,
  Settings,
  TicketCheck,
  Workflow,
  Wrench,
} from '@lucide/svelte'
import type { Component } from 'svelte'

export type SidebarNavItem = {
  label: string
  href: string
  icon: Component
  badge?: string | number
  active: boolean
}

type BuildProjectNavArgs = {
  currentPath: string
  currentOrgId: string | null
  currentProjectId: string | null
  agentCount: number
}

const projectSections = [
  { label: 'Overview', icon: LayoutDashboard, section: 'dashboard' as const },
  { label: 'Tickets', icon: TicketCheck, section: 'tickets' as const },
  { label: 'Agents', icon: Bot, section: 'agents' as const },
  { label: 'Machines', icon: Server, section: 'machines' as const },
  { label: 'Updates', icon: MessageSquare, section: 'updates' as const },
  { label: 'Activity', icon: Activity, section: 'activity' as const },
  { label: 'Workflows', icon: Workflow, section: 'workflows' as const },
  { label: 'Skills', icon: Wrench, section: 'skills' as const },
  { label: 'Scheduled Jobs', icon: CalendarClock, section: 'scheduled-jobs' as const },
  { label: 'Settings', icon: Settings, section: 'settings' as const },
]

export function buildGlobalNav(currentPath: string, currentOrgId: string | null): SidebarNavItem[] {
  const href = currentOrgId ? organizationPath(currentOrgId) : '/'
  return [
    {
      label: 'Dashboard',
      href,
      icon: LayoutDashboard,
      active: currentPath === href,
    },
  ]
}

export function buildProjectNav({
  currentPath,
  currentOrgId,
  currentProjectId,
  agentCount,
}: BuildProjectNavArgs): SidebarNavItem[] {
  return projectSections.map(({ label, icon, section }) => {
    const href = buildProjectHref(currentOrgId, currentProjectId, section)
    const active = buildProjectActive(currentPath, currentOrgId, currentProjectId, section)

    return {
      label,
      href,
      icon,
      badge: section === 'agents' ? agentCount || undefined : undefined,
      active,
    }
  })
}

function buildProjectHref(
  currentOrgId: string | null,
  currentProjectId: string | null,
  section: (typeof projectSections)[number]['section'],
) {
  if (!currentOrgId || !currentProjectId) {
    return currentOrgId ? organizationPath(currentOrgId) : '/orgs'
  }

  return projectPath(currentOrgId, currentProjectId, section)
}

function buildProjectActive(
  currentPath: string,
  currentOrgId: string | null,
  currentProjectId: string | null,
  section: (typeof projectSections)[number]['section'],
) {
  if (!currentOrgId || !currentProjectId) {
    return false
  }

  const href = projectPath(currentOrgId, currentProjectId, section)
  return section === 'dashboard' ? currentPath === href : currentPath.startsWith(href)
}
