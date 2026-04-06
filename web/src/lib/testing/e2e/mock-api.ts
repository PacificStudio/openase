/* eslint-disable max-lines-per-function, complexity, sonarjs/cognitive-complexity */

const ORG_ID = 'org-e2e'
const PROJECT_ID = 'project-e2e'
const LOCAL_MACHINE_ID = 'machine-local'
const GPU_MACHINE_ID = 'machine-gpu'
const DEFAULT_PROVIDER_ID = '1c7cae12-cafc-4359-90ed-5ab8a8574c63'
const CLAUDE_PROVIDER_ID = '22b906e4-d906-4b21-908a-9bf1e28d075a'
const GEMINI_PROVIDER_ID = 'ed7e5e5e-7d06-4685-8de6-45b502d7d393'
const OPENAI_PROVIDER_ID = '92619809-3a75-42a7-83e6-a6f93f6b3a6b'
const DEFAULT_AGENT_ID = 'agent-coder'
const DEFAULT_WORKFLOW_ID = 'workflow-coding'
const DEFAULT_REPO_ID = 'repo-todo'
const DEFAULT_JOB_ID = 'job-nightly'
const DEFAULT_TICKET_ID = 'ticket-1'
const DEFAULT_STATUS_IDS = {
  todo: 'status-todo',
  review: 'status-review',
  done: 'status-done',
} as const

type JsonValue = unknown
type MockGitHubCredentialScope = 'organization' | 'project'

type MockGitHubProbe = {
  state: string
  configured: boolean
  valid: boolean
  login?: string
  permissions: string[]
  repo_access: string
  checked_at?: string
  last_error: string
}

type MockGitHubSlot = {
  scope: MockGitHubCredentialScope
  configured: boolean
  source: string
  token_preview: string
  probe: MockGitHubProbe
}

type MockSecuritySettings = {
  project_id: string
  agent_tokens: {
    transport: string
    environment_variable: string
    token_prefix: string
    default_scopes: string[]
    supported_project_scopes: string[]
  }
  github: {
    effective: MockGitHubSlot
    organization: MockGitHubSlot
    project_override: MockGitHubSlot
  }
  webhooks: {
    connector_endpoint: string
  }
  secret_hygiene: {
    notification_channel_configs_redacted: boolean
  }
  approval_policies: {
    status: string
    rules_count: number
    summary: string
  }
  deferred: Array<{
    key: string
    title: string
    summary: string
  }>
}

type MockState = {
  organizations: Record<string, unknown>[]
  projects: Record<string, unknown>[]
  machines: Record<string, unknown>[]
  providers: Record<string, unknown>[]
  agents: Record<string, unknown>[]
  agentRuns: Record<string, unknown>[]
  activityEvents: Record<string, unknown>[]
  tickets: Record<string, unknown>[]
  statuses: Record<string, unknown>[]
  repos: Record<string, unknown>[]
  workflows: Record<string, unknown>[]
  harnessByWorkflowId: Record<
    string,
    {
      content: string
      path: string
      version: number
      history: Array<{ id: string; version: number; created_by: string; created_at: string }>
    }
  >
  scheduledJobs: Record<string, unknown>[]
  projectConversations: Record<string, unknown>[]
  projectConversationEntries: Record<string, unknown>[]
  skills: Record<string, unknown>[]
  builtinRoles: Record<string, unknown>[]
  securitySettingsByProjectId: Record<string, MockSecuritySettings>
  harnessVariables: { groups: Record<string, unknown>[] }
  counters: {
    machine: number
    repo: number
    workflow: number
    agent: number
    skill: number
    scheduledJob: number
    projectConversation: number
    projectConversationEntry: number
    projectConversationTurn: number
  }
}

const nowIso = '2026-03-27T10:00:00.000Z'
const encoder = new TextEncoder()

let mockState = createInitialState()
const projectConversationStreamControllers = new Map<
  string,
  Set<ReadableStreamDefaultController<Uint8Array>>
>()
const queuedProjectConversationFrames = new Map<string, string[]>()
const projectConversationMuxStreamControllers = new Map<
  string,
  Set<ReadableStreamDefaultController<Uint8Array>>
>()
const queuedProjectConversationMuxFrames = new Map<string, string[]>()

export function resetMockState() {
  mockState = createInitialState()
  projectConversationStreamControllers.clear()
  queuedProjectConversationFrames.clear()
  projectConversationMuxStreamControllers.clear()
  queuedProjectConversationMuxFrames.clear()
  return clone(mockState)
}

