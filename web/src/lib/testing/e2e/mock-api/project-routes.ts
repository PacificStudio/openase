import {
  DEFAULT_AGENT_ID,
  DEFAULT_PROVIDER_ID,
  DEFAULT_STATUS_IDS,
  DEFAULT_WORKFLOW_ID,
  PROJECT_ID,
  nowIso,
} from './constants'
import {
  asBoolean,
  asNumber,
  asString,
  asStringArray,
  clone,
  findById,
  jsonResponse,
  notFound,
  readBody,
} from './helpers'
import {
  applyRepoMutation,
  createRepoRecord,
  createScheduledJobRecord,
  defaultHarnessContent,
  slugify,
} from './entities'
import { getMockState } from './store'
import {
  createConfiguredGitHubProbe,
  createUnconfiguredGitHubProbe,
  parseGitHubCredentialScope,
  previewToken,
  resolveGitHubCredentialSlot,
  resolveSecuritySettings,
  syncEffectiveGitHubSlot,
} from './security'
import {
  buildTicketDetailPayload,
  buildTicketRunDetailPayload,
  buildTicketRunsPayload,
  createMockTicketRecord,
  createProjectUpdateCommentRecord,
  createProjectUpdateThreadRecord,
  deleteProjectUpdateCommentRecord,
  deleteProjectUpdateThreadRecord,
  readProjectUpdateComments,
  updateProjectUpdateCommentRecord,
  updateProjectUpdateThreadRecord,
} from './ticket-data'

