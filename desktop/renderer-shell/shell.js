function byId(id) {
  return document.getElementById(id)
}

function firstIssue(runtime) {
  const setup = runtime.setup ?? {}
  return (setup.preflight?.issues ?? [])[0] ?? (setup.lastApply?.issues ?? [])[0] ?? runtime.lastError ?? null
}

function setText(id, value) {
  const element = byId(id)
  if (element) {
    element.textContent = value
  }
}

function renderIssues(runtime) {
  const issuesRoot = byId('issues')
  if (!issuesRoot) {
    return
  }
  const issues = runtime.setup?.preflight?.issues ?? runtime.setup?.lastApply?.issues ?? []
  if (issues.length === 0) {
    issuesRoot.innerHTML = '<p class="inline-note">No blocking issues detected.</p>'
    return
  }

  issuesRoot.innerHTML = issues
    .map(
      (issue) => `
        <article class="issue-card">
          <div class="issue-code">${escapeHTML(issue.code)}</div>
          <h3>${escapeHTML(issue.title ?? issue.code)}</h3>
          <p>${escapeHTML(issue.message ?? '')}</p>
          ${issue.action ? `<p class="issue-action">${escapeHTML(issue.action)}</p>` : ''}
        </article>
      `,
    )
    .join('')
}

function renderDiagnostics(runtime) {
  const diagnosticsRoot = byId('cli-diagnostics')
  if (!diagnosticsRoot) {
    return
  }
  const diagnostics = runtime.setup?.bootstrap?.cli ?? []
  if (diagnostics.length === 0) {
    diagnosticsRoot.innerHTML = ''
    return
  }
  diagnosticsRoot.innerHTML = diagnostics
    .map(
      (item) => `
        <div class="diagnostic ${escapeHTML(item.status)}">
          <strong>${escapeHTML(item.name)}</strong>
          <span>${escapeHTML(item.status)}</span>
        </div>
      `,
    )
    .join('')
}

function populateSetupDefaults(runtime) {
  const defaults = runtime.setup?.bootstrap?.defaults ?? {}
  const manual = defaults.manual_database ?? {}
  const docker = defaults.docker_database ?? {}

  const manualDefaults = {
    'manual-host': manual.host ?? '127.0.0.1',
    'manual-port': String(manual.port ?? 5432),
    'manual-name': manual.name ?? 'openase',
    'manual-user': manual.user ?? 'openase',
    'manual-password': manual.password ?? '',
    'manual-ssl-mode': manual.ssl_mode ?? 'disable',
  }
  for (const [id, value] of Object.entries(manualDefaults)) {
    const element = byId(id)
    if (element && !element.value) {
      element.value = value
    }
  }

  const dockerDefaults = {
    'docker-container-name': docker.container_name ?? 'openase-local-postgres',
    'docker-database-name': docker.database_name ?? 'openase',
    'docker-user': docker.user ?? 'openase',
    'docker-port': String(docker.port ?? 15432),
    'docker-volume-name': docker.volume_name ?? 'openase-local-postgres-data',
    'docker-image': docker.image ?? 'postgres:16-alpine',
  }
  for (const [id, value] of Object.entries(dockerDefaults)) {
    const element = byId(id)
    if (element && !element.value) {
      element.value = value
    }
  }
}

function dockerReady(runtime) {
  const diagnostics = runtime.setup?.bootstrap?.cli ?? []
  const docker = diagnostics.find((item) => item.command === 'docker' || item.id === 'docker')
  return !docker || docker.status === 'ready'
}

async function handleManualSubmit(event) {
  event.preventDefault()
  await window.openaseDesktop.applySetup({
    database: {
      type: 'manual',
      manual: {
        host: byId('manual-host')?.value ?? '',
        port: Number(byId('manual-port')?.value ?? 5432),
        name: byId('manual-name')?.value ?? '',
        user: byId('manual-user')?.value ?? '',
        password: byId('manual-password')?.value ?? '',
        ssl_mode: byId('manual-ssl-mode')?.value ?? 'disable',
      },
    },
  })
}

async function handleDockerSubmit(event) {
  event.preventDefault()
  await window.openaseDesktop.applySetup({
    database: {
      type: 'docker',
      docker: {
        container_name: byId('docker-container-name')?.value ?? '',
        database_name: byId('docker-database-name')?.value ?? '',
        user: byId('docker-user')?.value ?? '',
        port: Number(byId('docker-port')?.value ?? 15432),
        volume_name: byId('docker-volume-name')?.value ?? '',
        image: byId('docker-image')?.value ?? '',
      },
    },
  })
}

async function main() {
  const runtime = await window.openaseDesktop.getRuntimeState()
  const issue = firstIssue(runtime)

  if (document.body.dataset.view === 'loading') {
    setText('message', runtime.loadingMessage || 'Booting the OpenASE service...')
    return
  }

  setText('config-path', runtime.paths.openaseConfigPath)

  if (document.body.dataset.view === 'error') {
    setText('message', runtime.lastError?.message || 'Unknown desktop startup error.')
  }

  if (document.body.dataset.view === 'setup') {
    setText('message', issue?.message || 'OpenASE needs a one-time PostgreSQL setup before the desktop app can start.')
    setText('issue-code', issue?.code || 'setup_required')
    populateSetupDefaults(runtime)
    renderIssues(runtime)
    renderDiagnostics(runtime)

    const dockerButton = byId('docker-submit')
    if (dockerButton && !dockerReady(runtime)) {
      dockerButton.disabled = true
      dockerButton.title = 'Docker is not ready on this machine.'
    }

    byId('manual-form')?.addEventListener('submit', (event) => {
      void handleManualSubmit(event)
    })
    byId('docker-form')?.addEventListener('submit', (event) => {
      void handleDockerSubmit(event)
    })
  }

  const actionMap = new Map([
    ['retry', () => window.openaseDesktop.restartService()],
    ['recheck', () => window.openaseDesktop.recheckSetup()],
    ['open-logs', () => window.openaseDesktop.openLogsDirectory()],
    ['open-data', () => window.openaseDesktop.openDataDirectory()],
    ['open-guide', () => window.openaseDesktop.openDesktopGuide()],
  ])

  for (const [id, action] of actionMap) {
    const element = byId(id)
    if (!element) {
      continue
    }
    element.addEventListener('click', () => {
      void action()
    })
  }
}

function escapeHTML(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

void main()