export async function handleMockApi(request: Request, url: URL): Promise<Response | null> {
  if (!url.pathname.startsWith('/api/v1/')) {
    return null
  }

  if (url.pathname === '/api/v1/auth/session' && request.method === 'GET') {
    return jsonResponse({
      auth_mode: 'disabled',
      authenticated: false,
      issuer_url: '',
      user: null,
      csrf_token: '',
      roles: [],
      permissions: [],
    })
  }

  if (url.pathname === '/api/v1/auth/logout' && request.method === 'POST') {
    return noContentResponse()
  }

  if (url.pathname === '/api/v1/__e2e__/reset' && request.method === 'POST') {
    resetMockState()
    return jsonResponse({ ok: true })
  }

  if (url.pathname === '/api/v1/__e2e__/seed-board' && request.method === 'POST') {
    const body = await readBody<{
      counts_by_status_id?: Record<string, number>
    }>(request)
    seedBoardState(body.counts_by_status_id ?? {})
    return jsonResponse({ ok: true })
  }

  if (url.pathname.endsWith('/stream') && !url.pathname.startsWith('/api/v1/chat/')) {
    return streamResponse()
  }

  if (url.pathname === '/api/v1/app-context' && request.method === 'GET') {
    return jsonResponse(buildAppContextPayload(url))
  }

  const segments = url.pathname
    .replace(/^\/api\/v1\//, '')
    .split('/')
    .filter(Boolean)

  if (segments[0] === 'orgs') {
    return handleOrgRoutes(request, segments)
  }
  if (segments[0] === 'projects') {
    return handleProjectRoutes(request, segments, url)
  }
  if (segments[0] === 'machines') {
    return handleMachineRoutes(request, segments)
  }
  if (segments[0] === 'providers') {
    return handleProviderRoutes(request, segments)
  }
  if (segments[0] === 'agents') {
    return handleAgentRoutes(request, segments)
  }
  if (segments[0] === 'workflows') {
    return handleWorkflowRoutes(request, segments)
  }
  if (segments[0] === 'chat') {
    return handleChatRoutes(request, segments)
  }
  if (segments[0] === 'skills') {
    return handleSkillRoutes(request, segments)
  }
  if (segments[0] === 'scheduled-jobs') {
    return handleScheduledJobRoutes(request, segments)
  }
  if (segments[0] === 'harness') {
    return handleHarnessRoutes(request, segments)
  }
  if (segments[0] === 'roles' && segments[1] === 'builtin') {
    return jsonResponse({ roles: clone(mockState.builtinRoles) })
  }

  return jsonResponse(
    { detail: `Mock route not implemented: ${request.method} ${url.pathname}` },
    404,
  )
}

async function handleOrgRoutes(request: Request, segments: string[]) {
  if (segments.length === 1 && request.method === 'GET') {
    return jsonResponse({ organizations: clone(mockState.organizations) })
  }

  const orgId = segments[1]
  if (orgId !== ORG_ID) {
    return notFound('Organization not found.')
  }

  if (segments[2] === 'projects' && request.method === 'GET') {
    return jsonResponse({
      projects: clone(mockState.projects.filter((project) => project.org_id === orgId)),
    })
  }
  if (segments[2] === 'providers' && request.method === 'GET') {
    return jsonResponse({
      providers: clone(mockState.providers.filter((provider) => provider.org_id === orgId)),
    })
  }
  if (segments[2] === 'machines') {
    if (request.method === 'GET') {
      return jsonResponse({
        machines: clone(mockState.machines.filter((machine) => machine.org_id === orgId)),
      })
    }
    if (request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const machine = createMachineRecord(body)
      mockState.machines.unshift(machine)
      return jsonResponse({ machine: clone(machine) }, 201)
    }
  }

  return notFound('Mock org route not found.')
}

async function handleProjectRoutes(request: Request, segments: string[], _url: URL) {
  const projectId = segments[1]
  if (projectId !== PROJECT_ID) {
    return notFound('Project not found.')
  }

  if (segments[2] === 'agents') {
    if (request.method === 'GET') {
      return jsonResponse({
        agents: clone(mockState.agents.filter((agent) => agent.project_id === projectId)),
      })
    }
    if (request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const providerId = asString(body.provider_id) ?? DEFAULT_PROVIDER_ID
      const provider = findById(mockState.providers, providerId)
      const agent = {
        id: `agent-${++mockState.counters.agent}`,
        project_id: projectId,
        provider_id: providerId,
        name: asString(body.name) ?? `Agent ${mockState.counters.agent}`,
        runtime_control_state: 'active',
        total_tickets_completed: 0,
        runtime: {
          status: 'idle',
          runtime_phase: 'none',
          active_run_count: 0,
          current_run_id: null,
          current_ticket_id: null,
          last_heartbeat_at: nowIso,
          runtime_started_at: null,
          session_id: '',
          last_error: '',
        },
      }
      mockState.agents.push(agent)
      if (provider) {
        provider.updated_at = nowIso
      }
      return jsonResponse({ agent: clone(agent) }, 201)
    }
  }

  if (segments[2] === 'agent-runs' && request.method === 'GET') {
    return jsonResponse({
      agent_runs: clone(mockState.agentRuns.filter((run) => run.project_id === projectId)),
    })
  }

  if (segments[2] === 'activity' && request.method === 'GET') {
    return jsonResponse({
      events: clone(mockState.activityEvents.filter((event) => event.project_id === projectId)),
    })
  }

  if (segments[2] === 'security-settings') {
    const security = resolveSecuritySettings(projectId)

    if (segments.length === 3 && request.method === 'GET') {
      return jsonResponse({ security: clone(security) })
    }

    if (segments[3] !== 'github-outbound-credential') {
      return notFound('Mock project route not found.')
    }

    if (segments.length === 4 && request.method === 'PUT') {
      const body = await readBody<Record<string, unknown>>(request)
      const scope = parseGitHubCredentialScope(asString(body.scope))
      if (!scope) {
        return jsonResponse({ detail: 'GitHub credential scope is required.' }, 400)
      }
      const slot = resolveGitHubCredentialSlot(security, scope)
      const token = asString(body.token)?.trim() || 'ghu_mock_manual_token'
      slot.configured = true
      slot.source = 'manual'
      slot.token_preview = previewToken(token)
      slot.probe = createConfiguredGitHubProbe('manual-user')
      syncEffectiveGitHubSlot(security)
      return jsonResponse({ security: clone(security) })
    }

    if (segments.length === 5 && segments[4] === 'import-gh-cli' && request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const scope = parseGitHubCredentialScope(asString(body.scope))
      if (!scope) {
        return jsonResponse({ detail: 'GitHub credential scope is required.' }, 400)
      }
      const slot = resolveGitHubCredentialSlot(security, scope)
      slot.configured = true
      slot.source = 'gh_cli_import'
      slot.token_preview = previewToken('ghu_mock_cli_token')
      slot.probe = createConfiguredGitHubProbe('octocat')
      syncEffectiveGitHubSlot(security)
      return jsonResponse({ security: clone(security) })
    }

    if (segments.length === 5 && segments[4] === 'retest' && request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const scope = parseGitHubCredentialScope(asString(body.scope))
      if (!scope) {
        return jsonResponse({ detail: 'GitHub credential scope is required.' }, 400)
      }
      const slot = resolveGitHubCredentialSlot(security, scope)
      if (!slot.configured) {
        return notFound('GitHub credential not configured.')
      }
      slot.probe = createConfiguredGitHubProbe(slot.probe.login || 'octocat')
      syncEffectiveGitHubSlot(security)
      return jsonResponse({ security: clone(security) })
    }

    if (segments.length === 4 && request.method === 'DELETE') {
      const scope = parseGitHubCredentialScope(new URL(request.url).searchParams.get('scope'))
      if (!scope) {
        return jsonResponse({ detail: 'GitHub credential scope is required.' }, 400)
      }
      const slot = resolveGitHubCredentialSlot(security, scope)
      slot.configured = false
      slot.source = ''
      slot.token_preview = ''
      slot.probe = createUnconfiguredGitHubProbe()
      syncEffectiveGitHubSlot(security)
      return jsonResponse({ security: clone(security) })
    }
  }

  if (segments[2] === 'tickets') {
    if (segments.length === 3 && request.method === 'GET') {
      return jsonResponse({
        tickets: clone(mockState.tickets.filter((ticket) => ticket.project_id === projectId)),
      })
    }
    if (segments.length === 3 && request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const statusId = asString(body.status_id) ?? DEFAULT_STATUS_IDS.todo
      const statusName =
        asString(mockState.statuses.find((status) => status.id === statusId)?.name) ?? 'Todo'
      const ticket = createMockTicketRecord({
        id: `ticket-${mockState.tickets.length + 1}`,
        identifier: `ASE-${200 + mockState.tickets.length + 1}`,
        title: asString(body.title) ?? `Ticket ${mockState.tickets.length + 1}`,
        description: asString(body.description) ?? '',
        statusId,
        statusName,
        workflowId: DEFAULT_WORKFLOW_ID,
      })
      mockState.tickets.unshift(ticket)
      return jsonResponse({ ticket: clone(ticket) }, 201)
    }
    if (segments.length === 5 && segments[4] === 'detail' && request.method === 'GET') {
      const payload = buildTicketDetailPayload(segments[3])
      if (!payload) {
        return notFound('Ticket detail not found.')
      }
      return jsonResponse(payload)
    }
    if (segments.length === 5 && segments[4] === 'runs' && request.method === 'GET') {
      return jsonResponse(buildTicketRunsPayload(segments[3]))
    }
    if (segments.length === 6 && segments[4] === 'runs' && request.method === 'GET') {
      const payload = buildTicketRunDetailPayload(segments[3], segments[5])
      if (!payload) {
        return notFound('Ticket run not found.')
      }
      return jsonResponse(payload)
    }
  }

  if (segments[2] === 'workflows') {
    if (request.method === 'GET') {
      return jsonResponse({
        workflows: clone(
          mockState.workflows.filter((workflow) => workflow.project_id === projectId),
        ),
      })
    }
    if (request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const nextId = `workflow-${++mockState.counters.workflow}`
      const workflow = {
        id: nextId,
        project_id: projectId,
        name: asString(body.name) ?? `Workflow ${mockState.counters.workflow}`,
        type: asString(body.type) ?? 'coding',
        agent_id: asString(body.agent_id) ?? DEFAULT_AGENT_ID,
        pickup_status_ids: asStringArray(body.pickup_status_ids),
        finish_status_ids: asStringArray(body.finish_status_ids),
        max_concurrent: asNumber(body.max_concurrent) ?? 0,
        max_retry_attempts: asNumber(body.max_retry_attempts) ?? 1,
        timeout_minutes: asNumber(body.timeout_minutes) ?? 30,
        stall_timeout_minutes: asNumber(body.stall_timeout_minutes) ?? 5,
        is_active: asBoolean(body.is_active) ?? true,
        harness_path: `.openase/harnesses/${slugify(asString(body.name) ?? nextId)}.md`,
        version: 1,
      }
      mockState.workflows.push(workflow)
      mockState.harnessByWorkflowId[nextId] = {
        content:
          asString(body.harness_content) ??
          defaultHarnessContent(asString(body.name) ?? `Workflow ${mockState.counters.workflow}`),
        path: workflow.harness_path,
        version: 1,
        history: [
          { id: `${nextId}-v1`, version: 1, created_by: 'user:manual', created_at: nowIso },
        ],
      }
      return jsonResponse({ workflow: clone(workflow) }, 201)
    }
  }

  if (segments[2] === 'repos') {
    if (segments.length === 3 && request.method === 'GET') {
      return jsonResponse({
        repos: clone(mockState.repos.filter((repo) => repo.project_id === projectId)),
      })
    }
    if (segments.length === 3 && request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const repo = createRepoRecord(body)
      mockState.repos.push(repo)
      return jsonResponse({ repo: clone(repo) }, 201)
    }
    if (segments.length === 4) {
      const repoId = segments[3]
      const repo = findById(mockState.repos, repoId)
      if (!repo) {
        return notFound('Repository not found.')
      }
      if (request.method === 'PATCH') {
        const body = await readBody<Record<string, unknown>>(request)
        applyRepoMutation(repo, body)
        return jsonResponse({ repo: clone(repo) })
      }
      if (request.method === 'DELETE') {
        mockState.repos = mockState.repos.filter((item) => item.id !== repoId)
        return jsonResponse({ repo: clone(repo) })
      }
    }
  }

  if (segments[2] === 'statuses' && request.method === 'GET') {
    return jsonResponse({
      statuses: clone(mockState.statuses.filter((status) => status.project_id === projectId)),
    })
  }

  if (segments[2] === 'skills' && request.method === 'GET') {
    return jsonResponse({
      skills: clone(mockState.skills.filter((skill) => skill.project_id === projectId)),
    })
  }

  if (segments[2] === 'skills' && request.method === 'POST') {
    const body = await readBody<Record<string, unknown>>(request)
    const name = asString(body.name)?.trim() || `skill-${++mockState.counters.skill}`
    const description = asString(body.description) ?? ''
    const content = asString(body.content) ?? ''
    const created = {
      id: `skill-${name}`,
      project_id: projectId,
      name,
      description,
      path: `/skills/${name}`,
      current_version: 1,
      is_builtin: false,
      is_enabled: asBoolean(body.is_enabled) ?? true,
      created_by: 'user:manual',
      created_at: nowIso,
      bound_workflows: [],
      content,
      files: [
        {
          path: 'SKILL.md',
          file_kind: 'entrypoint',
          media_type: 'text/markdown; charset=utf-8',
          encoding: 'utf8',
          is_executable: false,
          size_bytes: content.length,
          sha256: `sha-skill-${name}-1`,
          content,
          content_base64: 'ignored',
        },
      ],
      history: [
        { id: `skill-${name}-v1`, version: 1, created_by: 'user:manual', created_at: nowIso },
      ],
    }
    mockState.skills.unshift(created)
    return jsonResponse({ skill: clone(created), content }, 201)
  }

  if (segments[2] === 'scheduled-jobs') {
    if (request.method === 'GET') {
      return jsonResponse({
        scheduled_jobs: clone(
          mockState.scheduledJobs.filter((job) => job.project_id === projectId),
        ),
      })
    }
    if (request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const job = createScheduledJobRecord(body)
      mockState.scheduledJobs.push(job)
      return jsonResponse({ scheduled_job: clone(job) }, 201)
    }
  }

  return notFound('Mock project route not found.')
}

function buildAppContextPayload(url: URL) {
  const orgId = url.searchParams.get('org_id')
  const projectId = url.searchParams.get('project_id')

  return {
    organizations: clone(mockState.organizations),
    projects:
      orgId === ORG_ID
        ? clone(mockState.projects.filter((project) => project.org_id === orgId))
        : [],
    providers:
      orgId === ORG_ID
        ? clone(mockState.providers.filter((provider) => provider.org_id === orgId))
        : [],
    agent_count:
      projectId === PROJECT_ID
        ? mockState.agents.filter((agent) => agent.project_id === projectId).length
        : 0,
  }
}

async function handleMachineRoutes(request: Request, segments: string[]) {
  const machineId = segments[1]
  const machine = findById(mockState.machines, machineId)
  if (!machine) {
    return notFound('Machine not found.')
  }

  if (segments.length === 2) {
    if (request.method === 'GET') {
      return jsonResponse({ machine: clone(machine) })
    }
    if (request.method === 'PATCH') {
      const body = await readBody<Record<string, unknown>>(request)
      applyMachineMutation(machine, body)
      return jsonResponse({ machine: clone(machine) })
    }
    if (request.method === 'DELETE') {
      mockState.machines = mockState.machines.filter((item) => item.id !== machineId)
      return jsonResponse({ machine: clone(machine) })
    }
  }

  if (segments[2] === 'resources' && request.method === 'GET') {
    return jsonResponse({ resources: clone(machine.resources as Record<string, unknown>) })
  }

  if (segments[2] === 'test' && request.method === 'POST') {
    machine.last_heartbeat_at = nowIso
    machine.status = 'online'
    const mergedResources = {
      ...(asObject(machine.resources) ?? {}),
      transport: machine.host === 'local' ? 'local' : 'ssh',
      checked_at: nowIso,
      last_success: true,
      connection_test: {
        checked_at: nowIso,
        transport: machine.host === 'local' ? 'local' : 'ssh',
        last_success: true,
        output: `Connected to ${machine.name} successfully.`,
      },
    }
    machine.resources = mergedResources
    const probe = {
      checked_at: nowIso,
      transport: machine.host === 'local' ? 'local' : 'ssh',
      output: `Connected to ${machine.name} successfully.`,
    }
    return jsonResponse({ machine: clone(machine), probe })
  }

  return notFound('Mock machine route not found.')
}

async function handleProviderRoutes(request: Request, segments: string[]) {
  const providerId = segments[1]
  const provider = findById(mockState.providers, providerId)
  if (!provider) {
    return notFound('Provider not found.')
  }
  if (request.method !== 'PATCH') {
    return notFound('Mock provider route not found.')
  }

  const body = await readBody<Record<string, unknown>>(request)
  if (body.machine_id) {
    const machine = findById(mockState.machines, asString(body.machine_id) ?? '')
    if (machine) {
      provider.machine_id = machine.id
      provider.machine_name = machine.name
      provider.machine_host = machine.host
      provider.machine_status = machine.status
      provider.machine_workspace_root = machine.workspace_root ?? null
    }
  }
  provider.name = asString(body.name) ?? provider.name
  provider.adapter_type = asString(body.adapter_type) ?? provider.adapter_type
  provider.cli_command = asString(body.cli_command) ?? provider.cli_command
  provider.cli_args = asStringArray(body.cli_args)
  provider.model_name = asString(body.model_name) ?? provider.model_name
  provider.model_temperature = asNumber(body.model_temperature) ?? provider.model_temperature
  provider.model_max_tokens = asNumber(body.model_max_tokens) ?? provider.model_max_tokens
  provider.max_parallel_runs = asNumber(body.max_parallel_runs) ?? provider.max_parallel_runs
  provider.cost_per_input_token =
    asNumber(body.cost_per_input_token) ?? provider.cost_per_input_token
  provider.cost_per_output_token =
    asNumber(body.cost_per_output_token) ?? provider.cost_per_output_token
  provider.auth_config = asObject(body.auth_config) ?? {}

  return jsonResponse({ provider: clone(provider) })
}

async function handleAgentRoutes(request: Request, segments: string[]) {
  const agentId = segments[1]
  const agent = findById(mockState.agents, agentId)
  if (!agent) {
    return notFound('Agent not found.')
  }

  if (segments[2] === 'pause' && request.method === 'POST') {
    agent.runtime_control_state = 'paused'
    return jsonResponse({ agent: clone(agent) })
  }
  if (segments[2] === 'interrupt' && request.method === 'POST') {
    agent.runtime_control_state = 'interrupt_requested'
    return jsonResponse({ agent: clone(agent) })
  }
  if (segments[2] === 'resume' && request.method === 'POST') {
    agent.runtime_control_state = 'active'
    return jsonResponse({ agent: clone(agent) })
  }

  return notFound('Mock agent route not found.')
}

async function handleWorkflowRoutes(request: Request, segments: string[]) {
  const workflowId = segments[1]
  const workflow = findById(mockState.workflows, workflowId)
  if (!workflow) {
    return notFound('Workflow not found.')
  }

  if (segments[2] === 'harness') {
    if (segments[3] === 'history' && request.method === 'GET') {
      return jsonResponse({
        history: clone(mockState.harnessByWorkflowId[workflowId].history),
      })
    }
    if (request.method === 'GET') {
      const harness = mockState.harnessByWorkflowId[workflowId]
      return jsonResponse({
        harness: {
          workflow_id: workflowId,
          content: harness.content,
          path: harness.path,
          version: harness.version,
        },
      })
    }
    if (request.method === 'PUT') {
      const body = await readBody<Record<string, unknown>>(request)
      const current = mockState.harnessByWorkflowId[workflowId]
      const nextVersion = current.version + 1
      const content = asString(body.content) ?? current.content
      mockState.harnessByWorkflowId[workflowId] = {
        content,
        path: current.path,
        version: nextVersion,
        history: [
          {
            id: `${workflowId}-v${nextVersion}`,
            version: nextVersion,
            created_by: 'user:manual',
            created_at: nowIso,
          },
          ...current.history,
        ],
      }
      workflow.version = nextVersion
      return jsonResponse({
        harness: {
          workflow_id: workflowId,
          content,
          path: current.path,
          version: nextVersion,
        },
      })
    }
  }

  if (segments[2] === 'skills' && request.method === 'POST') {
    const body = await readBody<Record<string, unknown>>(request)
    const skillPaths = asStringArray(body.skills)
    const binding = segments[3] === 'bind'

    mockState.skills = mockState.skills.map((skill) => {
      if (
        !skillPaths.includes(skill.path as string) &&
        !skillPaths.includes(skill.name as string)
      ) {
        return skill
      }
      const existing = Array.isArray(skill.bound_workflows)
        ? (skill.bound_workflows as Record<string, unknown>[])
        : []
      const nextBound = binding
        ? dedupeById([...existing, { id: workflowId }])
        : existing.filter((item) => item.id !== workflowId)
      return { ...skill, bound_workflows: nextBound }
    })

    const harness = mockState.harnessByWorkflowId[workflowId]
    return jsonResponse({
      harness: {
        workflow_id: workflowId,
        content: harness.content,
        path: harness.path,
        version: harness.version,
      },
    })
  }

  return notFound('Mock workflow route not found.')
}

async function handleChatRoutes(request: Request, segments: string[]) {
  if (segments.length === 1) {
    if (request.method === 'POST') {
      return streamResponse()
    }
    return notFound('Mock chat route not found.')
  }

  if (segments[1] !== 'conversations') {
    if (
      segments[1] === 'projects' &&
      segments.length === 5 &&
      segments[3] === 'conversations' &&
      segments[4] === 'stream' &&
      request.method === 'GET'
    ) {
      if (segments[2] !== PROJECT_ID) {
        return notFound('Project not found.')
      }
      return projectConversationMuxStreamResponse(segments[2])
    }
    return notFound('Mock chat route not found.')
  }

  if (segments.length === 2 && request.method === 'GET') {
    const search = new URL(request.url).searchParams
    const projectId = search.get('project_id') ?? PROJECT_ID
    const providerId = search.get('provider_id')
    return jsonResponse({
      conversations: clone(
        mockState.projectConversations.filter((conversation) => {
          if (conversation.project_id !== projectId) {
            return false
          }
          if (providerId && conversation.provider_id !== providerId) {
            return false
          }
          return true
        }),
      ),
    })
  }

  if (segments.length === 2 && request.method === 'POST') {
    const body = await readBody<Record<string, unknown>>(request)
    const providerId = asString(body.provider_id) ?? DEFAULT_PROVIDER_ID
    const conversation = {
      id: `conversation-e2e-${++mockState.counters.projectConversation}`,
      project_id: PROJECT_ID,
      user_id: 'chat-user-e2e',
      source: 'project_sidebar',
      provider_id: providerId,
      status: 'active',
      rolling_summary: '',
      last_activity_at: nowIso,
      created_at: nowIso,
      updated_at: nowIso,
    }
    mockState.projectConversations.unshift(conversation)
    return jsonResponse(
      {
        conversation: clone(conversation),
      },
      201,
    )
  }

  if (segments.length === 3 && request.method === 'GET') {
    const conversation = findById(mockState.projectConversations, segments[2])
    if (!conversation) {
      return notFound('Project conversation not found.')
    }
    return jsonResponse({ conversation: clone(conversation) })
  }

  if (segments[3] === 'workspace-diff' && request.method === 'GET') {
    const conversation = findById(mockState.projectConversations, segments[2])
    if (!conversation) {
      return notFound('Project conversation not found.')
    }
    return jsonResponse({
      workspace_diff: buildMockProjectConversationWorkspaceDiff(segments[2]),
    })
  }

  if (segments[3] === 'entries' && request.method === 'GET') {
    return jsonResponse({
      entries: clone(
        mockState.projectConversationEntries.filter(
          (entry) => entry.conversation_id === segments[2],
        ),
      ),
    })
  }

  if (segments[3] === 'stream' && request.method === 'GET') {
    return projectConversationStreamResponse(segments[2])
  }

  if (segments[3] === 'turns' && request.method === 'POST') {
    const conversation = findById(mockState.projectConversations, segments[2])
    if (!conversation) {
      return notFound('Project conversation not found.')
    }

    const body = await readBody<Record<string, unknown>>(request)
    const message = asString(body.message) ?? ''
    const focus = asObject(body.focus)
    const ticketFocus =
      asString(focus?.kind) === 'ticket'
        ? {
            identifier: asString(focus?.ticket_identifier) ?? 'ASE-unknown',
            status: asString(focus?.ticket_status) ?? '',
            retryPaused: asBoolean(focus?.ticket_retry_paused) ?? false,
            pauseReason: asString(focus?.ticket_pause_reason) ?? '',
            repoScopes: Array.isArray(focus?.ticket_repo_scopes) ? focus.ticket_repo_scopes : [],
            hookHistory: Array.isArray(focus?.ticket_hook_history) ? focus.ticket_hook_history : [],
            currentRun: asObject(focus?.ticket_current_run),
          }
        : null
    const turnIndex = ++mockState.counters.projectConversationTurn
    const turnId = `turn-e2e-${turnIndex}`
    const userEntry = {
      id: `entry-e2e-${++mockState.counters.projectConversationEntry}`,
      conversation_id: segments[2],
      turn_id: turnId,
      seq: nextProjectConversationSeq(segments[2]),
      kind: 'user_message',
      payload: { content: message },
      created_at: shiftedIso(turnIndex),
    }
    const assistantPayload = {
      kind: 'assistant_message',
      payload: {
        content: buildMockProjectConversationReply(message, ticketFocus),
      },
    }
    const assistantEntry = {
      id: `entry-e2e-${++mockState.counters.projectConversationEntry}`,
      conversation_id: segments[2],
      turn_id: turnId,
      seq: userEntry.seq + 1,
      kind: assistantPayload.kind,
      payload: assistantPayload.payload,
      created_at: shiftedIso(turnIndex + 1),
    }
    mockState.projectConversationEntries.push(userEntry, assistantEntry)
    conversation.last_activity_at = assistantEntry.created_at
    conversation.updated_at = assistantEntry.created_at
    conversation.rolling_summary = message

    const sessionSentAt = shiftedIso(turnIndex)
    queueOrBroadcastProjectConversationEvent(
      segments[2],
      'session',
      {
        conversation_id: segments[2],
        runtime_state: 'active',
      },
      sessionSentAt,
    )
    setTimeout(() => {
      queueOrBroadcastProjectConversationEvent(
        segments[2],
        'message',
        {
          type: 'text',
          content: String(assistantEntry.payload.content ?? ''),
        },
        shiftedIso(turnIndex + 1),
      )
    }, 25)
    setTimeout(() => {
      queueOrBroadcastProjectConversationEvent(
        segments[2],
        'turn_done',
        {
          conversation_id: segments[2],
          turn_id: turnId,
          cost_usd: 0.01,
        },
        shiftedIso(turnIndex + 2),
      )
    }, 50)

    return jsonResponse(
      {
        turn: {
          id: turnId,
          turn_index: turnIndex,
          status: 'started',
        },
      },
      202,
    )
  }

  return notFound('Mock chat route not found.')
}

async function handleSkillRoutes(request: Request, segments: string[]) {
  const skillId = segments[1]
  const skill = findById(mockState.skills, skillId)
  if (!skill) {
    return notFound('Skill not found.')
  }

  const boundWorkflowRefs = (entries: Record<string, unknown>[] | undefined) =>
    (entries ?? []).map((entry) => {
      const workflowId = asString(entry.id)
      const workflow = workflowId ? findById(mockState.workflows, workflowId) : null
      return workflow ? { id: workflow.id, name: workflow.name } : { id: workflowId }
    })

  const currentFiles = () =>
    clone(
      ((skill.files as Record<string, unknown>[] | undefined) ?? [
        {
          path: 'SKILL.md',
          file_kind: 'entrypoint',
          media_type: 'text/markdown; charset=utf-8',
          encoding: 'utf8',
          is_executable: false,
          size_bytes: asString(skill.content)?.length ?? 0,
          sha256: `sha-${skillId}-${skill.current_version ?? 1}`,
          content: skill.content,
          content_base64: 'ignored',
        },
      ]) as Record<string, unknown>[],
    )

  if (segments[2] === 'history' && request.method === 'GET') {
    return jsonResponse({
      history: clone((skill.history as Record<string, unknown>[]) ?? []),
    })
  }

  if (segments[2] === 'files' && request.method === 'GET') {
    return jsonResponse({
      files: currentFiles(),
    })
  }

  if (segments.length === 2 && request.method === 'GET') {
    return jsonResponse({
      skill: clone({
        ...skill,
        bound_workflows: boundWorkflowRefs(skill.bound_workflows as Record<string, unknown>[]),
      }),
      content: skill.content,
      history: clone((skill.history as Record<string, unknown>[]) ?? []),
    })
  }

  if (segments.length === 2 && request.method === 'PUT') {
    const body = await readBody<Record<string, unknown>>(request)
    const nextVersion = (asNumber(skill.current_version) ?? 0) + 1
    const nextContent = asString(body.content) ?? asString(skill.content) ?? ''
    const nextDescription = asString(body.description) ?? asString(skill.description) ?? ''
    const requestFiles = (asObjectArray(body.files) ?? []).map((file, index) => ({
      path: asString(file.path) ?? `file-${index}`,
      file_kind: asString(file.file_kind) ?? (index === 0 ? 'entrypoint' : 'reference'),
      media_type: asString(file.media_type) ?? 'text/markdown; charset=utf-8',
      encoding: 'utf8',
      is_executable: asBoolean(file.is_executable) ?? false,
      size_bytes: (asString(file.content) ?? asString(file.content_base64) ?? '').length,
      sha256: `sha-${skillId}-${nextVersion}-${index}`,
      content: asString(file.content) ?? decodeBase64UTF8(asString(file.content_base64) ?? ''),
      content_base64: asString(file.content_base64) ?? 'ignored',
    }))
    skill.description = nextDescription
    skill.content = nextContent
    skill.current_version = nextVersion
    skill.files =
      requestFiles.length > 0
        ? requestFiles
        : [
            {
              path: 'SKILL.md',
              file_kind: 'entrypoint',
              media_type: 'text/markdown; charset=utf-8',
              encoding: 'utf8',
              is_executable: false,
              size_bytes: nextContent.length,
              sha256: `sha-${skillId}-${nextVersion}`,
              content: nextContent,
              content_base64: 'ignored',
            },
          ]
    skill.history = [
      {
        id: `${skillId}-v${nextVersion}`,
        version: nextVersion,
        created_by: 'user:manual',
        created_at: nowIso,
      },
      ...((skill.history as Record<string, unknown>[]) ?? []),
    ]
    return jsonResponse({
      skill: clone({
        ...skill,
        bound_workflows: boundWorkflowRefs(skill.bound_workflows as Record<string, unknown>[]),
      }),
    })
  }

  if (segments.length === 2 && request.method === 'DELETE') {
    mockState.skills = mockState.skills.filter((item) => item.id !== skillId)
    return jsonResponse({ skill: clone(skill) })
  }

  if (segments[2] === 'enable' && request.method === 'POST') {
    skill.is_enabled = true
    return jsonResponse({ skill: clone(skill) })
  }

  if (segments[2] === 'disable' && request.method === 'POST') {
    skill.is_enabled = false
    return jsonResponse({ skill: clone(skill) })
  }

  if ((segments[2] === 'bind' || segments[2] === 'unbind') && request.method === 'POST') {
    const body = await readBody<Record<string, unknown>>(request)
    const workflowIds = asStringArray(body.workflow_ids)
    const existing = (skill.bound_workflows as Record<string, unknown>[] | undefined) ?? []
    const nextBound =
      segments[2] === 'bind'
        ? dedupeById([...existing, ...workflowIds.map((workflowId) => ({ id: workflowId }))])
        : existing.filter((item) => !workflowIds.includes(asString(item.id) ?? ''))
    skill.bound_workflows = nextBound
    return jsonResponse({
      skill: clone({
        ...skill,
        bound_workflows: boundWorkflowRefs(nextBound),
      }),
    })
  }

  return notFound('Mock skill route not found.')
}

async function handleScheduledJobRoutes(request: Request, segments: string[]) {
  const jobId = segments[1]
  const job = findById(mockState.scheduledJobs, jobId)
  if (!job) {
    return notFound('Scheduled job not found.')
  }

  if (segments.length === 2 && request.method === 'PATCH') {
    const body = await readBody<Record<string, unknown>>(request)
    applyScheduledJobMutation(job, body)
    return jsonResponse({ scheduled_job: clone(job) })
  }
  if (segments.length === 2 && request.method === 'DELETE') {
    mockState.scheduledJobs = mockState.scheduledJobs.filter((item) => item.id !== jobId)
    return jsonResponse({ scheduled_job: clone(job) })
  }
  if (segments[2] === 'trigger' && request.method === 'POST') {
    job.last_run_at = nowIso
    job.next_run_at = '2026-03-28T02:00:00.000Z'
    return jsonResponse({ scheduled_job: clone(job) })
  }

  return notFound('Mock scheduled job route not found.')
}

async function handleHarnessRoutes(request: Request, segments: string[]) {
  if (segments[1] === 'validate' && request.method === 'POST') {
    const body = await readBody<Record<string, unknown>>(request)
    const content = asString(body.content) ?? ''
    const issues =
      content.trim().length === 0
        ? [{ level: 'error', message: 'Harness content must not be empty.' }]
        : []
    return jsonResponse({ valid: issues.length === 0, issues })
  }
  if (segments[1] === 'variables' && request.method === 'GET') {
    return jsonResponse(clone(mockState.harnessVariables))
  }

  return notFound('Mock harness route not found.')
}

function createInitialState(): MockState {
  const organizations = [
    {
      id: ORG_ID,
      name: 'E2E Org',
      slug: 'e2e-org',
      default_agent_provider_id: DEFAULT_PROVIDER_ID,
    },
  ]

  const projects = [
    {
      id: PROJECT_ID,
      org_id: ORG_ID,
      name: 'TodoApp',
      slug: 'todo-app',
      description: 'Playwright fixture project',
      status: 'active',
      default_agent_provider_id: DEFAULT_PROVIDER_ID,
      max_concurrent_agents: 0,
    },
  ]

  const machines = [
    {
      id: LOCAL_MACHINE_ID,
      org_id: ORG_ID,
      name: 'local',
      host: 'local',
      port: 22,
      reachability_mode: 'local',
      execution_mode: 'local_process',
      execution_capabilities: ['probe', 'workspace_prepare', 'artifact_sync', 'process_streaming'],
      ssh_helper_enabled: false,
      ssh_helper_required: false,
      connection_mode: 'local',
      transport_capabilities: ['probe', 'workspace_prepare', 'artifact_sync', 'process_streaming'],
      ssh_user: '',
      ssh_key_path: '',
      advertised_endpoint: null,
      daemon_status: {
        registered: true,
        last_registered_at: nowIso,
        current_session_id: null,
        session_state: 'connected',
      },
      detected_os: 'linux',
      detected_arch: 'amd64',
      detection_status: 'ok',
      detection_message: 'Detected amd64 on Linux.',
      channel_credential: {
        kind: 'none',
        token_id: null,
        certificate_id: null,
      },
      description: 'Seeded local runner',
      labels: ['local', 'default'],
      status: 'online',
      workspace_root: '~/.openase/workspace',
      agent_cli_path: '/usr/local/bin/codex',
      env_vars: ['OPENASE_ENV=dev'],
      last_heartbeat_at: nowIso,
      resources: machineResourceSnapshot('local', {
        cpuUsagePercent: 18,
        memoryTotalGB: 16,
        memoryUsedGB: 5.5,
        memoryAvailableGB: 10.5,
        diskTotalGB: 512,
        diskAvailableGB: 320,
        gpuDispatchable: false,
        gpus: [],
      }),
    },
    {
      id: GPU_MACHINE_ID,
      org_id: ORG_ID,
      name: 'gpu-runner-01',
      host: '10.0.0.42',
      port: 22,
      reachability_mode: 'direct_connect',
      execution_mode: 'ssh_compat',
      execution_capabilities: ['probe', 'workspace_prepare', 'artifact_sync', 'process_streaming'],
      ssh_helper_enabled: true,
      ssh_helper_required: true,
      connection_mode: 'ssh',
      transport_capabilities: ['probe', 'workspace_prepare', 'artifact_sync', 'process_streaming'],
      ssh_user: 'openase',
      ssh_key_path: '~/.ssh/openase_rsa',
      advertised_endpoint: null,
      daemon_status: {
        registered: false,
        last_registered_at: null,
        current_session_id: null,
        session_state: 'unknown',
      },
      detected_os: 'linux',
      detected_arch: 'amd64',
      detection_status: 'ok',
      detection_message: 'Detected amd64 on Linux.',
      channel_credential: {
        kind: 'none',
        token_id: null,
        certificate_id: null,
      },
      description: 'Primary remote runner',
      labels: ['gpu', 'remote'],
      status: 'online',
      workspace_root: '/srv/openase/workspace',
      agent_cli_path: '/usr/bin/codex',
      env_vars: ['CUDA_VISIBLE_DEVICES=0'],
      last_heartbeat_at: nowIso,
      resources: machineResourceSnapshot('ssh', {
        cpuUsagePercent: 46,
        memoryTotalGB: 64,
        memoryUsedGB: 21.4,
        memoryAvailableGB: 42.6,
        diskTotalGB: 1024,
        diskAvailableGB: 612,
        gpuDispatchable: true,
        gpus: [
          {
            index: 0,
            name: 'NVIDIA A100',
            memory_total_gb: 80,
            memory_used_gb: 22,
            utilization_percent: 54,
          },
        ],
      }),
    },
  ]

  const providers = [
    {
      id: DEFAULT_PROVIDER_ID,
      organization_id: ORG_ID,
      org_id: ORG_ID,
      machine_id: LOCAL_MACHINE_ID,
      machine_name: 'local',
      machine_host: 'local',
      machine_status: 'online',
      machine_workspace_root: '~/.openase/workspace',
      name: 'Fake Codex Validation Provider',
      adapter_type: 'codex-app-server',
      availability_state: 'available',
      available: true,
      availability_checked_at: '2026-04-01T02:54:26Z',
      availability_reason: null,
      capabilities: {
        ephemeral_chat: {
          state: 'available',
          reason: null,
        },
        harness_ai: {
          state: 'available',
          reason: null,
        },
        skill_ai: {
          state: 'available',
          reason: null,
        },
      },
      cli_command: 'python3',
      cli_args: ['/home/user/workspace/openase/scripts/dev/fake_codex_app_server.py'],
      auth_config: {},
      model_name: 'gpt-5.4',
      model_temperature: 0,
      model_max_tokens: 16384,
      max_parallel_runs: 0,
      cost_per_input_token: 0,
      cost_per_output_token: 0,
      pricing_config: {},
    },
    {
      id: CLAUDE_PROVIDER_ID,
      organization_id: ORG_ID,
      org_id: ORG_ID,
      machine_id: LOCAL_MACHINE_ID,
      machine_name: 'local',
      machine_host: 'local',
      machine_status: 'online',
      machine_workspace_root: '~/.openase/workspace',
      name: 'Claude Code',
      adapter_type: 'claude-code-cli',
      availability_state: 'available',
      available: true,
      availability_checked_at: '2026-04-01T02:54:26Z',
      availability_reason: null,
      capabilities: {
        ephemeral_chat: {
          state: 'available',
          reason: null,
        },
        harness_ai: {
          state: 'available',
          reason: null,
        },
        skill_ai: {
          state: 'unsupported',
          reason: 'skill_ai_requires_codex',
        },
      },
      cli_command: 'claude',
      cli_args: [],
      auth_config: {},
      model_name: 'claude-opus-4-6',
      model_temperature: 0,
      model_max_tokens: 16384,
      max_parallel_runs: 0,
      cost_per_input_token: 0,
      cost_per_output_token: 0,
      pricing_config: {},
    },
    {
      id: GEMINI_PROVIDER_ID,
      organization_id: ORG_ID,
      org_id: ORG_ID,
      machine_id: LOCAL_MACHINE_ID,
      machine_name: 'local',
      machine_host: 'local',
      machine_status: 'online',
      machine_workspace_root: '~/.openase/workspace',
      name: 'Gemini CLI',
      adapter_type: 'gemini-cli',
      availability_state: 'available',
      available: true,
      availability_checked_at: '2026-04-01T02:54:26Z',
      availability_reason: null,
      capabilities: {
        ephemeral_chat: {
          state: 'available',
          reason: null,
        },
        harness_ai: {
          state: 'available',
          reason: null,
        },
        skill_ai: {
          state: 'unsupported',
          reason: 'skill_ai_requires_codex',
        },
      },
      cli_command: 'gemini',
      cli_args: [],
      auth_config: {},
      model_name: 'gemini-2.5-pro',
      model_temperature: 0,
      model_max_tokens: 16384,
      max_parallel_runs: 0,
      cost_per_input_token: 0,
      cost_per_output_token: 0,
      pricing_config: {},
    },
    {
      id: OPENAI_PROVIDER_ID,
      organization_id: ORG_ID,
      org_id: ORG_ID,
      machine_id: LOCAL_MACHINE_ID,
      machine_name: 'local',
      machine_host: 'local',
      machine_status: 'online',
      machine_workspace_root: '~/.openase/workspace',
      name: 'OpenAI Codex',
      adapter_type: 'codex-app-server',
      availability_state: 'available',
      available: true,
      availability_checked_at: '2026-04-01T02:54:26Z',
      availability_reason: null,
      capabilities: {
        ephemeral_chat: {
          state: 'available',
          reason: null,
        },
        harness_ai: {
          state: 'available',
          reason: null,
        },
        skill_ai: {
          state: 'available',
          reason: null,
        },
      },
      cli_command: 'codex',
      cli_args: ['app-server', '--listen', 'stdio://'],
      auth_config: {},
      model_name: 'gpt-5.4',
      model_temperature: 0,
      model_max_tokens: 16384,
      max_parallel_runs: 0,
      cost_per_input_token: 0,
      cost_per_output_token: 0,
      pricing_config: {},
    },
  ]

  const agents = [
    {
      id: DEFAULT_AGENT_ID,
      project_id: PROJECT_ID,
      provider_id: DEFAULT_PROVIDER_ID,
      name: 'coding-main',
      runtime_control_state: 'active',
      total_tickets_completed: 12,
      runtime: {
        status: 'ready',
        runtime_phase: 'ready',
        active_run_count: 1,
        current_run_id: 'run-1',
        current_ticket_id: DEFAULT_TICKET_ID,
        last_heartbeat_at: nowIso,
        runtime_started_at: '2026-03-27T09:00:00.000Z',
        session_id: 'sess-1',
        last_error: '',
      },
    },
  ]

  const agentRuns = [
    {
      id: 'run-1',
      project_id: PROJECT_ID,
      agent_id: DEFAULT_AGENT_ID,
      provider_id: DEFAULT_PROVIDER_ID,
      workflow_id: DEFAULT_WORKFLOW_ID,
      ticket_id: DEFAULT_TICKET_ID,
      status: 'executing',
      last_heartbeat_at: nowIso,
      runtime_started_at: '2026-03-27T09:00:00.000Z',
      session_id: 'sess-1',
      last_error: '',
      created_at: '2026-03-27T09:00:00.000Z',
    },
  ]

  const activityEvents = [
    {
      id: 'activity-1',
      project_id: PROJECT_ID,
      ticket_id: DEFAULT_TICKET_ID,
      agent_id: DEFAULT_AGENT_ID,
      event_type: 'agent_started',
      message: 'coding-main started work.',
      metadata: {
        agent_name: 'coding-main',
      },
      created_at: nowIso,
    },
  ]

  const tickets = [
    createMockTicketRecord({
      id: DEFAULT_TICKET_ID,
      identifier: 'ASE-101',
      title: 'Improve machine management UX',
      statusId: DEFAULT_STATUS_IDS.todo,
      statusName: 'Todo',
      workflowId: DEFAULT_WORKFLOW_ID,
    }),
  ]

  const statuses = [
    {
      id: DEFAULT_STATUS_IDS.todo,
      project_id: PROJECT_ID,
      name: 'Todo',
      stage: 'unstarted',
      color: '#2563eb',
      icon: '',
      position: 1,
      active_runs: 1,
      max_active_runs: null,
    },
    {
      id: DEFAULT_STATUS_IDS.review,
      project_id: PROJECT_ID,
      name: 'In Review',
      stage: 'started',
      color: '#f59e0b',
      icon: '',
      position: 2,
      active_runs: 0,
      max_active_runs: null,
    },
    {
      id: DEFAULT_STATUS_IDS.done,
      project_id: PROJECT_ID,
      name: 'Done',
      stage: 'completed',
      color: '#16a34a',
      icon: '',
      position: 3,
      active_runs: 0,
      max_active_runs: null,
    },
  ]

  const repos = [
    {
      id: DEFAULT_REPO_ID,
      project_id: PROJECT_ID,
      name: 'TodoApp',
      repository_url: 'https://github.com/BetterAndBetterII/TodoApp.git',
      default_branch: 'main',
      workspace_dirname: 'TodoApp',
      labels: [],
    },
  ]

  const workflows = [
    {
      id: DEFAULT_WORKFLOW_ID,
      project_id: PROJECT_ID,
      name: 'Coding Workflow',
      type: 'coding',
      agent_id: DEFAULT_AGENT_ID,
      pickup_status_ids: [DEFAULT_STATUS_IDS.todo],
      finish_status_ids: [DEFAULT_STATUS_IDS.review],
      max_concurrent: 0,
      max_retry_attempts: 1,
      timeout_minutes: 30,
      stall_timeout_minutes: 5,
      is_active: true,
      harness_path: '.openase/harnesses/coding-workflow.md',
      version: 3,
    },
  ]

  const harnessByWorkflowId = {
    [DEFAULT_WORKFLOW_ID]: {
      content: defaultHarnessContent('Coding Workflow'),
      path: '.openase/harnesses/coding-workflow.md',
      version: 3,
      history: [
        {
          id: `${DEFAULT_WORKFLOW_ID}-v3`,
          version: 3,
          created_by: 'user:manual',
          created_at: nowIso,
        },
        {
          id: `${DEFAULT_WORKFLOW_ID}-v2`,
          version: 2,
          created_by: 'user:manual',
          created_at: '2026-03-26T10:00:00.000Z',
        },
      ],
    },
  }

  const scheduledJobs = [
    {
      id: DEFAULT_JOB_ID,
      project_id: PROJECT_ID,
      workflow_id: DEFAULT_WORKFLOW_ID,
      name: 'Nightly regression sweep',
      cron_expression: '0 2 * * *',
      is_enabled: true,
      ticket_template: {
        title: 'Run nightly regression sweep',
        description: 'Verify the core project workflows still behave as expected.',
        priority: 'medium',
        type: 'feature',
        budget_usd: 12,
        created_by: 'scheduler',
      },
      last_run_at: '2026-03-26T02:00:00.000Z',
      next_run_at: '2026-03-28T02:00:00.000Z',
    },
  ]

  const skills = [
    {
      id: 'skill-commit',
      project_id: PROJECT_ID,
      name: 'commit',
      description: 'Create a well-formed git commit.',
      path: '/skills/commit',
      current_version: 2,
      is_builtin: true,
      is_enabled: true,
      created_by: 'system:init',
      created_at: nowIso,
      bound_workflows: [{ id: DEFAULT_WORKFLOW_ID }],
      content: '# Commit\n\nCreate a well-formed git commit.',
      history: [
        { id: 'skill-commit-v2', version: 2, created_by: 'user:manual', created_at: nowIso },
        {
          id: 'skill-commit-v1',
          version: 1,
          created_by: 'system:init',
          created_at: '2026-03-26T10:00:00.000Z',
        },
      ],
    },
    {
      id: 'skill-deploy-openase',
      project_id: PROJECT_ID,
      name: 'deploy-openase',
      description: 'Build and redeploy OpenASE locally.',
      path: '/skills/deploy-openase',
      current_version: 1,
      is_builtin: false,
      is_enabled: true,
      created_by: 'user:manual',
      created_at: nowIso,
      bound_workflows: [],
      content: '# Deploy OpenASE\n\nBuild and redeploy OpenASE locally.',
      history: [
        {
          id: 'skill-deploy-openase-v1',
          version: 1,
          created_by: 'user:manual',
          created_at: nowIso,
        },
      ],
    },
  ]

  const builtinRoles = [
    {
      id: 'builtin-coding',
      workflow_type: 'coding',
      name: 'coding',
      content: defaultHarnessContent('Coding Workflow'),
    },
  ]

  const harnessVariables = {
    groups: [
      {
        name: 'ticket',
        variables: [
          { name: 'ticket.identifier', description: 'Ticket identifier' },
          { name: 'ticket.title', description: 'Ticket title' },
        ],
      },
      {
        name: 'project',
        variables: [{ name: 'project.name', description: 'Project name' }],
      },
    ],
  }

  return {
    organizations,
    projects,
    machines,
    providers,
    agents,
    agentRuns,
    activityEvents,
    tickets,
    statuses,
    repos,
    workflows,
    harnessByWorkflowId,
    scheduledJobs,
    projectConversations: [],
    projectConversationEntries: [],
    skills,
    builtinRoles,
    securitySettingsByProjectId: {
      [PROJECT_ID]: createDefaultSecuritySettings(PROJECT_ID),
    },
    harnessVariables,
    counters: {
      machine: 2,
      repo: 1,
      workflow: 1,
      agent: 1,
      skill: 2,
      scheduledJob: 1,
      projectConversation: 0,
      projectConversationEntry: 0,
      projectConversationTurn: 0,
    },
  }
}

function seedBoardState(countsByStatusID: Record<string, number>) {
  const statusNameByID = new Map(
    mockState.statuses.map((status) => [
      asString(status.id) ?? '',
      asString(status.name) ?? 'Todo',
    ]),
  )
  const seededTickets: Record<string, unknown>[] = []
  let sequence = 0

  for (const status of mockState.statuses) {
    const statusId = asString(status.id)
    if (!statusId) {
      continue
    }
    const count = Math.max(0, asNumber(countsByStatusID[statusId]) ?? 0)
    for (let index = 0; index < count; index += 1) {
      sequence += 1
      seededTickets.push(
        createMockTicketRecord({
          id: `ticket-seeded-${sequence}`,
          identifier: `ASE-${100 + sequence}`,
          title: `Seeded ticket ${sequence}`,
          statusId,
          statusName: statusNameByID.get(statusId) ?? 'Todo',
          workflowId: DEFAULT_WORKFLOW_ID,
          createdAt: new Date(Date.parse(nowIso) + sequence * 60_000).toISOString(),
        }),
      )
    }
  }

  mockState.tickets = seededTickets
  mockState.activityEvents = []
}

function resolveSecuritySettings(projectId: string): MockSecuritySettings {
  const existing = mockState.securitySettingsByProjectId[projectId]
  if (existing) {
    return existing
  }
  const created = createDefaultSecuritySettings(projectId)
  mockState.securitySettingsByProjectId[projectId] = created
  return created
}

function createDefaultSecuritySettings(projectId: string): MockSecuritySettings {
  const organization = createEmptyGitHubSlot('organization')
  const projectOverride = createEmptyGitHubSlot('project')
  return {
    project_id: projectId,
    agent_tokens: {
      transport: 'Bearer token',
      environment_variable: 'OPENASE_AGENT_TOKEN',
      token_prefix: 'ase_agent_',
      default_scopes: ['tickets.create', 'tickets.list'],
      supported_project_scopes: ['projects.update', 'projects.add_repo'],
    },
    github: {
      effective: clone(organization),
      organization,
      project_override: projectOverride,
    },
    webhooks: {
      connector_endpoint: 'POST /api/v1/webhooks/:connector/:provider',
    },
    secret_hygiene: {
      notification_channel_configs_redacted: true,
    },
    approval_policies: {
      status: 'reserved',
      rules_count: 0,
      summary:
        'Approval policy storage is reserved for future second-factor or approver requirements and stays separate from RBAC grants.',
    },
    deferred: [
      {
        key: 'github-device-flow',
        title: 'GitHub Device Flow',
        summary: 'Deferred until OAuth app wiring is available.',
      },
    ],
  }
}

function createEmptyGitHubSlot(scope: MockGitHubCredentialScope): MockGitHubSlot {
  return {
    scope,
    configured: false,
    source: '',
    token_preview: '',
    probe: createUnconfiguredGitHubProbe(),
  }
}

function createConfiguredGitHubProbe(login: string): MockGitHubProbe {
  return {
    state: 'valid',
    configured: true,
    valid: true,
    login,
    permissions: ['repo', 'read:org'],
    repo_access: 'granted',
    checked_at: nowIso,
    last_error: '',
  }
}

function createUnconfiguredGitHubProbe(): MockGitHubProbe {
  return {
    state: 'missing',
    configured: false,
    valid: false,
    permissions: [],
    repo_access: 'not_checked',
    checked_at: undefined,
    last_error: '',
  }
}

function resolveGitHubCredentialSlot(
  security: MockSecuritySettings,
  scope: MockGitHubCredentialScope,
): MockGitHubSlot {
  return scope === 'organization' ? security.github.organization : security.github.project_override
}

function syncEffectiveGitHubSlot(security: MockSecuritySettings) {
  security.github.effective = clone(
    security.github.project_override.configured
      ? security.github.project_override
      : security.github.organization,
  )
}

function parseGitHubCredentialScope(
  raw: string | null | undefined,
): MockGitHubCredentialScope | null {
  return raw === 'organization' || raw === 'project' ? raw : null
}

function previewToken(token: string): string {
  const trimmed = token.trim()
  if (trimmed.length <= 8) {
    return trimmed
  }
  return `${trimmed.slice(0, 8)}...${trimmed.slice(-4)}`
}

function createMockTicketRecord(input: {
  id: string
  identifier: string
  title: string
  description?: string
  statusId: string
  statusName: string
  workflowId: string
  createdAt?: string
}) {
  return {
    id: input.id,
    project_id: PROJECT_ID,
    identifier: input.identifier,
    title: input.title,
    description: input.description ?? '',
    status_id: input.statusId,
    status_name: input.statusName,
    priority: 'medium',
    type: 'feature',
    workflow_id: input.workflowId,
    current_run_id: null,
    target_machine_id: null,
    created_by: 'playwright',
    parent: null,
    children: [],
    dependencies: [],
    external_links: [],
    external_ref: '',
    budget_usd: 0,
    cost_tokens_input: 0,
    cost_tokens_output: 0,
    cost_tokens_total: 0,
    cost_amount: 0,
    attempt_count: 0,
    consecutive_errors: 0,
    started_at: null,
    completed_at: null,
    next_retry_at: null,
    retry_paused: false,
    pause_reason: '',
    created_at: input.createdAt ?? nowIso,
  }
}

function buildTicketDetailPayload(ticketId: string) {
  const ticket = findById(mockState.tickets, ticketId)
  if (!ticket) {
    return null
  }

  const repo = mockState.repos.find((item) => item.id === DEFAULT_REPO_ID)
  const detailDescription =
    ticketId === DEFAULT_TICKET_ID
      ? 'Project AI needs the full ticket capsule so the drawer no longer depends on a separate ticket-detail assistant.'
      : (asString(ticket.description) ?? '')

  const ticketRecord = {
    ...clone(ticket),
    description: detailDescription,
    priority: ticketId === DEFAULT_TICKET_ID ? 'high' : (asString(ticket.priority) ?? 'medium'),
    current_run_id: ticketId === DEFAULT_TICKET_ID ? 'run-1' : null,
    attempt_count: ticketId === DEFAULT_TICKET_ID ? 3 : (asNumber(ticket.attempt_count) ?? 0),
    consecutive_errors:
      ticketId === DEFAULT_TICKET_ID ? 2 : (asNumber(ticket.consecutive_errors) ?? 0),
    retry_paused: ticketId === DEFAULT_TICKET_ID ? true : (asBoolean(ticket.retry_paused) ?? false),
    pause_reason:
      ticketId === DEFAULT_TICKET_ID
        ? 'Repeated hook failures'
        : (asString(ticket.pause_reason) ?? ''),
    next_retry_at: ticketId === DEFAULT_TICKET_ID ? null : (asString(ticket.next_retry_at) ?? null),
    dependencies:
      ticketId === DEFAULT_TICKET_ID
        ? [
            {
              id: 'dependency-1',
              type: 'blocked_by',
              target: {
                id: 'ticket-blocker',
                identifier: 'ASE-77',
                title: 'Stabilize project conversation restore',
                status_id: DEFAULT_STATUS_IDS.review,
                status_name: 'In Review',
              },
            },
          ]
        : [],
    children: [],
  }

  const pickupDiagnosis = ticketRecord.retry_paused
    ? {
        state: 'blocked',
        primary_reason_code: 'retry_paused_repeated_stalls',
        primary_reason_message: 'Retries are paused after repeated stalls.',
        next_action_hint: 'Review the last failed attempt, then continue retry when ready.',
        reasons: [
          {
            code: 'retry_paused_repeated_stalls',
            message: 'Manual retry is required after repeated stalls.',
            severity: 'warning',
          },
        ],
        workflow: {
          id: DEFAULT_WORKFLOW_ID,
          name: 'Coding Workflow',
          is_active: true,
          pickup_status_match: true,
        },
        agent: {
          id: DEFAULT_AGENT_ID,
          name: 'coding-main',
          runtime_control_state: 'active',
        },
        provider: {
          id: DEFAULT_PROVIDER_ID,
          name: 'codex-app-server',
          machine_id: LOCAL_MACHINE_ID,
          machine_name: 'local-dev',
          machine_status: 'online',
          availability_state: 'available',
          availability_reason: null,
        },
        retry: {
          attempt_count: ticketRecord.attempt_count,
          retry_paused: ticketRecord.retry_paused,
          pause_reason: 'repeated_stalls',
          next_retry_at: null,
        },
        capacity: {
          workflow: { limited: false, active_runs: 0, capacity: 0 },
          project: { limited: false, active_runs: 0, capacity: 0 },
          provider: { limited: false, active_runs: 0, capacity: 0 },
          status: { limited: false, active_runs: 0, capacity: null },
        },
        blocked_by: ticketRecord.dependencies
          .filter((item) => item.type === 'blocked_by')
          .map((item) => ({
            id: item.target.id,
            identifier: item.target.identifier,
            title: item.target.title,
            status_id: item.target.status_id,
            status_name: item.target.status_name,
          })),
      }
    : ticketRecord.current_run_id
      ? {
          state: 'running',
          primary_reason_code: 'running_current_run',
          primary_reason_message: 'Ticket already has an active run.',
          next_action_hint: 'Wait for the current run to finish or inspect the active runtime.',
          reasons: [
            {
              code: 'running_current_run',
              message: 'Current run is still attached to the ticket.',
              severity: 'info',
            },
          ],
          workflow: {
            id: DEFAULT_WORKFLOW_ID,
            name: 'Coding Workflow',
            is_active: true,
            pickup_status_match: true,
          },
          agent: {
            id: DEFAULT_AGENT_ID,
            name: 'coding-main',
            runtime_control_state: 'active',
          },
          provider: {
            id: DEFAULT_PROVIDER_ID,
            name: 'codex-app-server',
            machine_id: LOCAL_MACHINE_ID,
            machine_name: 'local-dev',
            machine_status: 'online',
            availability_state: 'available',
            availability_reason: null,
          },
          retry: {
            attempt_count: ticketRecord.attempt_count,
            retry_paused: false,
            pause_reason: '',
            next_retry_at: ticketRecord.next_retry_at,
          },
          capacity: {
            workflow: { limited: false, active_runs: 1, capacity: 0 },
            project: { limited: false, active_runs: 1, capacity: 0 },
            provider: { limited: false, active_runs: 1, capacity: 0 },
            status: { limited: false, active_runs: 1, capacity: null },
          },
          blocked_by: [],
        }
      : {
          state: 'runnable',
          primary_reason_code: 'ready_for_pickup',
          primary_reason_message: 'Ticket is ready for pickup.',
          next_action_hint: 'Wait for the scheduler to claim the ticket.',
          reasons: [
            {
              code: 'ready_for_pickup',
              message: 'The scheduler can claim this ticket on the next tick.',
              severity: 'info',
            },
          ],
          workflow: {
            id: DEFAULT_WORKFLOW_ID,
            name: 'Coding Workflow',
            is_active: true,
            pickup_status_match: true,
          },
          agent: {
            id: DEFAULT_AGENT_ID,
            name: 'coding-main',
            runtime_control_state: 'active',
          },
          provider: {
            id: DEFAULT_PROVIDER_ID,
            name: 'codex-app-server',
            machine_id: LOCAL_MACHINE_ID,
            machine_name: 'local-dev',
            machine_status: 'online',
            availability_state: 'available',
            availability_reason: null,
          },
          retry: {
            attempt_count: ticketRecord.attempt_count,
            retry_paused: false,
            pause_reason: '',
            next_retry_at: ticketRecord.next_retry_at,
          },
          capacity: {
            workflow: { limited: false, active_runs: 0, capacity: 0 },
            project: { limited: false, active_runs: 0, capacity: 0 },
            provider: { limited: false, active_runs: 0, capacity: 0 },
            status: { limited: false, active_runs: 0, capacity: null },
          },
          blocked_by: [],
        }

  const repoScopes =
    ticketId === DEFAULT_TICKET_ID && repo
      ? [
          {
            id: 'repo-scope-1',
            ticket_id: ticketId,
            repo_id: DEFAULT_REPO_ID,
            repo,
            branch_name: 'feat/openase-470-project-ai',
            pull_request_url: 'https://github.com/PacificStudio/openase/pull/999',
            created_at: nowIso,
          },
        ]
      : []

  const timeline = [
    {
      id: `description:${ticketId}`,
      ticket_id: ticketId,
      item_type: 'description',
      actor_name: 'playwright',
      actor_type: 'user',
      title: asString(ticket.title),
      body_markdown: detailDescription,
      body_text: null,
      created_at: asString(ticket.created_at) ?? nowIso,
      updated_at: asString(ticket.created_at) ?? nowIso,
      edited_at: null,
      is_collapsible: false,
      is_deleted: false,
      metadata: {
        identifier: asString(ticket.identifier),
      },
    },
    {
      id: `activity:${ticketId}:retry-paused`,
      ticket_id: ticketId,
      item_type: 'activity',
      actor_name: 'orchestrator',
      actor_type: 'system',
      title: 'ticket.retry_paused',
      body_markdown: null,
      body_text: 'Paused retries after repeated hook failures.',
      created_at: '2026-04-02T08:10:00.000Z',
      updated_at: '2026-04-02T08:10:00.000Z',
      edited_at: null,
      is_collapsible: true,
      is_deleted: false,
      metadata: {
        event_type: 'ticket.retry_paused',
      },
    },
  ]

  return {
    assigned_agent:
      ticketId === DEFAULT_TICKET_ID
        ? {
            id: DEFAULT_AGENT_ID,
            name: 'coding-main',
            provider: 'codex-app-server',
            runtime_control_state: 'active',
            runtime_phase: 'executing',
          }
        : null,
    pickup_diagnosis: pickupDiagnosis,
    ticket: ticketRecord,
    repo_scopes: repoScopes,
    comments: [],
    timeline,
    activity: [],
    hook_history:
      ticketId === DEFAULT_TICKET_ID
        ? [
            {
              id: 'hook-history-1',
              ticket_id: ticketId,
              event_type: 'ticket.on_complete',
              message: 'go test ./... failed in internal/chat',
              metadata: {},
              created_at: '2026-04-02T08:15:00.000Z',
            },
          ]
        : [],
  }
}

function buildTicketRunsPayload(ticketId: string) {
  if (ticketId !== DEFAULT_TICKET_ID) {
    return { runs: [] }
  }
  return {
    runs: [
      {
        id: 'run-1',
        ticket_id: ticketId,
        attempt_number: 3,
        agent_id: DEFAULT_AGENT_ID,
        agent_name: 'coding-main',
        provider: 'codex-app-server',
        adapter_type: 'codex-app-server',
        model_name: 'gpt-5.4',
        usage: {
          total: 1540,
          input: 1200,
          output: 340,
          cached_input: 120,
          cache_creation: 45,
          reasoning: 80,
          prompt: 920,
          candidate: 260,
          tool: 30,
        },
        status: 'failed',
        current_step_status: 'failed',
        current_step_summary: 'openase test ./internal/chat',
        created_at: '2026-04-02T08:00:00.000Z',
        runtime_started_at: '2026-04-02T08:01:00.000Z',
        last_heartbeat_at: '2026-04-02T08:14:00.000Z',
        completed_at: '2026-04-02T08:15:00.000Z',
        last_error: 'ticket.on_complete hook failed',
      },
    ],
  }
}

function buildTicketRunDetailPayload(ticketId: string, runId: string) {
  if (ticketId !== DEFAULT_TICKET_ID || runId !== 'run-1') {
    return null
  }
  return {
    run: buildTicketRunsPayload(ticketId).runs[0],
    trace_entries: [],
    step_entries: [
      {
        id: 'step-1',
        agent_run_id: 'run-1',
        step_status: 'failed',
        summary: 'openase test ./internal/chat',
        source_trace_event_id: null,
        created_at: '2026-04-02T08:15:00.000Z',
      },
    ],
  }
}

function buildMockProjectConversationReply(
  message: string,
  ticketFocus: {
    identifier: string
    status: string
    retryPaused: boolean
    pauseReason: string
    repoScopes: unknown[]
    hookHistory: unknown[]
    currentRun: Record<string, unknown> | null
  } | null,
) {
  if (!ticketFocus) {
    return `Mock assistant reply for: ${message}`
  }

  const normalized = message.toLowerCase()
  if (normalized.includes('why is this ticket not running')) {
    const currentRunStatus = asString(ticketFocus.currentRun?.status) ?? 'unknown'
    const lastError = asString(ticketFocus.currentRun?.last_error) ?? 'unknown'
    return `${ticketFocus.identifier} is currently ${ticketFocus.status}. Retries are paused=${ticketFocus.retryPaused} because "${ticketFocus.pauseReason}". The latest run status was ${currentRunStatus} and the latest failure was "${lastError}".`
  }

  if (normalized.includes('which repos does this ticket currently affect')) {
    const scopes = ticketFocus.repoScopes
      .map((scope) => asObject(scope))
      .filter((scope): scope is Record<string, unknown> => scope !== null)
      .map((scope) =>
        [
          asString(scope.repo_name) ?? asString(scope.repo_id) ?? 'unknown-repo',
          asString(scope.branch_name),
        ]
          .filter(Boolean)
          .join(' @ '),
      )
      .filter(Boolean)
    return `${ticketFocus.identifier} currently affects ${scopes.join(', ')}.`
  }

  if (normalized.includes('what hook failed most recently')) {
    const latestHook = ticketFocus.hookHistory
      .map((hook) => asObject(hook))
      .filter((hook): hook is Record<string, unknown> => hook !== null)
      .at(-1)
    return latestHook
      ? `The latest hook was ${asString(latestHook.hook_name) ?? 'unknown'} and it reported "${asString(latestHook.output) ?? ''}".`
      : `No hook history is available for ${ticketFocus.identifier}.`
  }

  return `Mock assistant reply for ${ticketFocus.identifier}: ${message}`
}

function buildMockProjectConversationWorkspaceDiff(conversationId: string) {
  return {
    conversation_id: conversationId,
    workspace_path: `/tmp/${conversationId}`,
    dirty: false,
    repos_changed: 0,
    files_changed: 0,
    added: 0,
    removed: 0,
    repos: [],
  }
}

function createMachineRecord(body: Record<string, unknown>) {
  const id = `machine-${++mockState.counters.machine}`
  const host = asString(body.host) ?? '127.0.0.1'
  const reachabilityMode = resolveMachineReachabilityMode(body, host)
  const executionMode = resolveMachineExecutionMode(body, host)
  const connectionMode = resolveMachineConnectionMode(reachabilityMode, executionMode)
  const executionCapabilities = executionCapabilitiesForConnectionMode(connectionMode)
  const sshUser = asString(body.ssh_user) ?? ''
  const sshKeyPath = asString(body.ssh_key_path) ?? ''
  return {
    id,
    org_id: ORG_ID,
    name: asString(body.name) ?? `machine-${mockState.counters.machine}`,
    host,
    port: asNumber(body.port) ?? 22,
    reachability_mode: reachabilityMode,
    execution_mode: executionMode,
    execution_capabilities: executionCapabilities,
    ssh_helper_enabled: Boolean(sshUser || sshKeyPath),
    ssh_helper_required: executionMode === 'ssh_compat',
    connection_mode: connectionMode,
    transport_capabilities: executionCapabilities,
    ssh_user: sshUser,
    ssh_key_path: sshKeyPath,
    advertised_endpoint: asString(body.advertised_endpoint),
    daemon_status: {
      registered: false,
      last_registered_at: null,
      current_session_id: null,
      session_state: 'unknown',
    },
    detected_os: 'unknown',
    detected_arch: 'unknown',
    detection_status: 'unknown',
    detection_message:
      executionMode === 'ssh_compat'
        ? 'This machine still uses SSH compatibility until it is migrated to websocket execution.'
        : 'Operating system and architecture are still unknown. You can keep configuring the machine and verify the platform manually.',
    channel_credential: {
      kind: 'none',
      token_id: null,
      certificate_id: null,
    },
    description: asString(body.description) ?? '',
    labels: asStringArray(body.labels),
    status: asString(body.status) ?? 'maintenance',
    workspace_root: asString(body.workspace_root) ?? '',
    agent_cli_path: asString(body.agent_cli_path) ?? '',
    env_vars: asStringArray(body.env_vars),
    last_heartbeat_at: null,
    resources: {},
  }
}

function createRepoRecord(body: Record<string, unknown>) {
  return {
    id: `repo-${++mockState.counters.repo}`,
    project_id: PROJECT_ID,
    name: asString(body.name) ?? `repo-${mockState.counters.repo}`,
    repository_url: asString(body.repository_url) ?? '',
    default_branch: asString(body.default_branch) ?? 'main',
    workspace_dirname:
      asString(body.workspace_dirname) ?? asString(body.name) ?? `repo-${mockState.counters.repo}`,
    labels: asStringArray(body.labels),
  }
}

function createScheduledJobRecord(body: Record<string, unknown>) {
  const ticketTemplate = asObject(body.ticket_template) ?? {}
  return {
    id: `job-${++mockState.counters.scheduledJob}`,
    project_id: PROJECT_ID,
    workflow_id: asString(body.workflow_id) ?? DEFAULT_WORKFLOW_ID,
    name: asString(body.name) ?? `Job ${mockState.counters.scheduledJob}`,
    cron_expression: asString(body.cron_expression) ?? '0 0 * * *',
    is_enabled: asBoolean(body.is_enabled) ?? true,
    ticket_template: {
      title: asString(ticketTemplate.title) ?? null,
      description: asString(ticketTemplate.description) ?? null,
      priority: asString(ticketTemplate.priority) ?? 'medium',
      type: asString(ticketTemplate.type) ?? 'feature',
      budget_usd: asNumber(ticketTemplate.budget_usd) ?? 0,
      created_by: asString(ticketTemplate.created_by) ?? null,
    },
    last_run_at: null,
    next_run_at: '2026-03-28T02:00:00.000Z',
  }
}

function applyMachineMutation(machine: Record<string, unknown>, body: Record<string, unknown>) {
  machine.name = asString(body.name) ?? machine.name
  machine.host = asString(body.host) ?? machine.host
  machine.port = asNumber(body.port) ?? machine.port
  machine.reachability_mode = resolveMachineReachabilityMode(body, asString(machine.host) ?? '')
  machine.execution_mode = resolveMachineExecutionMode(body, asString(machine.host) ?? '')
  machine.connection_mode = resolveMachineConnectionMode(
    asString(machine.reachability_mode) ?? 'direct_connect',
    asString(machine.execution_mode) ?? 'websocket',
  )
  machine.execution_capabilities = executionCapabilitiesForConnectionMode(
    asString(machine.connection_mode) ?? 'ssh',
  )
  machine.transport_capabilities = clone(machine.execution_capabilities)
  machine.ssh_user = asString(body.ssh_user) ?? machine.ssh_user
  machine.ssh_key_path = asString(body.ssh_key_path) ?? machine.ssh_key_path
  machine.ssh_helper_enabled = Boolean(
    (asString(machine.ssh_user) ?? '').trim() || (asString(machine.ssh_key_path) ?? '').trim(),
  )
  machine.ssh_helper_required = asString(machine.execution_mode) === 'ssh_compat'
  machine.advertised_endpoint = asString(body.advertised_endpoint) ?? machine.advertised_endpoint
  machine.description = asString(body.description) ?? machine.description
  machine.labels = asStringArray(body.labels)
  machine.status = asString(body.status) ?? machine.status
  machine.workspace_root = asString(body.workspace_root) ?? machine.workspace_root
  machine.agent_cli_path = asString(body.agent_cli_path) ?? machine.agent_cli_path
  machine.env_vars = asStringArray(body.env_vars)
}

function resolveMachineReachabilityMode(body: Record<string, unknown>, host: string) {
  const raw = asString(body.reachability_mode)
  if (raw === 'local' || raw === 'direct_connect' || raw === 'reverse_connect') {
    return raw
  }
  const legacy = asString(body.connection_mode)
  switch (legacy) {
    case 'local':
      return 'local'
    case 'ws_reverse':
      return 'reverse_connect'
    case 'ssh':
    case 'ws_listener':
      return 'direct_connect'
    default:
      return host.trim().toLowerCase() === 'local' ? 'local' : 'direct_connect'
  }
}

function resolveMachineExecutionMode(body: Record<string, unknown>, host: string) {
  const raw = asString(body.execution_mode)
  if (raw === 'local_process' || raw === 'websocket' || raw === 'ssh_compat') {
    return raw
  }
  const legacy = asString(body.connection_mode)
  switch (legacy) {
    case 'local':
      return 'local_process'
    case 'ssh':
      return 'ssh_compat'
    case 'ws_reverse':
    case 'ws_listener':
      return 'websocket'
    default:
      return host.trim().toLowerCase() === 'local' ? 'local_process' : 'websocket'
  }
}

function resolveMachineConnectionMode(reachabilityMode: string, executionMode: string) {
  if (reachabilityMode === 'local') {
    return 'local'
  }
  if (reachabilityMode === 'reverse_connect') {
    return 'ws_reverse'
  }
  return executionMode === 'ssh_compat' ? 'ssh' : 'ws_listener'
}

function executionCapabilitiesForConnectionMode(connectionMode: string) {
  if (connectionMode === 'ws_reverse') {
    return []
  }
  return ['probe', 'workspace_prepare', 'artifact_sync', 'process_streaming']
}

function applyRepoMutation(repo: Record<string, unknown>, body: Record<string, unknown>) {
  repo.name = asString(body.name) ?? repo.name
  repo.repository_url = asString(body.repository_url) ?? repo.repository_url
  repo.default_branch = asString(body.default_branch) ?? repo.default_branch
  repo.workspace_dirname = asString(body.workspace_dirname) ?? repo.workspace_dirname
  repo.labels = asStringArray(body.labels)
}

function applyScheduledJobMutation(job: Record<string, unknown>, body: Record<string, unknown>) {
  job.name = asString(body.name) ?? job.name
  job.cron_expression = asString(body.cron_expression) ?? job.cron_expression
  job.workflow_id = asString(body.workflow_id) ?? job.workflow_id
  if (typeof body.is_enabled === 'boolean') {
    job.is_enabled = body.is_enabled
  }
  const ticketTemplate = {
    ...(asObject(job.ticket_template) ?? {}),
    ...(asObject(body.ticket_template) ?? {}),
  }
  job.ticket_template = {
    title: asString(ticketTemplate.title) ?? null,
    description: asString(ticketTemplate.description) ?? null,
    priority: asString(ticketTemplate.priority) ?? 'medium',
    type: asString(ticketTemplate.type) ?? 'feature',
    budget_usd: asNumber(ticketTemplate.budget_usd) ?? 0,
    created_by: asString(ticketTemplate.created_by) ?? null,
  }
}

function machineResourceSnapshot(
  transport: string,
  values: {
    cpuUsagePercent: number
    memoryTotalGB: number
    memoryUsedGB: number
    memoryAvailableGB: number
    diskTotalGB: number
    diskAvailableGB: number
    gpuDispatchable: boolean
    gpus: Record<string, unknown>[]
  },
) {
  return {
    transport,
    checked_at: nowIso,
    last_success: true,
    cpu_cores: 16,
    cpu_usage_percent: values.cpuUsagePercent,
    memory_total_gb: values.memoryTotalGB,
    memory_used_gb: values.memoryUsedGB,
    memory_available_gb: values.memoryAvailableGB,
    disk_total_gb: values.diskTotalGB,
    disk_available_gb: values.diskAvailableGB,
    gpu_dispatchable: values.gpuDispatchable,
    gpu: values.gpus,
    monitor: {
      l1: {
        checked_at: nowIso,
        reachable: true,
        latency_ms: transport === 'local' ? 1 : 18,
        transport,
      },
      l2: {
        checked_at: nowIso,
        available: true,
        disk_low: false,
        memory_low: false,
      },
      l3: {
        checked_at: nowIso,
        gpu_dispatchable: values.gpuDispatchable,
      },
    },
  }
}

function defaultHarnessContent(name: string) {
  return `---\nname: ${name}\n---\n\n# ${name}\n\n- Inspect the ticket context.\n- Make the requested change.\n- Summarize the result.`
}

function dedupeById(items: Record<string, unknown>[]) {
  const seen = new Set<string>()
  return items.filter((item) => {
    const id = asString(item.id)
    if (!id || seen.has(id)) {
      return false
    }
    seen.add(id)
    return true
  })
}

function slugify(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

function clone<T>(value: T): T {
  return structuredClone(value)
}

function findById(
  items: Record<string, unknown>[],
  id: string,
): Record<string, unknown> | undefined {
  return items.find((item) => item.id === id)
}

function asString(value: JsonValue | undefined): string | undefined {
  return typeof value === 'string' ? value : undefined
}

function asNumber(value: JsonValue | undefined): number | undefined {
  return typeof value === 'number' ? value : undefined
}

function asBoolean(value: JsonValue | undefined): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined
}

function asObject(value: JsonValue | undefined): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null
}

