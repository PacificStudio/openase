import type { TerminalInstance } from './terminal-manager-types'

export function createTerminalManagerInstanceState() {
  let instances = $state<TerminalInstance[]>([])
  let activeId = $state('')
  let panelOpen = $state(false)

  function updateInstance(id: string, updates: Partial<TerminalInstance>) {
    instances = instances.map((inst) => (inst.id === id ? { ...inst, ...updates } : inst))
  }

  function getActiveInstance(): TerminalInstance | undefined {
    return instances.find((inst) => inst.id === activeId)
  }

  function hasInstance(id: string) {
    return instances.some((inst) => inst.id === id)
  }

  function createInstance(createId: () => string) {
    const id = createId()
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
    const closingIndex = instances.findIndex((inst) => inst.id === id)
    instances = instances.filter((inst) => inst.id !== id)
    if (activeId === id) {
      const nextActive = instances[closingIndex] ?? instances[Math.max(closingIndex - 1, 0)]
      activeId = nextActive?.id ?? ''
    }
    if (instances.length === 0) {
      panelOpen = false
    }
  }

  function openPanel(seedInstance: () => void) {
    panelOpen = true
    if (instances.length === 0) {
      seedInstance()
    }
  }

  function togglePanel(seedInstance: () => void) {
    if (panelOpen) {
      panelOpen = false
      return
    }
    openPanel(seedInstance)
  }

  function closePanel() {
    panelOpen = false
  }

  function reset() {
    instances = []
    activeId = ''
    panelOpen = false
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
    closePanel,
    reset,
  }
}
