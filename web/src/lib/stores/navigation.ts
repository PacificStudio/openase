export type NavItem = {
  label: string
  href: string
  icon: string
  badge?: () => number | undefined
}

export const globalNav: NavItem[] = [
  { label: 'Dashboard', href: '/', icon: 'layout-dashboard' },
  { label: 'Approvals', href: '/approvals', icon: 'shield-check' },
]

export const projectNav: NavItem[] = [
  { label: 'Overview', href: '/overview', icon: 'gauge' },
  { label: 'Board', href: '/board', icon: 'kanban' },
  { label: 'Tickets', href: '/tickets', icon: 'ticket' },
  { label: 'Workflows', href: '/workflows', icon: 'workflow' },
  { label: 'Agents', href: '/agents', icon: 'bot' },
  { label: 'Machines', href: '/machines', icon: 'server' },
  { label: 'Activity', href: '/activity', icon: 'activity' },
  { label: 'Insights', href: '/insights', icon: 'bar-chart-3' },
  { label: 'Settings', href: '/settings', icon: 'settings' },
]