export async function handleProjectRoutes(request: Request, segments: string[], _url: URL) {
  const state = getMockState()
  const projectId = segments[1]
  if (projectId !== PROJECT_ID) {
    return notFound('Project not found.')
  }

  if (segments[2] === 'agents') {
    if (request.method === 'GET') {
      return jsonResponse({
        agents: clone(state.agents.filter((agent) => agent.project_id === projectId)),
      })
    }
    if (request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const providerId = asString(body.provider_id) ?? DEFAULT_PROVIDER_ID
      const provider = findById(state.providers, providerId)
      const agent = {
        id: `agent-${++state.counters.agent}`,
        project_id: projectId,
        provider_id: providerId,
        name: asString(body.name) ?? `Agent ${state.counters.agent}`,
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
      state.agents.push(agent)
      if (provider) {
        provider.updated_at = nowIso
      }
      return jsonResponse({ agent: clone(agent) }, 201)
    }
  }

  if (segments[2] === 'agent-runs' && request.method === 'GET') {
    return jsonResponse({
      agent_runs: clone(state.agentRuns.filter((run) => run.project_id === projectId)),
    })
  }

  if (segments[2] === 'activity' && request.method === 'GET') {
    return jsonResponse({
      events: clone(state.activityEvents.filter((event) => event.project_id === projectId)),
    })
  }

  if (segments[2] === 'updates') {
    if (segments.length === 3 && request.method === 'GET') {
      return jsonResponse({
        threads: clone(state.projectUpdates.filter((thread) => thread.project_id === projectId)),
      })
    }

    if (segments.length === 3 && request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const thread = createProjectUpdateThreadRecord(state, projectId, body)
      state.projectUpdates.unshift(thread)
      return jsonResponse({ thread: clone(thread) }, 201)
    }

    const thread = findById(state.projectUpdates, segments[3])
    if (!thread) {
      return notFound('Project update not found.')
    }

    if (segments.length === 4 && request.method === 'PATCH') {
      const body = await readBody<Record<string, unknown>>(request)
      updateProjectUpdateThreadRecord(thread, body)
      return jsonResponse({ thread: clone(thread) })
    }

    if (segments.length === 4 && request.method === 'DELETE') {
      deleteProjectUpdateThreadRecord(thread)
      return jsonResponse({ deleted_thread_id: asString(thread.id) })
    }

    if (segments.length === 5 && segments[4] === 'comments' && request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const comment = createProjectUpdateCommentRecord(state, asString(thread.id) ?? '', body)
      const comments = readProjectUpdateComments(thread)
      comments.push(comment)
      thread.comments = comments
      thread.comment_count = comments.filter((item) => !item.is_deleted).length
      thread.last_activity_at = comment.updated_at
      thread.updated_at = comment.updated_at
      return jsonResponse({ comment: clone(comment) }, 201)
    }

    if (segments.length === 6 && segments[4] === 'comments') {
      const comments = readProjectUpdateComments(thread)
      const comment = comments.find((item) => item.id === segments[5])
      if (!comment) {
        return notFound('Project update comment not found.')
      }

      if (request.method === 'PATCH') {
        const body = await readBody<Record<string, unknown>>(request)
        updateProjectUpdateCommentRecord(comment, body)
        thread.last_activity_at = comment.updated_at
        thread.updated_at = comment.updated_at
        return jsonResponse({ comment: clone(comment) })
      }

      if (request.method === 'DELETE') {
        deleteProjectUpdateCommentRecord(comment)
        thread.comment_count = comments.filter((item) => !item.is_deleted).length
        thread.last_activity_at = comment.updated_at
        thread.updated_at = comment.updated_at
        return jsonResponse({ deleted_comment_id: asString(comment.id) })
      }
    }
  }

  if (segments[2] === 'security-settings') {
    const security = resolveSecuritySettings(state, projectId)

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
        tickets: clone(state.tickets.filter((ticket) => ticket.project_id === projectId)),
      })
    }
    if (segments.length === 3 && request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const statusId = asString(body.status_id) ?? DEFAULT_STATUS_IDS.todo
      const statusName =
        asString(state.statuses.find((status) => status.id === statusId)?.name) ?? 'Todo'
      const ticket = createMockTicketRecord({
        id: `ticket-${state.tickets.length + 1}`,
        identifier: `ASE-${200 + state.tickets.length + 1}`,
        title: asString(body.title) ?? `Ticket ${state.tickets.length + 1}`,
        description: asString(body.description) ?? '',
        statusId,
        statusName,
        workflowId: DEFAULT_WORKFLOW_ID,
      })
      state.tickets.unshift(ticket)
      return jsonResponse({ ticket: clone(ticket) }, 201)
    }
    if (segments.length === 5 && segments[4] === 'detail' && request.method === 'GET') {
      const payload = buildTicketDetailPayload(state, segments[3])
      if (!payload) {
        return notFound('Ticket detail not found.')
      }
      return jsonResponse(payload)
    }
    if (segments.length === 5 && segments[4] === 'runs' && request.method === 'GET') {
      return jsonResponse(buildTicketRunsPayload(state, segments[3]))
    }
    if (segments.length === 6 && segments[4] === 'runs' && request.method === 'GET') {
      const payload = buildTicketRunDetailPayload(state, segments[3], segments[5])
      if (!payload) {
        return notFound('Ticket run not found.')
      }
      return jsonResponse(payload)
    }
  }

  if (segments[2] === 'workflows') {
    if (request.method === 'GET') {
      return jsonResponse({
        workflows: clone(state.workflows.filter((workflow) => workflow.project_id === projectId)),
      })
    }
    if (request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const nextId = `workflow-${++state.counters.workflow}`
      const workflow = {
        id: nextId,
        project_id: projectId,
        name: asString(body.name) ?? `Workflow ${state.counters.workflow}`,
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
      state.workflows.push(workflow)
      state.harnessByWorkflowId[nextId] = {
        content:
          asString(body.harness_content) ??
          defaultHarnessContent(asString(body.name) ?? `Workflow ${state.counters.workflow}`),
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
        repos: clone(state.repos.filter((repo) => repo.project_id === projectId)),
      })
    }
    if (segments.length === 3 && request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const repo = createRepoRecord(state, body)
      state.repos.push(repo)
      return jsonResponse({ repo: clone(repo) }, 201)
    }
    if (segments.length === 4) {
      const repoId = segments[3]
      const repo = findById(state.repos, repoId)
      if (!repo) {
        return notFound('Repository not found.')
      }
      if (request.method === 'PATCH') {
        const body = await readBody<Record<string, unknown>>(request)
        applyRepoMutation(repo, body)
        return jsonResponse({ repo: clone(repo) })
      }
      if (request.method === 'DELETE') {
        state.repos = state.repos.filter((item) => item.id !== repoId)
        return jsonResponse({ repo: clone(repo) })
      }
    }
  }

  if (segments[2] === 'statuses' && request.method === 'GET') {
    return jsonResponse({
      statuses: clone(state.statuses.filter((status) => status.project_id === projectId)),
    })
  }

  if (segments[2] === 'skills' && request.method === 'GET') {
    return jsonResponse({
      skills: clone(state.skills.filter((skill) => skill.project_id === projectId)),
    })
  }

  if (segments[2] === 'skills' && request.method === 'POST') {
    const body = await readBody<Record<string, unknown>>(request)
    const name = asString(body.name)?.trim() || `skill-${++state.counters.skill}`
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
    state.skills.unshift(created)
    return jsonResponse({ skill: clone(created), content }, 201)
  }

  if (segments[2] === 'scheduled-jobs') {
    if (request.method === 'GET') {
      return jsonResponse({
        scheduled_jobs: clone(state.scheduledJobs.filter((job) => job.project_id === projectId)),
      })
    }
    if (request.method === 'POST') {
      const body = await readBody<Record<string, unknown>>(request)
      const job = createScheduledJobRecord(state, body)
      state.scheduledJobs.push(job)
      return jsonResponse({ scheduled_job: clone(job) }, 201)
    }
  }

  return notFound('Mock project route not found.')
}
