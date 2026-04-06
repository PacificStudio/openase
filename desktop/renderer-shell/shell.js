async function main() {
  const runtime = await window.openaseDesktop.getRuntimeState()
  const message = document.getElementById('message')
  const configPath = document.getElementById('config-path')

  if (message) {
    if (document.body.dataset.view === 'loading') {
      message.textContent = runtime.loadingMessage || 'Booting the OpenASE service...'
    } else {
      message.textContent = runtime.lastError?.message || 'Unknown desktop startup error.'
    }
  }

  if (configPath) {
    configPath.textContent = runtime.paths.openaseConfigPath
  }

  const actionMap = new Map([
    ['retry', () => window.openaseDesktop.restartService()],
    ['open-logs', () => window.openaseDesktop.openLogsDirectory()],
    ['open-data', () => window.openaseDesktop.openDataDirectory()],
    ['open-guide', () => window.openaseDesktop.openDesktopGuide()],
  ])

  for (const [id, action] of actionMap) {
    const element = document.getElementById(id)
    if (!element) {
      continue
    }
    element.addEventListener('click', () => {
      void action()
    })
  }
}

void main()
