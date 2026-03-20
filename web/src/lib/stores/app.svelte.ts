import type { Organization, Project } from '$lib/api/contracts'

type AppPanelContent = { type: 'ticket'; id: string }
export type AppTheme = 'light' | 'dark'

const flipTheme = (theme: AppTheme): AppTheme => (theme === 'dark' ? 'light' : 'dark')

function createAppStore() {
  let currentOrg = $state<Organization | null>(null),
    currentProject = $state<Project | null>(null)
  let sidebarCollapsed = $state(false),
    newTicketDialogOpen = $state(false),
    rightPanelOpen = $state(false)
  let rightPanelContent = $state<AppPanelContent | null>(null)
  let sseStatus = $state<'idle' | 'connecting' | 'live' | 'retrying'>('idle')
  let theme = $state<AppTheme>('dark')
  return {
    get currentOrg() {
      return currentOrg
    },
    set currentOrg(v) {
      currentOrg = v
    },
    get currentProject() {
      return currentProject
    },
    set currentProject(v) {
      currentProject = v
    },
    get sidebarCollapsed() {
      return sidebarCollapsed
    },
    set sidebarCollapsed(v) {
      sidebarCollapsed = v
    },
    toggleSidebar() {
      sidebarCollapsed = !sidebarCollapsed
    },
    get newTicketDialogOpen() {
      return newTicketDialogOpen
    },
    set newTicketDialogOpen(v) {
      newTicketDialogOpen = v
    },
    openNewTicketDialog() {
      newTicketDialogOpen = true
    },
    closeNewTicketDialog() {
      newTicketDialogOpen = false
    },
    get rightPanelOpen() {
      return rightPanelOpen
    },
    openRightPanel(content: AppPanelContent) {
      rightPanelContent = content
      rightPanelOpen = true
    },
    closeRightPanel() {
      rightPanelOpen = false
      rightPanelContent = null
    },
    get rightPanelContent() {
      return rightPanelContent
    },
    get sseStatus() {
      return sseStatus
    },
    set sseStatus(v) {
      sseStatus = v
    },
    get theme() {
      return theme
    },
    setTheme(nextTheme: AppTheme) {
      theme = nextTheme
    },
    toggleTheme() {
      theme = flipTheme(theme)
    },
  }
}

export const appStore = createAppStore()
