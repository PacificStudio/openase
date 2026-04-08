<script lang="ts">
  import { type EffectivePermissionsResponse, type RoleBinding } from '$lib/api/auth'
  import { authStore } from '$lib/stores/auth.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Shield } from '@lucide/svelte'
  import SecuritySettingsHumanAuthAuthenticatedView from './security-settings-human-auth-authenticated-view.svelte'
  import SecuritySettingsHumanAuthGuideLinks from './security-settings-human-auth-guide-links.svelte'
  import SecuritySettingsHumanAuthSignInHint from './security-settings-human-auth-sign-in-hint.svelte'
  import SecuritySettingsHumanAuthSetupPanel from './security-settings-human-auth-setup-panel.svelte'
  import SecuritySettingsHumanAuthSummary from './security-settings-human-auth-summary.svelte'
  import {
    createRoleBindingForScope,
    deleteRoleBindingForScope,
    loadHumanAuthRbacState,
    reloadHumanAuthScope,
    scopeDisplayName,
  } from './security-settings-human-auth.data'
  import {
    defaultBindingDraftForScope,
    formatError,
    type BindingDraft,
    type ScopeKind,
  } from './security-settings-human-auth.model'
  import {
    type ApprovalPoliciesSummary,
    type SecuritySettingsSecurity,
  } from './security-settings-human-auth.types'

  let { security = null }: { security?: SecuritySettingsSecurity | null } = $props()

  let loading = $state(false)
  let error = $state('')
  let mutationKey = $state('')
  let instancePermissions = $state<EffectivePermissionsResponse | null>(null)
  let orgPermissions = $state<EffectivePermissionsResponse | null>(null)
  let projectPermissions = $state<EffectivePermissionsResponse | null>(null)
  let instanceBindings = $state<RoleBinding[]>([])
  let orgBindings = $state<RoleBinding[]>([])
  let projectBindings = $state<RoleBinding[]>([])
  let instanceDraft = $state<BindingDraft>(defaultBindingDraftForScope('instance'))
  let orgDraft = $state<BindingDraft>(defaultBindingDraftForScope('organization'))
  let projectDraft = $state<BindingDraft>(defaultBindingDraftForScope('project'))
  const approvalPolicies = $derived<ApprovalPoliciesSummary | null>(
    security?.approval_policies ?? null,
  )
  const authSummary = $derived(security?.auth ?? null)
  const currentOrgId = $derived(appStore.currentOrg?.id ?? '')
  const currentProjectId = $derived(appStore.currentProject?.id ?? '')
  const currentGroups = $derived(projectPermissions?.groups ?? orgPermissions?.groups ?? [])
  const canManageInstanceBindings = $derived(
    instancePermissions?.permissions.includes('rbac.manage') ?? false,
  )
  const canManageOrgBindings = $derived(
    orgPermissions?.permissions.includes('rbac.manage') ?? false,
  )
  const canManageProjectBindings = $derived(
    projectPermissions?.permissions.includes('rbac.manage') ?? false,
  )

  $effect(() => {
    const loginRequired = authStore.loginRequired
    const authenticated = authStore.authenticated
    const orgId = currentOrgId
    const projectId = currentProjectId

    if (!loginRequired || !authenticated || !orgId || !projectId) {
      loading = false
      error = ''
      instancePermissions = null
      orgPermissions = null
      projectPermissions = null
      instanceBindings = []
      orgBindings = []
      projectBindings = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const nextState = await loadHumanAuthRbacState(orgId, projectId)
        if (cancelled) {
          return
        }

        instancePermissions = nextState.instancePermissions
        orgPermissions = nextState.orgPermissions
        projectPermissions = nextState.projectPermissions
        instanceBindings = nextState.instanceBindings
        orgBindings = nextState.orgBindings
        projectBindings = nextState.projectBindings
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
    if (scope === 'instance') {
      instanceDraft = defaultBindingDraftForScope('instance')
      return
    }
    if (scope === 'organization') {
      orgDraft = defaultBindingDraftForScope('organization')
      return
    }
    projectDraft = defaultBindingDraftForScope('project')
  }

  async function reloadScope(scope: ScopeKind) {
    const orgId = currentOrgId
    const projectId = currentProjectId

    if (!orgId || !projectId) {
      return
    }

    const nextState = await reloadHumanAuthScope(scope, orgId, projectId)
    instancePermissions = nextState.instancePermissions ?? instancePermissions
    orgPermissions = nextState.orgPermissions ?? orgPermissions
    projectPermissions = nextState.projectPermissions ?? projectPermissions
    instanceBindings = nextState.instanceBindings ?? instanceBindings
    orgBindings = nextState.orgBindings ?? orgBindings
    projectBindings = nextState.projectBindings ?? projectBindings
  }

  async function handleCreateBinding(scope: ScopeKind) {
    const orgId = currentOrgId
    const projectId = currentProjectId
    const draft =
      scope === 'instance' ? instanceDraft : scope === 'organization' ? orgDraft : projectDraft

    if (!orgId || !projectId) {
      return
    }

    const key = `${scope}:create`
    mutationKey = key
    error = ''

    try {
      await createRoleBindingForScope(scope, orgId, projectId, draft)
      await reloadScope(scope)
      resetDraft(scope)
      toastStore.success(`${scopeDisplayName(scope)} role binding added.`)
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
      await deleteRoleBindingForScope(scope, orgId, projectId, bindingId)
      await reloadScope(scope)
      toastStore.success(`${scopeDisplayName(scope)} role binding deleted.`)
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to delete role binding.')
      error = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }

  function updateDraft(scope: ScopeKind, nextDraft: BindingDraft) {
    if (scope === 'instance') {
      instanceDraft = nextDraft
      return
    }
    if (scope === 'organization') {
      orgDraft = nextDraft
      return
    }
    projectDraft = nextDraft
  }

  function patchDraft(scope: ScopeKind, patch: Partial<BindingDraft>) {
    const currentDraft =
      scope === 'instance' ? instanceDraft : scope === 'organization' ? orgDraft : projectDraft
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
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <Shield class="text-muted-foreground size-4" />
    <h3 class="text-sm font-semibold">Human access and IAM</h3>
  </div>

  {#if authSummary}
    <SecuritySettingsHumanAuthSummary
      authMode={authSummary.active_mode}
      configuredMode={authSummary.configured_mode}
      issuerURL={authSummary.issuer_url ?? ''}
      user={authStore.user}
      bootstrapSummary={authSummary.bootstrap_state.summary}
      publicExposureRisk={authSummary.public_exposure_risk}
      localPrincipal={authSummary.local_principal}
    />
  {/if}

  {#if authSummary && !authStore.loginRequired}
    <SecuritySettingsHumanAuthSetupPanel
      auth={authSummary}
      projectId={currentProjectId}
      onSecurityChange={(nextSecurity) => (security = nextSecurity)}
    />
  {:else if authStore.loginRequired && !authStore.authenticated}
    <SecuritySettingsHumanAuthSignInHint />
  {:else if loading}
    <div class="space-y-3">
      <div class="bg-muted h-16 animate-pulse rounded-lg"></div>
      <div class="bg-muted h-32 animate-pulse rounded-lg"></div>
    </div>
  {:else if authStore.loginRequired}
    <SecuritySettingsHumanAuthAuthenticatedView
      user={authStore.user}
      {currentOrgId}
      currentOrgName={appStore.currentOrg?.name ?? ''}
      currentProjectName={appStore.currentProject?.name ?? ''}
      {currentGroups}
      {approvalPolicies}
      {error}
      {instancePermissions}
      {orgPermissions}
      {projectPermissions}
      {instanceBindings}
      {orgBindings}
      {projectBindings}
      {canManageInstanceBindings}
      {canManageOrgBindings}
      {canManageProjectBindings}
      {instanceDraft}
      {orgDraft}
      {projectDraft}
      {mutationKey}
      onDraftSubjectKind={handleDraftSubjectKind}
      onDraftSubjectKey={handleDraftSubjectKey}
      onDraftRoleKey={handleDraftRoleKey}
      onDraftExpiresAt={handleDraftExpiresAt}
      onCreateBinding={(scope) => void handleCreateBinding(scope)}
      onDeleteBinding={(scope, bindingId) => void handleDeleteBinding(scope, bindingId)}
    />
  {/if}

  {#if authSummary}
    <SecuritySettingsHumanAuthGuideLinks docs={authSummary.docs} />
  {/if}
</div>
