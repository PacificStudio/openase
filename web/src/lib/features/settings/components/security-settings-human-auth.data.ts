import {
  createInstanceRoleBinding,
  createOrganizationRoleBinding,
  createProjectRoleBinding,
  deleteInstanceRoleBinding,
  deleteOrganizationRoleBinding,
  deleteProjectRoleBinding,
  getEffectivePermissions,
  listInstanceRoleBindings,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
  type EffectivePermissionsResponse,
  type RoleBinding,
} from '$lib/api/auth'
import {
  createBindingPayload,
  type BindingDraft,
  type ScopeKind,
} from './security-settings-human-auth.model'

export type HumanAuthRbacState = {
  instancePermissions: EffectivePermissionsResponse
  orgPermissions: EffectivePermissionsResponse
  projectPermissions: EffectivePermissionsResponse
  instanceBindings: RoleBinding[]
  orgBindings: RoleBinding[]
  projectBindings: RoleBinding[]
}

type ScopedHumanAuthRbacState = Partial<HumanAuthRbacState>

export async function loadHumanAuthRbacState(
  orgId: string,
  projectId: string,
): Promise<HumanAuthRbacState> {
  const [
    instancePermissions,
    orgPermissions,
    projectPermissions,
    instanceBindings,
    orgBindings,
    projectBindings,
  ] = await Promise.all([
    getEffectivePermissions({}),
    getEffectivePermissions({ orgId }),
    getEffectivePermissions({ projectId }),
    listInstanceRoleBindings(),
    listOrganizationRoleBindings(orgId),
    listProjectRoleBindings(projectId),
  ])
  return {
    instancePermissions,
    orgPermissions,
    projectPermissions,
    instanceBindings,
    orgBindings,
    projectBindings,
  }
}

export async function reloadHumanAuthScope(
  scope: ScopeKind,
  orgId: string,
  projectId: string,
): Promise<ScopedHumanAuthRbacState> {
  if (scope === 'instance') {
    const [instancePermissions, instanceBindings] = await Promise.all([
      getEffectivePermissions({}),
      listInstanceRoleBindings(),
    ])
    return { instancePermissions, instanceBindings }
  }
  if (scope === 'organization') {
    const [orgPermissions, orgBindings] = await Promise.all([
      getEffectivePermissions({ orgId }),
      listOrganizationRoleBindings(orgId),
    ])
    return { orgPermissions, orgBindings }
  }
  const [projectPermissions, projectBindings] = await Promise.all([
    getEffectivePermissions({ projectId }),
    listProjectRoleBindings(projectId),
  ])
  return { projectPermissions, projectBindings }
}

export async function createRoleBindingForScope(
  scope: ScopeKind,
  orgId: string,
  projectId: string,
  draft: BindingDraft,
) {
  const payload = createBindingPayload(scope, draft)
  if (scope === 'instance') {
    return createInstanceRoleBinding(payload)
  }
  if (scope === 'organization') {
    return createOrganizationRoleBinding(orgId, payload)
  }
  return createProjectRoleBinding(projectId, payload)
}

export async function deleteRoleBindingForScope(
  scope: ScopeKind,
  orgId: string,
  projectId: string,
  bindingId: string,
) {
  if (scope === 'instance') {
    return deleteInstanceRoleBinding(bindingId)
  }
  if (scope === 'organization') {
    return deleteOrganizationRoleBinding(orgId, bindingId)
  }
  return deleteProjectRoleBinding(projectId, bindingId)
}

export function scopeDisplayName(scope: ScopeKind) {
  if (scope === 'instance') {
    return 'Instance'
  }
  return scope === 'organization' ? 'Organization' : 'Project'
}
