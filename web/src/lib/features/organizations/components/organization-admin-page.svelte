<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import type { EffectivePermissionsResponse, RoleBinding } from '$lib/api/auth'
  import {
    createOrganizationRoleBinding,
    deleteOrganizationRoleBinding,
    getEffectivePermissions,
    listOrganizationRoleBindings,
  } from '$lib/api/auth'
  import { authStore } from '$lib/stores/auth.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import SecuritySettingsHumanAuthAccessCard from '$lib/features/settings/components/security-settings-human-auth-access-card.svelte'
  import SecuritySettingsHumanAuthBindingSection from '$lib/features/settings/components/security-settings-human-auth-binding-section.svelte'
  import {
    createBindingPayload,
    defaultBindingDraftForScope,
    formatError,
    type BindingDraft,
  } from '$lib/features/settings/components/security-settings-human-auth.model'
  import OrganizationMembersSection from './organization-members-section.svelte'

  let { organizationId }: { organizationId: string } = $props()

  let loading = $state(false)
  let error = $state('')
  let permissions = $state<EffectivePermissionsResponse | null>(null)
  let bindings = $state<RoleBinding[]>([])
  let draft = $state<BindingDraft>(defaultBindingDraftForScope('organization'))
  let mutationKey = $state('')

  const orgName = $derived(appStore.currentOrg?.name ?? 'Organization')
  const canManageBindings = $derived(permissions?.permissions.includes('rbac.manage') ?? false)

  $effect(() => {
    if (authStore.authMode !== 'oidc' || !authStore.authenticated || !organizationId) {
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
          getEffectivePermissions({ orgId: organizationId }),
          listOrganizationRoleBindings(organizationId),
        ])
        if (cancelled) {
          return
        }
        permissions = nextPermissions
        bindings = nextBindings
      } catch (caughtError) {
        if (!cancelled) {
          error = formatError(caughtError, 'Failed to load organization admin state.')
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

  async function reloadOrganizationRBAC() {
    const [nextPermissions, nextBindings] = await Promise.all([
      getEffectivePermissions({ orgId: organizationId }),
      listOrganizationRoleBindings(organizationId),
    ])
    permissions = nextPermissions
    bindings = nextBindings
  }

  async function handleCreateBinding() {
    const key = 'organization:create'
    mutationKey = key
    error = ''

    try {
      await createOrganizationRoleBinding(
        organizationId,
        createBindingPayload('organization', draft),
      )
      await reloadOrganizationRBAC()
      draft = defaultBindingDraftForScope('organization')
      toastStore.success('Organization role binding added.')
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
    const key = `organization:delete:${bindingId}`
    mutationKey = key
    error = ''

    try {
      await deleteOrganizationRoleBinding(organizationId, bindingId)
      await reloadOrganizationRBAC()
      toastStore.success('Organization role binding deleted.')
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to delete role binding.')
      error = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }
</script>

<PageScaffold
  title="Organization Admin"
  description="Owner/admin membership lifecycle, invitations, and organization-scoped RBAC live here."
>
  <div class="space-y-6">
    <div class="flex flex-wrap items-center gap-2">
      <Badge variant="outline">/{'orgs'}/{organizationId}/admin</Badge>
      <Badge variant="secondary">{orgName}</Badge>
    </div>

    {#if authStore.authMode !== 'oidc'}
      <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-4 text-sm">
        Organization membership and invitation lifecycle is only active in OIDC mode. Configure or
        enable OIDC from <a href="/admin" class="text-foreground underline">/admin</a> first.
      </div>
    {:else if loading}
      <div class="space-y-3">
        <div class="bg-muted h-20 animate-pulse rounded-lg"></div>
        <div class="bg-muted h-32 animate-pulse rounded-lg"></div>
      </div>
    {:else}
      <SecuritySettingsHumanAuthAccessCard
        title="Organization effective access"
        subtitle={orgName}
        roles={permissions?.roles ?? []}
        permissions={permissions?.permissions ?? []}
        emptyRoles="No organization roles"
        emptyPermissions="No organization permissions"
      />

      {#if error}
        <div class="text-destructive text-sm">{error}</div>
      {/if}

      <SecuritySettingsHumanAuthBindingSection
        scope="organization"
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

      <OrganizationMembersSection {organizationId} />
    {/if}
  </div>
</PageScaffold>
