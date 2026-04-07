<script lang="ts">
  import type { EffectivePermissionsResponse, RoleBinding } from '$lib/api/auth'
  import {
    createProjectRoleBinding,
    deleteProjectRoleBinding,
    getEffectivePermissions,
    listProjectRoleBindings,
  } from '$lib/api/auth'
  import { authStore } from '$lib/stores/auth.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import SecuritySettingsHumanAuthAccessCard from './security-settings-human-auth-access-card.svelte'
  import SecuritySettingsHumanAuthBindingSection from './security-settings-human-auth-binding-section.svelte'
  import {
    createBindingPayload,
    defaultBindingDraftForScope,
    formatError,
    type BindingDraft,
  } from './security-settings-human-auth.model'

  let {
    projectId = '',
    projectName = '',
  }: {
    projectId?: string
    projectName?: string
  } = $props()

  let loading = $state(false)
  let error = $state('')
  let permissions = $state<EffectivePermissionsResponse | null>(null)
  let bindings = $state<RoleBinding[]>([])
  let draft = $state<BindingDraft>(defaultBindingDraftForScope('project'))
  let mutationKey = $state('')

  const canManageBindings = $derived(permissions?.permissions.includes('rbac.manage') ?? false)

  $effect(() => {
    if (authStore.authMode !== 'oidc' || !authStore.authenticated || !projectId) {
      loading = false
      error = ''
      permissions = null
      bindings = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''
      try {
        const [nextPermissions, nextBindings] = await Promise.all([
          getEffectivePermissions({ projectId }),
          listProjectRoleBindings(projectId),
        ])
        if (cancelled) {
          return
        }
        permissions = nextPermissions
        bindings = nextBindings
      } catch (caughtError) {
        if (!cancelled) {
          error = formatError(caughtError, 'Failed to load project RBAC state.')
        }
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

  async function reloadProjectRBAC() {
    const [nextPermissions, nextBindings] = await Promise.all([
      getEffectivePermissions({ projectId }),
      listProjectRoleBindings(projectId),
    ])
    permissions = nextPermissions
    bindings = nextBindings
  }

  async function handleCreateBinding() {
    const key = 'project:create'
    mutationKey = key
    error = ''

    try {
      await createProjectRoleBinding(projectId, createBindingPayload('project', draft))
      await reloadProjectRBAC()
      draft = defaultBindingDraftForScope('project')
      toastStore.success('Project role binding added.')
    } catch (caughtError) {
      const message =
        caughtError instanceof Error ? caughtError.message : 'Failed to create role binding.'
      error = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }

  async function handleDeleteBinding(bindingId: string) {
    const key = `project:delete:${bindingId}`
    mutationKey = key
    error = ''

    try {
      await deleteProjectRoleBinding(projectId, bindingId)
      await reloadProjectRBAC()
      toastStore.success('Project role binding deleted.')
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to delete role binding.')
      error = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }
</script>

<div class="space-y-4">
  <div>
    <h3 class="text-foreground text-sm font-semibold">Project access</h3>
    <p class="text-muted-foreground mt-1 text-sm">
      Keep only project-scoped RBAC here. Instance auth lives in `/admin`, and organization members
      and invitations live in org admin.
    </p>
  </div>

  {#if authStore.authMode !== 'oidc'}
    <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-4 text-sm">
      Disabled mode keeps project collaboration single-user. Enable OIDC from <a
        href="/admin"
        class="text-foreground underline">/admin</a
      > before you use project-scoped human RBAC.
    </div>
  {:else if loading}
    <div class="space-y-3">
      <div class="bg-muted h-16 animate-pulse rounded-lg"></div>
      <div class="bg-muted h-32 animate-pulse rounded-lg"></div>
    </div>
  {:else}
    <SecuritySettingsHumanAuthAccessCard
      title="Project effective access"
      subtitle={projectName}
      roles={permissions?.roles ?? []}
      permissions={permissions?.permissions ?? []}
      emptyRoles="No project roles"
      emptyPermissions="No project permissions"
    />

    {#if error}
      <div class="text-destructive text-sm">{error}</div>
    {/if}

    <SecuritySettingsHumanAuthBindingSection
      scope="project"
      {bindings}
      canManage={canManageBindings}
      {draft}
      {mutationKey}
      onSubjectKind={(_scope, value) => (draft = { ...draft, subjectKind: value })}
      onSubjectKey={(_scope, value) => (draft = { ...draft, subjectKey: value })}
      onRoleKey={(_scope, value) => (draft = { ...draft, roleKey: value })}
      onExpiresAt={(_scope, value) => (draft = { ...draft, expiresAtLocal: value })}
      onCreate={() => void handleCreateBinding()}
      onDelete={(_scope, bindingId) => void handleDeleteBinding(bindingId)}
    />
  {/if}
</div>
