type ProjectShellShortcutsInput = {
  getProjectAssistantOpen: () => boolean
  setProjectAssistantOpen: (value: boolean) => void
  openSearch: () => void
  openProjectAssistant: () => void
}

export function bindProjectShellShortcuts(input: ProjectShellShortcutsInput) {
  const handleKeydown = (event: KeyboardEvent) => {
    if (event.defaultPrevented) {
      return
    }

    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault()
      input.openSearch()
    }

    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'i') {
      event.preventDefault()
      if (input.getProjectAssistantOpen()) {
        input.setProjectAssistantOpen(false)
      } else {
        input.openProjectAssistant()
      }
    }
  }

  window.addEventListener('keydown', handleKeydown)
  return () => {
    window.removeEventListener('keydown', handleKeydown)
  }
}
