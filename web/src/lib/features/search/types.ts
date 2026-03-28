export type SearchItemAction =
  | { kind: 'navigate'; href: string }
  | { kind: 'open_ticket'; ticketId: string }
  | { kind: 'open_project_ai' }
  | { kind: 'new_ticket' }
  | { kind: 'toggle_theme' }

export type SearchItemGroup =
  | 'Commands'
  | 'Pages'
  | 'Projects'
  | 'Organizations'
  | 'Tickets'
  | 'Workflows'
  | 'Agents'

export type SearchItemKind =
  | 'command'
  | 'page'
  | 'project'
  | 'organization'
  | 'ticket'
  | 'workflow'
  | 'agent'

export type SearchItem = {
  id: string
  group: SearchItemGroup
  kind: SearchItemKind
  title: string
  subtitle: string
  badge?: string
  searchText: string
  action: SearchItemAction
}

export const searchItemGroupOrder: SearchItemGroup[] = [
  'Commands',
  'Pages',
  'Projects',
  'Organizations',
  'Tickets',
  'Workflows',
  'Agents',
]
