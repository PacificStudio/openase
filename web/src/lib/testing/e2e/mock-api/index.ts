import { PROJECT_ID } from './constants'
import { buildAppContextPayload } from './app-context'
import { clone, jsonResponse, noContentResponse, notFound, readBody } from './helpers'
import { seedBoardState } from './initial-state'
import { handleChatRoutes } from './chat-routes'
import {
  handleAgentRoutes,
  handleHarnessRoutes,
  handleMachineRoutes,
  handleOrgRoutes,
  handleProviderRoutes,
  handleScheduledJobRoutes,
  handleSkillRoutes,
  handleWorkflowRoutes,
} from './misc-routes'
import { handleProjectRoutes } from './project-routes'
import { resolveSecuritySettings } from './security'
import { getMockState, resetMockState } from './store'
import { streamResponse } from './streams'

export { resetMockState }

export async function handleMockApi(request: Request, url: URL): Promise<Response | null> {
  if (!url.pathname.startsWith('/api/v1/')) {
    return null
  }

  if (url.pathname === '/api/v1/auth/session' && request.method === 'GET') {
    return jsonResponse({
      auth_mode: 'disabled',
      login_required: false,
      authenticated: true,
      principal_kind: 'local_bootstrap',
      auth_configured: false,
      session_governance_available: false,
      can_manage_auth: true,
      issuer_url: '',
      user: null,
      csrf_token: '',
      roles: ['instance_admin'],
      permissions: ['security_setting.read', 'security_setting.update', 'rbac.manage'],
    })
  }

  if (url.pathname === '/api/v1/auth/me/permissions' && request.method === 'GET') {
    const projectId = url.searchParams.get('project_id') ?? ''
    const orgId = url.searchParams.get('org_id') ?? ''
    const scopeKind = projectId ? 'project' : orgId ? 'organization' : 'instance'
    const scopeID = projectId || orgId
    const roles =
      scopeKind === 'project'
        ? ['project_admin']
        : scopeKind === 'organization'
          ? ['instance_admin']
          : ['instance_admin']
    const permissions =
      scopeKind === 'project'
        ? ['project.read', 'project.update', 'rbac.manage']
        : scopeKind === 'organization'
          ? ['org.read', 'org.update', 'rbac.manage']
          : ['security_setting.read', 'security_setting.update', 'rbac.manage']
    return jsonResponse({
      auth_mode: 'disabled',
      login_required: false,
      authenticated: true,
      principal_kind: 'local_bootstrap',
      auth_configured: false,
      session_governance_available: false,
      can_manage_auth: true,
      scope: {
        kind: scopeKind,
        id: scopeID,
      },
      roles,
      permissions,
      groups: [],
    })
  }

  if (url.pathname === '/api/v1/admin/auth' && request.method === 'GET') {
    return jsonResponse({ auth: clone(resolveSecuritySettings(getMockState(), PROJECT_ID).auth) })
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
    seedBoardState(getMockState(), body.counts_by_status_id ?? {})
    return jsonResponse({ ok: true })
  }

  if (url.pathname.endsWith('/stream') && !url.pathname.startsWith('/api/v1/chat/')) {
    return streamResponse()
  }

  if (url.pathname === '/api/v1/app-context' && request.method === 'GET') {
    return jsonResponse(buildAppContextPayload(getMockState(), url))
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
    return jsonResponse({ roles: clone(getMockState().builtinRoles) })
  }

  return notFound(`Mock route not implemented: ${request.method} ${url.pathname}`)
}