function asStringArray(value: JsonValue | undefined): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : []
}

function asObjectArray(value: JsonValue | undefined): Record<string, unknown>[] | null {
  return Array.isArray(value)
    ? value.filter(
        (item): item is Record<string, unknown> =>
          !!item && typeof item === 'object' && !Array.isArray(item),
      )
    : null
}

function decodeBase64UTF8(value: string): string {
  try {
    return Buffer.from(value, 'base64').toString('utf8')
  } catch {
    return ''
  }
}

async function readBody<T>(request: Request): Promise<T> {
  const text = await request.text()
  if (!text) {
    return {} as T
  }
  return JSON.parse(text) as T
}

function jsonResponse(body: JsonValue | Record<string, unknown>, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'content-type': 'application/json',
      'cache-control': 'no-store',
    },
  })
}

function noContentResponse() {
  return new Response(null, {
    status: 204,
    headers: {
      'cache-control': 'no-store',
    },
  })
}

function notFound(detail: string) {
  return jsonResponse({ detail }, 404)
}

function streamResponse() {
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      controller.enqueue(encoder.encode(': openase-e2e\n\n'))
    },
    cancel() {},
  })

  return new Response(stream, {
    headers: {
      'content-type': 'text/event-stream',
      'cache-control': 'no-store',
      connection: 'keep-alive',
    },
  })
}

