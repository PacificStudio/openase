import { ORG_ID, nowIso } from './constants'
import {
  asNumber,
  asObject,
  asSecretBindings,
  asString,
  asStringArray,
  clone,
  findById,
  jsonResponse,
  notFound,
  readBody,
} from './helpers'
import { applyMachineMutation, createMachineRecord } from './entities'
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
    machine.resources = {
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
