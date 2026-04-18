import { nowIso } from './constants'
import {
  asBoolean,
  asNumber,
  asObjectArray,
  asString,
  asStringArray,
  clone,
  decodeBase64UTF8,
  findById,
  jsonResponse,
  notFound,
  readBody,
} from './helpers'
import { applyScheduledJobMutation, dedupeById } from './entities'
import { getMockState } from './store'

function boundWorkflowRefs(
  workflowEntries: Record<string, unknown>[] | undefined,
  workflowState: Record<string, unknown>[],
) {
  return (workflowEntries ?? []).map((entry) => {
    const workflowId = asString(entry.id)
    const workflow = workflowId ? findById(workflowState, workflowId) : null
    return workflow ? { id: workflow.id, name: workflow.name } : { id: workflowId }
  })
}

function currentSkillFiles(skillId: string, skill: Record<string, unknown>) {
  return clone(
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
      return jsonResponse({ history: clone(state.harnessByWorkflowId[workflowId].history) })
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

  if (segments[2] === 'history' && request.method === 'GET') {
    return jsonResponse({ history: clone((skill.history as Record<string, unknown>[]) ?? []) })
  }
  if (segments[2] === 'files' && request.method === 'GET') {
    return jsonResponse({ files: currentSkillFiles(skillId, skill) })
  }
  if (segments.length === 2 && request.method === 'GET') {
    return jsonResponse({
      skill: clone({
        ...skill,
        bound_workflows: boundWorkflowRefs(
          skill.bound_workflows as Record<string, unknown>[],
          state.workflows,
        ),
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
        bound_workflows: boundWorkflowRefs(
          skill.bound_workflows as Record<string, unknown>[],
          state.workflows,
        ),
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
    skill.bound_workflows =
      segments[2] === 'bind'
        ? dedupeById([...existing, ...workflowIds.map((workflowId) => ({ id: workflowId }))])
        : existing.filter((item) => !workflowIds.includes(asString(item.id) ?? ''))
    return jsonResponse({
      skill: clone({
        ...skill,
        bound_workflows: boundWorkflowRefs(
          skill.bound_workflows as Record<string, unknown>[],
          state.workflows,
        ),
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
