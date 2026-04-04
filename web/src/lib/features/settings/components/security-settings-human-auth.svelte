<script lang="ts">
  import {
    createOrganizationRoleBinding,
    createProjectRoleBinding,
    deleteOrganizationRoleBinding,
    deleteProjectRoleBinding,
    getEffectivePermissions,
    listOrganizationRoleBindings,
    listProjectRoleBindings,
    type EffectivePermissionsResponse,
    type RoleBinding,
  } from '$lib/api/auth'
  import { authStore } from '$lib/stores/auth.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Shield } from '@lucide/svelte'
  import SecuritySettingsHumanAuthAuthenticatedView from './security-settings-human-auth-authenticated-view.svelte'
  import SecuritySettingsHumanAuthSummary from './security-settings-human-auth-summary.svelte'
  import {
    createBindingPayload,
    defaultBindingDraft,
    formatError,
    type BindingDraft,
    type ScopeKind,
  } from './security-settings-human-auth.model'

  type ApprovalPoliciesSummary = {
    status: string
    rules_count: number
    summary: string
  }

  let { approvalPolicies = null }: { approvalPolicies?: ApprovalPoliciesSummary | null } = $props()

  let loading = $state(false)
  let error = $state('')
  let mutationKey = $state('')
  let orgPermissions = $state<EffectivePermissionsResponse | null>(null)
  let projectPermissions = $state<EffectivePermissionsResponse | null>(null)
  let orgBindings = $state<RoleBinding[]>([])
  let projectBindings = $state<RoleBinding[]>([])
  let orgDraft = $state<BindingDraft>(defaultBindingDraft('org_member'))
  let projectDraft = $state<BindingDraft>(defaultBindingDraft())

  const currentOrgId = $derived(appStore.currentOrg?.id ?? '')
  const currentProjectId = $derived(appStore.currentProject?.id ?? '')
  const currentGroups = $derived(projectPermissions?.groups ?? orgPermissions?.groups ?? [])
  const canManageOrgBindings = $derived(
    orgPermissions?.permissions.includes('rbac.manage') ?? false,
  )
  const canManageProjectBindings = $derived(
    projectPermissions?.permissions.includes('rbac.manage') ?? false,
  )

  $effect(() => {
    const authMode = authStore.authMode
    const authenticated = authStore.authenticated
    const orgId = currentOrgId
    const projectId = currentProjectId

    if (authMode !== 'oidc' || !authenticated || !orgId || !projectId) {
      loading = false
      error = ''
      orgPermissions = null
      projectPermissions = null
      orgBindings = []
      projectBindings = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const [nextOrgPermissions, nextProjectPermissions, nextOrgBindings, nextProjectBindings] =
          await Promise.all([
            getEffectivePermissions({ orgId }),
            getEffectivePermissions({ projectId }),
            listOrganizationRoleBindings(orgId),
            listProjectRoleBindings(projectId),
          ])
        if (cancelled) {
          return
        }

        orgPermissions = nextOrgPermissions
        projectPermissions = nextProjectPermissions
        orgBindings = nextOrgBindings
        projectBindings = nextProjectBindings
      } catch (caughtError) {
        if (cancelled) {
          return
        }
        error = formatError(caughtError, 'Failed to load human auth and RBAC state.')
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  function resetDraft(scope: ScopeKind) {
    if (scope === 'organization') {
      orgDraft = defaultBindingDraft('org_member')
      return
    }
    projectDraft = defaultBindingDraft()
  }

  async function reloadScope(scope: ScopeKind) {
    const orgId = currentOrgId
    const projectId = currentProjectId

    if (!orgId || !projectId) {
      return
    }

    if (scope === 'organization') {
      const [nextPermissions, nextBindings] = await Promise.all([
        getEffectivePermissions({ orgId }),
        listOrganizationRoleBindings(orgId),
      ])
      orgPermissions = nextPermissions
      orgBindings = nextBindings
      return
    }

    const [nextPermissions, nextBindings] = await Promise.all([
      getEffectivePermissions({ projectId }),
      listProjectRoleBindings(projectId),
    ])
    projectPermissions = nextPermissions
    projectBindings = nextBindings
  }

  async function handleCreateBinding(scope: ScopeKind) {
    const orgId = currentOrgId
    const projectId = currentProjectId
    const draft = scope === 'organization' ? orgDraft : projectDraft

    if (!orgId || !projectId) {
      return
    }

    const key = `${scope}:create`
    mutationKey = key
    error = ''

    try {
      const payload = createBindingPayload(scope, draft)
      if (scope === 'organization') {
        await createOrganizationRoleBinding(orgId, payload)
      } else {
        await createProjectRoleBinding(projectId, payload)
      }
      await reloadScope(scope)
      resetDraft(scope)
      toastStore.success(
        `${scope === 'organization' ? 'Organization' : 'Project'} role binding added.`,
      )
    } catch (caughtError) {
      const message =
        caughtError instanceof Error ? caughtError.message : 'Failed to create role binding.'
      error = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }

  async function handleDeleteBinding(scope: ScopeKind, bindingId: string) {
    const orgId = currentOrgId
    const projectId = currentProjectId
    if (!orgId || !projectId) {
      return
    }

    const key = `${scope}:delete:${bindingId}`
    mutationKey = key
    error = ''

    try {
      if (scope === 'organization') {
        await deleteOrganizationRoleBinding(orgId, bindingId)
      } else {
        await deleteProjectRoleBinding(projectId, bindingId)
      }
      await reloadScope(scope)
      toastStore.success(
        `${scope === 'organization' ? 'Organization' : 'Project'} role binding deleted.`,
      )
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to delete role binding.')
      error = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }

  function updateDraft(scope: ScopeKind, nextDraft: BindingDraft) {
    if (scope === 'organization') {
      orgDraft = nextDraft
      return
    }
    projectDraft = nextDraft
  }

  function patchDraft(scope: ScopeKind, patch: Partial<BindingDraft>) {
    const currentDraft = scope === 'organization' ? orgDraft : projectDraft
    updateDraft(scope, { ...currentDraft, ...patch })
  }

  function handleDraftSubjectKind(scope: ScopeKind, value: BindingDraft['subjectKind']) {
    patchDraft(scope, { subjectKind: value })
  }

  function handleDraftSubjectKey(scope: ScopeKind, value: string) {
    patchDraft(scope, { subjectKey: value })
  }

  function handleDraftRoleKey(scope: ScopeKind, value: string) {
    patchDraft(scope, { roleKey: value })
  }

  function handleDraftExpiresAt(scope: ScopeKind, value: string) {
    patchDraft(scope, { expiresAtLocal: value })
  }

  function handleCreateBindingClick(scope: ScopeKind) {
    void handleCreateBinding(scope)
  }

  function handleDeleteBindingClick(scope: ScopeKind, bindingId: string) {
    void handleDeleteBinding(scope, bindingId)
  }
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <Shield class="text-muted-foreground size-4" />
    <h3 class="text-sm font-semibold">Human access and RBAC</h3>
  </div>

  <SecuritySettingsHumanAuthSummary
    authMode={authStore.authMode}
    issuerURL={authStore.issuerURL}
    user={authStore.user}
  />

  {#if authStore.authMode !== 'oidc'}
    <div class="bg-muted/20 text-muted-foreground rounded-lg border px-4 py-3 text-sm">
      Human auth is disabled. Enable <code>auth.mode=oidc</code> to enforce browser login and RBAC.
    </div>
  {:else if !authStore.authenticated}
    <div class="bg-muted/20 text-muted-foreground rounded-lg border px-4 py-3 text-sm">
      Sign in to inspect effective permissions and manage role bindings.
    </div>
  {:else if loading}
    <div class="space-y-3">
      <div class="bg-muted h-16 animate-pulse rounded-lg"></div>
      <div class="bg-muted h-32 animate-pulse rounded-lg"></div>
    </div>
  {:else}
    <SecuritySettingsHumanAuthAuthenticatedView
      user={authStore.user}
      currentOrgName={appStore.currentOrg?.name ?? ''}
      currentProjectName={appStore.currentProject?.name ?? ''}
      {currentGroups}
      {approvalPolicies}
      {error}
      {orgPermissions}
      {projectPermissions}
      {orgBindings}
      {projectBindings}
      {canManageOrgBindings}
      {canManageProjectBindings}
      {orgDraft}
      {projectDraft}
      {mutationKey}
      onDraftSubjectKind={handleDraftSubjectKind}
      onDraftSubjectKey={handleDraftSubjectKey}
      onDraftRoleKey={handleDraftRoleKey}
      onDraftExpiresAt={handleDraftExpiresAt}
      onCreateBinding={handleCreateBindingClick}
      onDeleteBinding={handleDeleteBindingClick}
    />
  {/if}
</div>
