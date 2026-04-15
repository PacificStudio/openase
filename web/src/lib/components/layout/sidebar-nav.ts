import { translate, type AppLocale, type TranslationKey } from '$lib/i18n'
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
  tourId?: string
}

type BuildProjectNavArgs = {
  currentPath: string
  currentOrgId: string | null
  currentProjectId: string | null
  agentCount: number
  locale: AppLocale
}

type ProjectNavSection = {
  labelKey: TranslationKey
  icon: Component
  section:
    | 'dashboard'
    | 'tickets'
    | 'agents'
    | 'machines'
    | 'updates'
    | 'activity'
    | 'workflows'
    | 'skills'
    | 'scheduled-jobs'
    | 'settings'
}

const projectSections = [
  { labelKey: 'nav.overview', icon: LayoutDashboard, section: 'dashboard' as const },
  { labelKey: 'nav.tickets', icon: TicketCheck, section: 'tickets' as const },
  { labelKey: 'nav.agents', icon: Bot, section: 'agents' as const },
  { labelKey: 'nav.machines', icon: Server, section: 'machines' as const },
  { labelKey: 'nav.updates', icon: MessageSquare, section: 'updates' as const },
  { labelKey: 'nav.activity', icon: Activity, section: 'activity' as const },
  { labelKey: 'nav.workflows', icon: Workflow, section: 'workflows' as const },
  { labelKey: 'nav.skills', icon: Wrench, section: 'skills' as const },
  { labelKey: 'nav.scheduledJobs', icon: CalendarClock, section: 'scheduled-jobs' as const },
  { labelKey: 'nav.settings', icon: Settings, section: 'settings' as const },
] as const satisfies ReadonlyArray<ProjectNavSection>

export function buildGlobalNav(
  currentPath: string,
  currentOrgId: string | null,
  locale: AppLocale,
): SidebarNavItem[] {
  const href = currentOrgId ? organizationPath(currentOrgId) : '/'
  return [
    {
      label: translate(locale, 'nav.dashboard'),
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
  locale,
}: BuildProjectNavArgs): SidebarNavItem[] {
  return projectSections.map(({ labelKey, icon, section }) => {
    const href = buildProjectHref(currentOrgId, currentProjectId, section)
    const active = buildProjectActive(currentPath, currentOrgId, currentProjectId, section)

    return {
      label: translate(locale, labelKey),
      href,
      icon,
      badge: section === 'agents' ? agentCount || undefined : undefined,
      active,
      tourId: `nav-${section}`,
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
