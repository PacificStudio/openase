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
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { ShieldCheck } from '@lucide/svelte'
  import OrganizationAdminRolesBindingsList from './organization-admin-roles-bindings-list.svelte'

  let { organizationId }: { organizationId: string } = $props()

  let loading = $state(false)
  let error = $state('')
  let mutationKey = $state('')
  let bindings = $state<RoleBinding[]>([])
  let permissions = $state<EffectivePermissionsResponse | null>(null)
  let draft = $state<BindingDraft>(defaultBindingDraftForScope())
  let dialogOpen = $state(false)

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
    if (!canManageBindings) return false
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
      dialogOpen = false
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

  const selectClass =
    'h-9 w-full rounded-md border border-input bg-background px-3 text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50'
</script>

<div class="space-y-4">
  <!-- Header -->
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div class="flex items-center gap-2">
      <ShieldCheck class="text-muted-foreground size-4" />
      <h2 class="text-sm font-semibold">Org role bindings</h2>
      <Badge variant="outline" class="text-xs">Inherited by projects</Badge>
    </div>
    {#if canManageBindings}
      <Button size="sm" variant="outline" onclick={() => (dialogOpen = true)}>Add binding</Button>
    {/if}
  </div>

  <!-- Your access status bar -->
  <div class="bg-muted/40 rounded-md px-4 py-2.5">
    <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
      <span class="text-muted-foreground text-xs font-medium">Your access</span>
      {#if permissions?.roles?.length}
        {#each permissions.roles as role (role)}
          <Badge variant="secondary" class="text-xs">{role}</Badge>
        {/each}
      {:else}
        <span class="text-muted-foreground text-xs">No effective org roles</span>
      {/if}
      {#if permissions?.groups?.length}
        <span class="text-muted-foreground text-xs">·</span>
        {#each permissions.groups as group (group.issuer + ':' + group.group_key)}
          <code class="bg-muted text-muted-foreground rounded px-1.5 py-0.5 text-xs">
            {group.group_name || group.group_key}
          </code>
        {/each}
      {/if}
      {#if !canManageBindings}
        <span class="text-muted-foreground text-xs">· Read only</span>
      {:else if !canManagePrivilegedRoles}
        <span class="text-muted-foreground text-xs"
          >· Owner approval required to change privileged bindings</span
        >
      {/if}
    </div>
  </div>

  <!-- Bindings list -->
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

<!-- Add binding dialog -->
<Dialog.Root bind:open={dialogOpen}>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <Dialog.Title>Add role binding</Dialog.Title>
      <Dialog.Description>
        Bind a user or group to an org role. Org owners can grant <code>org_owner</code> and
        <code>org_admin</code>; org admins can manage member-level bindings.
      </Dialog.Description>
    </Dialog.Header>

    <div class="grid gap-4 sm:grid-cols-2">
      <div class="space-y-1.5">
        <Label>Subject kind</Label>
        <select
          class={selectClass}
          value={draft.subjectKind}
          onchange={(event) => {
            draft = {
              ...draft,
              subjectKind: (event.currentTarget as HTMLSelectElement).value as 'user' | 'group',
            }
          }}
        >
          <option value="user">User</option>
          <option value="group">Group</option>
        </select>
      </div>

      <div class="space-y-1.5">
        <Label>Subject key</Label>
        <Input
          value={draft.subjectKey}
          placeholder={draft.subjectKind === 'group' ? 'oidc:platform-admins' : 'user@example.com'}
          oninput={(event) => {
            draft = { ...draft, subjectKey: (event.currentTarget as HTMLInputElement).value }
          }}
        />
      </div>

      <div class="space-y-1.5">
        <Label>Role</Label>
        <select
          class={selectClass}
          value={roleOptions.includes(draft.roleKey) ? draft.roleKey : roleOptions[0]}
          onchange={(event) => {
            draft = { ...draft, roleKey: (event.currentTarget as HTMLSelectElement).value }
          }}
        >
          {#each roleOptions as roleKey (roleKey)}
            <option value={roleKey}>{resolveRoleOption(roleKey)?.label ?? roleKey}</option>
          {/each}
        </select>
      </div>

      <div class="space-y-1.5">
        <Label>Expires at <span class="text-muted-foreground font-normal">(optional)</span></Label>
        <Input
          type="datetime-local"
          value={draft.expiresAtLocal}
          oninput={(event) => {
            draft = { ...draft, expiresAtLocal: (event.currentTarget as HTMLInputElement).value }
          }}
        />
      </div>
    </div>

    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={mutationKey === 'create'}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={handleCreateBinding} disabled={mutationKey === 'create'}>
        {mutationKey === 'create' ? 'Adding…' : 'Add binding'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
