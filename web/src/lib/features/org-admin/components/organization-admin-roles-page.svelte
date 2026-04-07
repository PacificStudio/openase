<script lang="ts">
  import {
    createBindingPayload,
    defaultBindingDraftForScope,
    resolveRoleOption,
    type BindingDraft,
  } from '$lib/features/org-admin/role-bindings'
  import {
    createOrganizationRoleBinding,
    deleteOrganizationRoleBinding,
    getEffectivePermissions,
    listOrganizationRoleBindings,
    type EffectivePermissionsResponse,
    type RoleBinding,
  } from '$lib/api/auth'
  import { ApiError } from '$lib/api/client'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import OrganizationAdminRolesBindingsList from './organization-admin-roles-bindings-list.svelte'

  let { organizationId }: { organizationId: string } = $props()

  let loading = $state(false)
  let error = $state('')
  let mutationKey = $state('')
  let bindings = $state<RoleBinding[]>([])
  let permissions = $state<EffectivePermissionsResponse | null>(null)
  let draft = $state<BindingDraft>(defaultBindingDraftForScope())

  const canManageBindings = $derived(permissions?.permissions.includes('rbac.manage') ?? false)
  const canManagePrivilegedRoles = $derived(
    permissions?.roles.includes('instance_admin') ||
      permissions?.roles.includes('org_owner') ||
      false,
  )
  const roleOptions = $derived(
    canManagePrivilegedRoles ? ['org_owner', 'org_admin', 'org_member'] : ['org_member'],
  )

  function canDeleteBinding(binding: RoleBinding) {
    if (!canManageBindings) {
      return false
    }
    return canManagePrivilegedRoles || binding.roleKey === 'org_member'
  }

  async function loadState() {
    if (!organizationId) {
      bindings = []
      permissions = null
      return
    }
    loading = true
    error = ''
    try {
      const [nextPermissions, nextBindings] = await Promise.all([
        getEffectivePermissions({ orgId: organizationId }),
        listOrganizationRoleBindings(organizationId),
      ])
      permissions = nextPermissions
      bindings = nextBindings
      const nextCanManagePrivilegedRoles =
        nextPermissions.roles.includes('instance_admin') ||
        nextPermissions.roles.includes('org_owner')
      if (!nextCanManagePrivilegedRoles && draft.roleKey !== 'org_member') {
        draft = { ...draft, roleKey: 'org_member' }
      }
    } catch (caughtError) {
      error =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load org role bindings.'
    } finally {
      loading = false
    }
  }

  async function handleCreateBinding() {
    mutationKey = 'create'
    error = ''
    try {
      const payload = createBindingPayload({
        ...draft,
        roleKey: roleOptions.includes(draft.roleKey) ? draft.roleKey : 'org_member',
      })
      await createOrganizationRoleBinding(organizationId, payload)
      draft = defaultBindingDraftForScope()
      await loadState()
      toastStore.success('Organization role binding added.')
    } catch (caughtError) {
      const message =
        caughtError instanceof Error ? caughtError.message : 'Failed to create org role binding.'
      error = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }

  async function handleDeleteBinding(binding: RoleBinding) {
    mutationKey = `delete:${binding.id}`
    error = ''
    try {
      await deleteOrganizationRoleBinding(organizationId, binding.id)
      await loadState()
      toastStore.success('Organization role binding removed.')
    } catch (caughtError) {
      const message =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete org role binding.'
      error = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }

  $effect(() => {
    void organizationId
    void loadState()
  })
</script>

<div class="space-y-6">
  <div class="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
    <div class="rounded-3xl border bg-white p-5 shadow-sm">
      <div class="flex items-center gap-2">
        <h2 class="text-lg font-semibold">Org-scoped role bindings</h2>
        <Badge variant="outline">Inherited by projects</Badge>
      </div>
      <p class="text-muted-foreground mt-2 text-sm leading-6">
        Bind users or synchronized groups to org roles. Every org role here flows into descendant
        project permission evaluation, but instance-wide `/admin` bindings stay separate and are
        never editable from this surface.
      </p>
      <div class="mt-4 grid gap-3 md:grid-cols-3">
        {#each ['org_owner', 'org_admin', 'org_member'] as roleKey (roleKey)}
          {@const role = resolveRoleOption(roleKey)}
          <div class="rounded-2xl border border-slate-200 p-4">
            <div class="font-medium">{role?.label ?? roleKey}</div>
            <div class="text-muted-foreground mt-2 text-sm leading-6">{role?.summary}</div>
          </div>
        {/each}
      </div>
    </div>

    <div class="rounded-3xl border bg-white p-5 shadow-sm">
      <h2 class="text-lg font-semibold">Identity and groups</h2>
      <div class="mt-4 space-y-4 text-sm">
        <div>
          <div class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Current roles</div>
          <div class="mt-2 flex flex-wrap gap-2">
            {#if permissions?.roles?.length}
              {#each permissions.roles as role (role)}
                <Badge variant="secondary">{role}</Badge>
              {/each}
            {:else}
              <span class="text-muted-foreground">No effective org roles</span>
            {/if}
          </div>
        </div>
        <div>
          <div class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Synced groups</div>
          <div class="mt-2 flex flex-wrap gap-2">
            {#if permissions?.groups?.length}
              {#each permissions.groups as group (group.issuer + ':' + group.group_key)}
                <code class="rounded-full bg-slate-100 px-3 py-1 text-xs">
                  {group.group_name || group.group_key}
                </code>
              {/each}
            {:else}
              <span class="text-muted-foreground">No synchronized groups</span>
            {/if}
          </div>
        </div>
        <div
          class="text-muted-foreground rounded-2xl border border-dashed px-3 py-3 text-xs leading-6"
        >
          Group bindings let org admins express durable access without editing each membership
          individually. Subject keys must match the synchronized OIDC group key that appears in
          effective permissions.
        </div>
      </div>
    </div>
  </div>

  <div class="rounded-3xl border bg-white p-5 shadow-sm">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h3 class="text-lg font-semibold">Manage bindings</h3>
        <p class="text-muted-foreground mt-1 text-sm">
          Org owners can grant `org_owner` and `org_admin`. Org admins can keep member-level
          bindings current.
        </p>
      </div>
      <Badge variant={canManageBindings ? 'secondary' : 'outline'}>
        {canManageBindings ? 'Editable' : 'Read only'}
      </Badge>
    </div>

    <div class="mt-5 grid gap-3 xl:grid-cols-[10rem_minmax(0,1fr)_12rem_12rem_auto]">
      <label class="space-y-1 text-xs">
        <span class="text-muted-foreground">Subject kind</span>
        <select
          class="border-input bg-background h-10 rounded-xl border px-3 text-sm"
          value={draft.subjectKind}
          onchange={(event) => {
            draft = {
              ...draft,
              subjectKind: (event.currentTarget as HTMLSelectElement).value as 'user' | 'group',
            }
          }}
          disabled={!canManageBindings}
        >
          <option value="user">User</option>
          <option value="group">Group</option>
        </select>
      </label>

      <label class="space-y-1 text-xs">
        <span class="text-muted-foreground">Subject key</span>
        <Input
          value={draft.subjectKey}
          placeholder={draft.subjectKind === 'group' ? 'oidc:platform-admins' : 'user@example.com'}
          oninput={(event) => {
            draft = { ...draft, subjectKey: (event.currentTarget as HTMLInputElement).value }
          }}
          disabled={!canManageBindings}
        />
      </label>

      <label class="space-y-1 text-xs">
        <span class="text-muted-foreground">Role</span>
        <select
          class="border-input bg-background h-10 rounded-xl border px-3 text-sm"
          value={roleOptions.includes(draft.roleKey) ? draft.roleKey : roleOptions[0]}
          onchange={(event) => {
            draft = { ...draft, roleKey: (event.currentTarget as HTMLSelectElement).value }
          }}
          disabled={!canManageBindings}
        >
          {#each roleOptions as roleKey (roleKey)}
            <option value={roleKey}>{resolveRoleOption(roleKey)?.label ?? roleKey}</option>
          {/each}
        </select>
      </label>

      <label class="space-y-1 text-xs">
        <span class="text-muted-foreground">Expires at</span>
        <Input
          type="datetime-local"
          value={draft.expiresAtLocal}
          oninput={(event) => {
            draft = { ...draft, expiresAtLocal: (event.currentTarget as HTMLInputElement).value }
          }}
          disabled={!canManageBindings}
        />
      </label>

      <div class="flex items-end">
        <Button
          onclick={handleCreateBinding}
          disabled={!canManageBindings || mutationKey === 'create'}
        >
          {mutationKey === 'create' ? 'Adding…' : 'Add binding'}
        </Button>
      </div>
    </div>
  </div>

  <OrganizationAdminRolesBindingsList
    {loading}
    {error}
    {bindings}
    {canManageBindings}
    {canDeleteBinding}
    {mutationKey}
    onDeleteBinding={handleDeleteBinding}
  />
</div>
