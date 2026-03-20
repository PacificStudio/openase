import type { Organization, Project } from '$lib/api/contracts'

function createAppStore() {
  let currentOrg = $state<Organization | null>(null)
  let currentProject = $state<Project | null>(null)
  let sidebarCollapsed = $state(false)
  let rightPanelOpen = $state(false)
  let rightPanelContent = $state<{ type: string; id?: string } | null>(null)
  let sseStatus = $state<'idle' | 'connecting' | 'live' | 'retrying'>('idle')
  let theme = $state<'light' | 'dark'>('dark')

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
    get rightPanelOpen() {
      return rightPanelOpen
    },
    openRightPanel(content: { type: string; id?: string }) {
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
    toggleTheme() {
      theme = theme === 'dark' ? 'light' : 'dark'
      if (typeof document !== 'undefined') {
        document.documentElement.classList.toggle('dark', theme === 'dark')
      }
    },
  }
}

export const appStore = createAppStore()