function projectConversationStreamResponse(conversationId: string) {
  let sink: ReadableStreamDefaultController<Uint8Array> | null = null
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      sink = controller
      controller.enqueue(encoder.encode(': openase-e2e\n\n'))
      let sinks = projectConversationStreamControllers.get(conversationId)
      if (!sinks) {
        sinks = new Set()
        projectConversationStreamControllers.set(conversationId, sinks)
      }
      sinks.add(controller)

      const queuedFrames = queuedProjectConversationFrames.get(conversationId) ?? []
      for (const frame of queuedFrames) {
        controller.enqueue(encoder.encode(frame))
      }
      queuedProjectConversationFrames.delete(conversationId)
    },
    cancel() {
      if (!sink) {
        return
      }
      const sinks = projectConversationStreamControllers.get(conversationId)
      if (!sinks) {
        return
      }
      sinks.delete(sink)
      if (sinks.size === 0) {
        projectConversationStreamControllers.delete(conversationId)
      }
    },
  })

  return new Response(stream, {
    headers: {
      'content-type': 'text/event-stream',
      'cache-control': 'no-store',
      connection: 'keep-alive',
    },
  })
}

function projectConversationMuxStreamResponse(projectId: string) {
  let sink: ReadableStreamDefaultController<Uint8Array> | null = null
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      sink = controller
      controller.enqueue(encoder.encode(': openase-e2e\n\n'))
      let sinks = projectConversationMuxStreamControllers.get(projectId)
      if (!sinks) {
        sinks = new Set()
        projectConversationMuxStreamControllers.set(projectId, sinks)
      }
      sinks.add(controller)

      const queuedFrames = queuedProjectConversationMuxFrames.get(projectId) ?? []
      for (const frame of queuedFrames) {
        controller.enqueue(encoder.encode(frame))
      }
      queuedProjectConversationMuxFrames.delete(projectId)
    },
    cancel() {
      if (!sink) {
        return
      }
      const sinks = projectConversationMuxStreamControllers.get(projectId)
      if (!sinks) {
        return
      }
      sinks.delete(sink)
      if (sinks.size === 0) {
        projectConversationMuxStreamControllers.delete(projectId)
      }
    },
  })

  return new Response(stream, {
    headers: {
      'content-type': 'text/event-stream',
      'cache-control': 'no-store',
      connection: 'keep-alive',
    },
  })
}

