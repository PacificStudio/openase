import type { TerminalInstance } from './terminal-manager-types'

let nextTerminalID = 1

export function createTerminalManagerPanelState() {
  let instances = $state<TerminalInstance[]>([])
  let activeId = $state('')
  let panelOpen = $state(false)

  function updateInstance(id: string, updates: Partial<TerminalInstance>) {
    instances = instances.map((inst) => (inst.id === id ? { ...inst, ...updates } : inst))
  }

  function getActiveInstance(): TerminalInstance | undefined {
    return instances.find((instance) => instance.id === activeId)
  }

  function hasInstance(id: string) {
    return instances.some((instance) => instance.id === id)
  }

  function createInstance(): string {
    const id = `term-${nextTerminalID++}`
    const index = instances.length + 1
    instances = [
      ...instances,
      {
        id,
        label: `Terminal ${index}`,
        status: 'idle',
        statusMessage: 'Connecting...',
        sessionID: '',
      },
    ]
    activeId = id
    return id
  }

  function removeInstance(id: string) {
    const closingIndex = instances.findIndex((instance) => instance.id === id)
    instances = instances.filter((instance) => instance.id !== id)
    if (activeId === id) {
      const nextActive = instances[closingIndex] ?? instances[Math.max(closingIndex - 1, 0)]
      activeId = nextActive?.id ?? ''
    }
    if (instances.length === 0) {
      panelOpen = false
    }
  }

  function openPanel() {
    panelOpen = true
    if (instances.length === 0) {
      createInstance()
    }
  }

  function togglePanel() {
    panelOpen = !panelOpen
    if (panelOpen && instances.length === 0) {
      createInstance()
    }
  }

  return {
    get instances() {
      return instances
    },
    get activeId() {
      return activeId
    },
    set activeId(id: string) {
      activeId = id
    },
    get panelOpen() {
      return panelOpen
    },
    updateInstance,
    getActiveInstance,
    hasInstance,
    createInstance,
    removeInstance,
    openPanel,
    togglePanel,
    closePanel: () => {
      panelOpen = false
    },
    reset: () => {
      instances = []
      activeId = ''
      panelOpen = false
    },
  }
}
