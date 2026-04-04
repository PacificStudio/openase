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
  import { ApiError } from '$lib/api/client'
  import { authStore } from '$lib/stores/auth.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { LockKeyhole, Shield, Users } from '@lucide/svelte'

  type ScopeKind = 'organization' | 'project'
  type SubjectKind = 'user' | 'group'

  type RoleOption = {
    key: string
    label: string
    summary: string
  }

  type BindingDraft = {
    subjectKind: SubjectKind
    subjectKey: string
    roleKey: string
    expiresAtLocal: string
  }

  type BindingSectionView = {
    scope: ScopeKind
    bindings: RoleBinding[]
    canManage: boolean
  }

  const roleOptions: RoleOption[] = [
    {
      key: 'instance_admin',
      label: 'Instance Admin',
      summary: 'Full control across orgs, projects, security, jobs, and RBAC.',
    },
    {
      key: 'org_owner',
      label: 'Org Owner',
      summary: 'Full organization and project control, including RBAC.',
    },
    {
      key: 'org_admin',
      label: 'Org Admin',
      summary: 'Manage organization settings and descendant project operations.',
    },
    {
      key: 'org_member',
      label: 'Org Member',
      summary: 'Read organization resources and perform standard project work.',
    },
    {
      key: 'project_admin',
      label: 'Project Admin',
      summary: 'Manage project settings, repos, workflows, jobs, security, and bindings.',
    },
    {
      key: 'project_operator',
      label: 'Project Operator',
      summary: 'Operate project runtime surfaces without full security or RBAC control.',
    },
    {
      key: 'project_reviewer',
      label: 'Project Reviewer',
      summary: 'Review conversations, tickets, and proposals with approval capability.',
    },
    {
      key: 'project_member',
      label: 'Project Member',
      summary: 'Standard contributor access for tickets, comments, and conversations.',
    },
    {
      key: 'project_viewer',
      label: 'Project Viewer',
      summary: 'Read-only access to project state and diagnostics.',
    },
  ]

  function defaultBindingDraft(): BindingDraft {
    return {
      subjectKind: 'user',
      subjectKey: '',
      roleKey: 'project_member',
      expiresAtLocal: '',
    }
  }

  let loading = $state(false)
  let error = $state('')
  let mutationKey = $state('')
  let orgPermissions = $state<EffectivePermissionsResponse | null>(null)
  let projectPermissions = $state<EffectivePermissionsResponse | null>(null)
  let orgBindings = $state<RoleBinding[]>([])
  let projectBindings = $state<RoleBinding[]>([])
  let orgDraft = $state<BindingDraft>({ ...defaultBindingDraft(), roleKey: 'org_member' })
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
  const bindingSections = $derived<BindingSectionView[]>([
    {
      scope: 'organization',
      bindings: orgBindings,
      canManage: canManageOrgBindings,
    },
    {
      scope: 'project',
      bindings: projectBindings,
      canManage: canManageProjectBindings,
    },
  ])

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

  function formatError(caughtError: unknown, fallback: string) {
    return caughtError instanceof ApiError ? caughtError.detail : fallback
  }

  function resolveRoleOption(roleKey: string) {
    return roleOptions.find((option) => option.key === roleKey)
  }

  function resetDraft(scope: ScopeKind) {
    if (scope === 'organization') {
      orgDraft = { ...defaultBindingDraft(), roleKey: 'org_member' }
      return
    }
    projectDraft = defaultBindingDraft()
  }

  function createPayload(scope: ScopeKind, draft: BindingDraft) {
    const subjectKey = draft.subjectKey.trim()
    if (!subjectKey) {
      throw new Error('Subject key is required.')
    }

    let expiresAt: string | undefined
    if (draft.expiresAtLocal.trim() !== '') {
      const parsed = new Date(draft.expiresAtLocal)
      if (Number.isNaN(parsed.getTime())) {
        throw new Error('Expiration must be a valid date and time.')
      }
      expiresAt = parsed.toISOString()
    }

    return {
      subject_kind: draft.subjectKind,
      subject_key: subjectKey,
      role_key:
        draft.roleKey.trim() || (scope === 'organization' ? 'org_member' : 'project_member'),
      expires_at: expiresAt,
    }
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
      const payload = createPayload(scope, draft)
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

  function formatTimestamp(value: string | undefined) {
    if (!value) {
      return 'Never'
    }
    const parsed = new Date(value)
    if (Number.isNaN(parsed.getTime())) {
      return value
    }
    return parsed.toLocaleString()
  }

  function bindingPlaceholder(subjectKind: SubjectKind) {
    return subjectKind === 'group' ? 'oidc:platform-admins' : 'user@example.com'
  }

  function scopeTitle(scope: ScopeKind) {
    return scope === 'organization' ? 'Organization RBAC' : 'Project RBAC'
  }

  function updateDraftSubjectKind(scope: ScopeKind, event: Event) {
    const nextValue = (event.currentTarget as HTMLSelectElement).value as SubjectKind
    if (scope === 'organization') {
      orgDraft.subjectKind = nextValue
      return
    }
    projectDraft.subjectKind = nextValue
  }

  function updateDraftSubjectKey(scope: ScopeKind, event: Event) {
    const nextValue = (event.currentTarget as HTMLInputElement).value
    if (scope === 'organization') {
      orgDraft.subjectKey = nextValue
      return
    }
    projectDraft.subjectKey = nextValue
  }

  function updateDraftRoleKey(scope: ScopeKind, event: Event) {
    const nextValue = (event.currentTarget as HTMLSelectElement).value
    if (scope === 'organization') {
      orgDraft.roleKey = nextValue
      return
    }
    projectDraft.roleKey = nextValue
  }

  function updateDraftExpiresAt(scope: ScopeKind, event: Event) {
    const nextValue = (event.currentTarget as HTMLInputElement).value
    if (scope === 'organization') {
      orgDraft.expiresAtLocal = nextValue
      return
    }
    projectDraft.expiresAtLocal = nextValue
  }
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <Shield class="text-muted-foreground size-4" />
    <h3 class="text-sm font-semibold">Human access and RBAC</h3>
  </div>

  <div class="bg-muted/30 grid gap-3 rounded-lg px-4 py-3 text-xs md:grid-cols-2 xl:grid-cols-4">
    <div>
      <div class="text-muted-foreground">Auth mode</div>
      <div class="text-foreground mt-1 font-medium uppercase">{authStore.authMode}</div>
    </div>
    <div>
      <div class="text-muted-foreground">Issuer</div>
      <div class="text-foreground mt-1 font-mono break-all">
        {authStore.issuerURL || 'Not configured'}
      </div>
    </div>
    <div>
      <div class="text-muted-foreground">Current user</div>
      <div class="text-foreground mt-1 font-medium">
        {authStore.user?.displayName || 'Anonymous'}
      </div>
      {#if authStore.user?.primaryEmail}
        <div class="text-muted-foreground">{authStore.user.primaryEmail}</div>
      {/if}
    </div>
    <div>
      <div class="text-muted-foreground">Session boundary</div>
      <div class="text-foreground mt-1">httpOnly cookie + CSRF header</div>
      <div class="text-muted-foreground">OIDC tokens stay server-side.</div>
    </div>
  </div>

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
    <div class="grid gap-4 xl:grid-cols-2">
      <div class="border-border bg-card space-y-3 rounded-lg border p-4">
        <div class="flex items-center gap-2">
          <Users class="text-muted-foreground size-4" />
          <h4 class="text-sm font-semibold">Identity</h4>
        </div>
        <div class="grid gap-3 text-xs sm:grid-cols-2">
          <div>
            <div class="text-muted-foreground">User ID</div>
            <div class="mt-1 font-mono break-all">{authStore.user?.id ?? ''}</div>
          </div>
          <div>
            <div class="text-muted-foreground">Groups</div>
            <div class="mt-1 flex flex-wrap gap-1">
              {#if currentGroups.length > 0}
                {#each currentGroups as group (group.issuer + ':' + group.group_key)}
                  <code class="bg-muted rounded px-1.5 py-0.5">
                    {group.group_name || group.group_key}
                  </code>
                {/each}
              {:else}
                <span class="text-muted-foreground">No synchronized groups</span>
              {/if}
            </div>
          </div>
        </div>
      </div>

      <div class="border-border bg-card space-y-3 rounded-lg border p-4">
        <div class="flex items-center gap-2">
          <LockKeyhole class="text-muted-foreground size-4" />
          <h4 class="text-sm font-semibold">Approval boundary</h4>
        </div>
        <p class="text-muted-foreground text-xs">
          RBAC decides whether a user can start an action. Approval policy stays reserved for future
          second-factor or approver requirements.
        </p>
        <div class="text-muted-foreground text-xs">
          Audit attribution stays on the human principal, including project conversation confirms.
        </div>
      </div>
    </div>

    <div class="grid gap-4 xl:grid-cols-2">
      <div class="border-border bg-card space-y-3 rounded-lg border p-4">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h4 class="text-sm font-semibold">Organization effective access</h4>
            <p class="text-muted-foreground text-xs">{appStore.currentOrg?.name ?? ''}</p>
          </div>
        </div>
        <div class="space-y-2 text-xs">
          <div>
            <div class="text-muted-foreground">Roles</div>
            <div class="mt-1 flex flex-wrap gap-1">
              {#if (orgPermissions?.roles.length ?? 0) > 0}
                {#each orgPermissions?.roles ?? [] as role (role)}
                  <code class="bg-muted rounded px-1.5 py-0.5">{role}</code>
                {/each}
              {:else}
                <span class="text-muted-foreground">No organization roles</span>
              {/if}
            </div>
          </div>
          <div>
            <div class="text-muted-foreground">Permissions</div>
            <div class="mt-1 flex flex-wrap gap-1">
              {#if (orgPermissions?.permissions.length ?? 0) > 0}
                {#each orgPermissions?.permissions ?? [] as permission (permission)}
                  <code class="bg-muted rounded px-1.5 py-0.5">{permission}</code>
                {/each}
              {:else}
                <span class="text-muted-foreground">No organization permissions</span>
              {/if}
            </div>
          </div>
        </div>
      </div>

      <div class="border-border bg-card space-y-3 rounded-lg border p-4">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h4 class="text-sm font-semibold">Project effective access</h4>
            <p class="text-muted-foreground text-xs">{appStore.currentProject?.name ?? ''}</p>
          </div>
        </div>
        <div class="space-y-2 text-xs">
          <div>
            <div class="text-muted-foreground">Roles</div>
            <div class="mt-1 flex flex-wrap gap-1">
              {#if (projectPermissions?.roles.length ?? 0) > 0}
                {#each projectPermissions?.roles ?? [] as role (role)}
                  <code class="bg-muted rounded px-1.5 py-0.5">{role}</code>
                {/each}
              {:else}
                <span class="text-muted-foreground">No project roles</span>
              {/if}
            </div>
          </div>
          <div>
            <div class="text-muted-foreground">Permissions</div>
            <div class="mt-1 flex flex-wrap gap-1">
              {#if (projectPermissions?.permissions.length ?? 0) > 0}
                {#each projectPermissions?.permissions ?? [] as permission (permission)}
                  <code class="bg-muted rounded px-1.5 py-0.5">{permission}</code>
                {/each}
              {:else}
                <span class="text-muted-foreground">No project permissions</span>
              {/if}
            </div>
          </div>
        </div>
      </div>
    </div>

    {#if error}
      <div class="text-destructive text-sm">{error}</div>
    {/if}

    {#each bindingSections as section (section.scope)}
      <div class="border-border bg-card space-y-4 rounded-lg border p-4">
        <div class="flex items-start justify-between gap-3">
          <div>
            <h4 class="text-sm font-semibold">{scopeTitle(section.scope)}</h4>
            <p class="text-muted-foreground text-xs">
              {section.scope === 'organization'
                ? 'Bindings here inherit into descendant projects.'
                : 'Project-scoped roles stack with direct and group bindings.'}
            </p>
          </div>
          <div class="text-muted-foreground text-xs">
            {section.canManage ? 'Editable' : 'Read only'}
          </div>
        </div>

        {#if section.canManage}
          <div class="grid gap-2 lg:grid-cols-[9rem_minmax(0,1fr)_13rem_13rem_auto]">
            <label class="space-y-1 text-xs">
              <span class="text-muted-foreground">Subject kind</span>
              <select
                class="border-input bg-background h-9 rounded-md border px-3 text-sm"
                value={section.scope === 'organization'
                  ? orgDraft.subjectKind
                  : projectDraft.subjectKind}
                onchange={(event) => updateDraftSubjectKind(section.scope, event)}
              >
                <option value="user">User</option>
                <option value="group">Group</option>
              </select>
            </label>

            <label class="space-y-1 text-xs">
              <span class="text-muted-foreground">Subject key</span>
              <Input
                value={section.scope === 'organization'
                  ? orgDraft.subjectKey
                  : projectDraft.subjectKey}
                oninput={(event) => updateDraftSubjectKey(section.scope, event)}
                placeholder={bindingPlaceholder(
                  section.scope === 'organization'
                    ? orgDraft.subjectKind
                    : projectDraft.subjectKind,
                )}
              />
            </label>

            <label class="space-y-1 text-xs">
              <span class="text-muted-foreground">Role</span>
              <select
                class="border-input bg-background h-9 rounded-md border px-3 text-sm"
                value={section.scope === 'organization' ? orgDraft.roleKey : projectDraft.roleKey}
                onchange={(event) => updateDraftRoleKey(section.scope, event)}
              >
                {#each roleOptions as roleOption (roleOption.key)}
                  <option value={roleOption.key}>{roleOption.label}</option>
                {/each}
              </select>
            </label>

            <label class="space-y-1 text-xs">
              <span class="text-muted-foreground">Expires at</span>
              <Input
                type="datetime-local"
                value={section.scope === 'organization'
                  ? orgDraft.expiresAtLocal
                  : projectDraft.expiresAtLocal}
                oninput={(event) => updateDraftExpiresAt(section.scope, event)}
              />
            </label>

            <div class="flex items-end">
              <Button
                disabled={mutationKey === `${section.scope}:create`}
                onclick={() => {
                  void handleCreateBinding(section.scope)
                }}
              >
                {mutationKey === `${section.scope}:create` ? 'Adding…' : 'Add binding'}
              </Button>
            </div>
          </div>

          <p class="text-muted-foreground text-xs">
            Use a user email or stable user id for direct grants. Group subject keys should match
            the synchronized OIDC group key.
          </p>
        {:else}
          <p class="text-muted-foreground text-xs">
            <code>rbac.manage</code> is required on this scope before bindings can be edited.
          </p>
        {/if}

        <div class="space-y-2">
          {#if section.bindings.length > 0}
            {#each section.bindings as binding (binding.id)}
              <div
                class="border-border bg-muted/20 flex flex-col gap-3 rounded-lg border px-4 py-3 text-xs lg:flex-row lg:items-start lg:justify-between"
              >
                <div class="space-y-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="font-medium">
                      {resolveRoleOption(binding.roleKey)?.label ?? binding.roleKey}
                    </span>
                    <code class="bg-background rounded px-1.5 py-0.5">
                      {binding.subjectKind}:{binding.subjectKey}
                    </code>
                  </div>
                  <div class="text-muted-foreground">
                    {resolveRoleOption(binding.roleKey)?.summary ?? 'Static builtin role'}
                  </div>
                  <div class="text-muted-foreground">
                    Granted by {binding.grantedBy} · created {formatTimestamp(binding.createdAt)}
                    {#if binding.expiresAt}
                      · expires {formatTimestamp(binding.expiresAt)}
                    {/if}
                  </div>
                </div>

                {#if section.canManage}
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={mutationKey === `${section.scope}:delete:${binding.id}`}
                    onclick={() => {
                      void handleDeleteBinding(section.scope, binding.id)
                    }}
                  >
                    {mutationKey === `${section.scope}:delete:${binding.id}`
                      ? 'Deleting…'
                      : 'Delete'}
                  </Button>
                {/if}
              </div>
            {/each}
          {:else}
            <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-3 text-sm">
              No bindings yet on this scope.
            </div>
          {/if}
        </div>
      </div>
    {/each}
  {/if}
</div>
