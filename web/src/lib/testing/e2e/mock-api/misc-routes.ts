import { ORG_ID, nowIso } from './constants'
import {
  asBoolean,
  asNumber,
  asObject,
  asObjectArray,
  asSecretBindings,
  asString,
  asStringArray,
  clone,
  decodeBase64UTF8,
  findById,
  jsonResponse,
  notFound,
  readBody,
} from './helpers'
import {
  applyMachineMutation,
  applyScheduledJobMutation,
  createMachineRecord,
  dedupeById,
} from './entities'
import { getMockState } from './store'

export async function handleOrgRoutes(request: Request, segments: string[]) {
  const state = getMockState()
  if (segments.length === 1 && request.method === 'GET') {
    return jsonResponse({ organizations: clone(state.organizations) })
  }

  const orgId = segments[1]
  if (orgId !== ORG_ID) {
    return notFound('Organization not found.')
  }

  if (segments[2] === 'projects' && request.method === 'GET') {
    return jsonResponse({
      projects: clone(state.projects.filter((project) => project.org_id === orgId)),
    })
  }
  if (segments[2] === 'summary' && request.method === 'GET') {
    return jsonResponse({
      organization: {
        id: ORG_ID,
        name: 'OpenASE E2E',
        slug: 'openase-e2e',
        project_count: 1,
        active_project_count: 1,
      },
      projects: clone(state.projects.filter((project) => project.org_id === orgId)),
    })
  }
  if (segments[2] === 'token-usage' && request.method === 'GET') {
    return jsonResponse({
      summary: {
        total_tokens: 4200,
        avg_daily_tokens: 600,
        peak_day: {
          date: '2026-03-25',
          total_tokens: 900,
        },
      },
      days: [],
    })
  }
  if (segments[2] === 'members' && request.method === 'GET') {
    return jsonResponse({
      memberships: [
        {
          id: 'membership-1',
          organization_id: ORG_ID,
          user_id: 'user-1',
          email: 'alice@example.com',
          role: 'org_admin',
          status: 'active',
          invited_by: 'user:seed',
          invited_at: nowIso,
          accepted_at: nowIso,
          created_at: nowIso,
          updated_at: nowIso,
          user: {
            id: 'user-1',
            primary_email: 'alice@example.com',
            display_name: 'Alice Admin',
            avatar_url: '',
          },
        },
      ],
    })
  }
  if (segments[2] === 'providers' && request.method === 'GET') {
    return jsonResponse({
      providers: clone(state.providers.filter((provider) => provider.org_id === orgId)),
    })
  }
  if (segments[2] === 'machines') {
    if (request.method === 'GET') {
      return jsonResponse({
        machines: clone(state.machines.filter((machine) => machine.org_id === orgId)),
      })
    }
    if (request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const machine = createMachineRecord(state, body)
      state.machines.unshift(machine)
      return jsonResponse({ machine: clone(machine) }, 201)
    }
  }

  return notFound('Mock org route not found.')
}

export async function handleMachineRoutes(request: Request, segments: string[]) {
  const state = getMockState()
  const machineId = segments[1]
  const machine = findById(state.machines, machineId)
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
      state.machines = state.machines.filter((item) => item.id !== machineId)
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

export async function handleProviderRoutes(request: Request, segments: string[]) {
  const state = getMockState()
  const providerId = segments[1]
  const provider = findById(state.providers, providerId)
  if (!provider) {
    return notFound('Provider not found.')
  }
  if (request.method !== 'PATCH') {
    return notFound('Mock provider route not found.')
  }

  const body = await readBody<Record<string, unknown>>(request)
  if (body.machine_id) {
    const machine = findById(state.machines, asString(body.machine_id) ?? '')
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
  provider.secret_bindings = asSecretBindings(body.secret_bindings)

  return jsonResponse({ provider: clone(provider) })
}

export async function handleAgentRoutes(request: Request, segments: string[]) {
  const state = getMockState()
  const agentId = segments[1]
  const agent = findById(state.agents, agentId)
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

export async function handleWorkflowRoutes(request: Request, segments: string[]) {
  const state = getMockState()
  const workflowId = segments[1]
  const workflow = findById(state.workflows, workflowId)
  if (!workflow) {
    return notFound('Workflow not found.')
  }

  if (segments[2] === 'harness') {
    if (segments[3] === 'history' && request.method === 'GET') {
      return jsonResponse({
        history: clone(state.harnessByWorkflowId[workflowId].history),
      })
    }
    if (request.method === 'GET') {
      const harness = state.harnessByWorkflowId[workflowId]
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
      const current = state.harnessByWorkflowId[workflowId]
      const nextVersion = current.version + 1
      const content = asString(body.content) ?? current.content
      state.harnessByWorkflowId[workflowId] = {
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

    state.skills = state.skills.map((skill) => {
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

    const harness = state.harnessByWorkflowId[workflowId]
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

export async function handleSkillRoutes(request: Request, segments: string[]) {
  const state = getMockState()
  const skillId = segments[1]
  const skill = findById(state.skills, skillId)
  if (!skill) {
    return notFound('Skill not found.')
  }

  const boundWorkflowRefs = (entries: Record<string, unknown>[] | undefined) =>
    (entries ?? []).map((entry) => {
      const workflowId = asString(entry.id)
      const workflow = workflowId ? findById(state.workflows, workflowId) : null
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
    state.skills = state.skills.filter((item) => item.id !== skillId)
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

export async function handleScheduledJobRoutes(request: Request, segments: string[]) {
  const state = getMockState()
  const jobId = segments[1]
  const job = findById(state.scheduledJobs, jobId)
  if (!job) {
    return notFound('Scheduled job not found.')
  }

  if (segments.length === 2 && request.method === 'PATCH') {
    const body = await readBody<Record<string, unknown>>(request)
    applyScheduledJobMutation(job, body)
    return jsonResponse({ scheduled_job: clone(job) })
  }
  if (segments.length === 2 && request.method === 'DELETE') {
    state.scheduledJobs = state.scheduledJobs.filter((item) => item.id !== jobId)
    return jsonResponse({ scheduled_job: clone(job) })
  }
  if (segments[2] === 'trigger' && request.method === 'POST') {
    job.last_run_at = nowIso
    job.next_run_at = '2026-03-28T02:00:00.000Z'
    return jsonResponse({ scheduled_job: clone(job) })
  }

  return notFound('Mock scheduled job route not found.')
}

export async function handleHarnessRoutes(request: Request, segments: string[]) {
  const state = getMockState()
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
    return jsonResponse(clone(state.harnessVariables))
  }

  return notFound('Mock harness route not found.')
}