function queueOrBroadcastProjectConversationFrame(conversationId: string, frame: string) {
  const sinks = projectConversationStreamControllers.get(conversationId)
  if (!sinks || sinks.size === 0) {
    const queued = queuedProjectConversationFrames.get(conversationId) ?? []
    queued.push(frame)
    queuedProjectConversationFrames.set(conversationId, queued)
    return
  }

  for (const sink of sinks) {
    try {
      sink.enqueue(encoder.encode(frame))
    } catch {
      sinks.delete(sink)
    }
  }

  if (sinks.size === 0) {
    projectConversationStreamControllers.delete(conversationId)
  }
}

function queueOrBroadcastProjectConversationMuxFrame(projectId: string, frame: string) {
  const sinks = projectConversationMuxStreamControllers.get(projectId)
  if (!sinks || sinks.size === 0) {
    const queued = queuedProjectConversationMuxFrames.get(projectId) ?? []
    queued.push(frame)
    queuedProjectConversationMuxFrames.set(projectId, queued)
    return
  }

  for (const sink of sinks) {
    try {
      sink.enqueue(encoder.encode(frame))
    } catch {
      sinks.delete(sink)
    }
  }

  if (sinks.size === 0) {
    projectConversationMuxStreamControllers.delete(projectId)
  }
}

function queueOrBroadcastProjectConversationEvent(
  conversationId: string,
  event: string,
  payload: Record<string, unknown>,
  sentAt: string,
) {
  queueOrBroadcastProjectConversationFrame(conversationId, encodeSSEFrame(event, payload))

  const conversation = findById(mockState.projectConversations, conversationId)
  const projectId = asString(conversation?.project_id)
  if (!projectId) {
    return
  }

  queueOrBroadcastProjectConversationMuxFrame(
    projectId,
    encodeSSEFrame(event, {
      conversation_id: conversationId,
      sent_at: sentAt,
      payload,
    }),
  )
}

function encodeSSEFrame(event: string, payload: Record<string, unknown>) {
  return `event: ${event}\ndata: ${JSON.stringify(payload)}\n\n`
}

function shiftedIso(offsetMinutes: number) {
  return new Date(Date.parse(nowIso) + offsetMinutes * 60_000).toISOString()
}

function nextProjectConversationSeq(conversationId: string) {
  return (
    mockState.projectConversationEntries.filter((entry) => entry.conversation_id === conversationId)
      .length + 1
  )
}
