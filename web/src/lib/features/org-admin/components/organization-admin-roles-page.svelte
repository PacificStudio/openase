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
  import { i18nStore } from '$lib/i18n/store.svelte'

  let { organizationId }: { organizationId: string } = $props()

  let loading = $state(false)
  let error = $state('')
  let mutationKey = $state('')
  let bindings = $state<RoleBinding[]>([])
  let permissions = $state<EffectivePermissionsResponse | null>(null)
  let draft = $state<BindingDraft>(defaultBindingDraftForScope())
  let dialogOpen = $state(false)
  const t = i18nStore.t

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
      error = t('orgAdmin.roles.errors.load')
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
      toastStore.success(
        t('orgAdmin.roles.toast.bindingAdded'),
      )
    } catch (caughtError) {
      const message =
        caughtError instanceof Error ? caughtError.message : t('orgAdmin.roles.errors.create')
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
      toastStore.success(
        t('orgAdmin.roles.toast.bindingRemoved'),
      )
    } catch (caughtError) {
      const message =
        caughtError instanceof ApiError ? caughtError.detail : t('orgAdmin.roles.errors.delete')
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
      <h2 class="text-sm font-semibold">
        {t('orgAdmin.roles.page.heading')}
      </h2>
      <Badge variant="outline" class="text-xs">
        {t('orgAdmin.roles.page.badge.inherited')}
      </Badge>
    </div>
    {#if canManageBindings}
      <Button
        size="sm"
        variant="outline"
        onclick={() => (dialogOpen = true)}
      >
        {t('orgAdmin.roles.page.actions.addBinding')}
      </Button>
    {/if}
  </div>

  <!-- Your access status bar -->
      <div class="bg-muted/40 rounded-md px-4 py-2.5">
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span class="text-muted-foreground text-xs font-medium">
              {t('orgAdmin.roles.page.access.yourAccess')}
            </span>
          {#if permissions?.roles?.length}
            {#each permissions.roles as role (role)}
              <Badge variant="secondary" class="text-xs">{role}</Badge>
            {/each}
          {:else}
             <span class="text-muted-foreground text-xs">
                {t('orgAdmin.roles.page.access.noRoles')}
              </span>
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
           <span class="text-muted-foreground text-xs">
              · {t('orgAdmin.roles.page.access.readOnly')}
            </span>
          {:else if !canManagePrivilegedRoles}
            <span class="text-muted-foreground text-xs">
              · {t('orgAdmin.roles.page.access.ownerApproval')}
            </span>
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
      <Dialog.Title>
        {t('orgAdmin.roles.dialog.title')}
      </Dialog.Title>
      <Dialog.Description>
        {t('orgAdmin.roles.dialog.description.prefix')}
        <code>org_owner</code>
        {t('orgAdmin.roles.dialog.description.middle')}
        <code>org_admin</code>
        {t('orgAdmin.roles.dialog.description.suffix')}
      </Dialog.Description>
    </Dialog.Header>

    <div class="grid gap-4 sm:grid-cols-2">
      <div class="space-y-1.5">
        <Label>
            {t('orgAdmin.roles.dialog.labels.subjectKind')}
        </Label>
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
          <option value="user">
          {t('orgAdmin.roles.dialog.options.user')}
          </option>
          <option value="group">
          {t('orgAdmin.roles.dialog.options.group')}
          </option>
        </select>
      </div>

      <div class="space-y-1.5">
        <Label>
          {t('orgAdmin.roles.dialog.labels.subjectKey')}
        </Label>
        <Input
          value={draft.subjectKey}
          placeholder={
            draft.subjectKind === 'group'
              ? t('orgAdmin.roles.dialog.placeholders.group')
              : t('orgAdmin.roles.dialog.placeholders.user')
          }
          oninput={(event) => {
            draft = { ...draft, subjectKey: (event.currentTarget as HTMLInputElement).value }
          }}
        />
      </div>

      <div class="space-y-1.5">
        <Label>
          {t('orgAdmin.roles.dialog.labels.role')}
        </Label>
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
        <Label>
          {t('orgAdmin.roles.dialog.labels.expiresAt')}{' '}
          <span class="text-muted-foreground font-normal">
            ({t('orgAdmin.roles.dialog.labels.optional')})
          </span>
        </Label>
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
          <Button
            variant="outline"
            {...props}
            disabled={mutationKey === 'create'}
          >
            {t('orgAdmin.roles.dialog.actions.cancel')}
          </Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={handleCreateBinding} disabled={mutationKey === 'create'}>
        {mutationKey === 'create'
          ? t('orgAdmin.roles.dialog.actions.adding')
          : t('orgAdmin.roles.dialog.actions.add')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
