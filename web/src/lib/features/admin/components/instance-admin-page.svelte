<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import type { AdminSecuritySettingsResponse } from '$lib/api/contracts'
  import {
    createInstanceRoleBinding,
    deleteInstanceRoleBinding,
    getEffectivePermissions,
    listInstanceRoleBindings,
    type EffectivePermissionsResponse,
    type RoleBinding,
  } from '$lib/api/auth'
  import { getAdminSecuritySettings } from '$lib/api/openase'
  import { authStore } from '$lib/stores/auth.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Separator } from '$ui/separator'
  import { Badge } from '$ui/badge'
  import SecuritySettingsHumanAuthAccessCard from '$lib/features/settings/components/security-settings-human-auth-access-card.svelte'
  import SecuritySettingsHumanAuthBindingSection from '$lib/features/settings/components/security-settings-human-auth-binding-section.svelte'
  import SecuritySettingsHumanAuthGuideLinks from '$lib/features/settings/components/security-settings-human-auth-guide-links.svelte'
  import SecuritySettingsHumanAuthSessions from '$lib/features/settings/components/security-settings-human-auth-sessions.svelte'
  import SecuritySettingsUserDirectory from '$lib/features/settings/components/security-settings-user-directory.svelte'
  import {
    createBindingPayload,
    defaultBindingDraftForScope,
    formatError,
    type BindingDraft,
  } from '$lib/features/settings/components/security-settings-human-auth.model'
  import InstanceAdminAuthSetupPanel from './instance-admin-auth-setup-panel.svelte'

  let settings = $state<AdminSecuritySettingsResponse['settings'] | null>(null)
  let loading = $state(false)
  let error = $state('')
  let rbacLoading = $state(false)
  let rbacError = $state('')
  let mutationKey = $state('')
  let instancePermissions = $state<EffectivePermissionsResponse | null>(null)
  let instanceBindings = $state<RoleBinding[]>([])
  let instanceDraft = $state<BindingDraft>(defaultBindingDraftForScope('instance'))

  const canManageInstanceBindings = $derived(
    instancePermissions?.permissions.includes('rbac.manage') ?? false,
  )
  const canReadUserDirectory = $derived(
    instancePermissions?.permissions.includes('security.read') ?? false,
  )
  const canManageUserDirectory = $derived(
    instancePermissions?.permissions.includes('security.manage') ?? false,
  )

  $effect(() => {
    let cancelled = false

    const load = async () => {
      loading = true
      error = ''
      try {
        const payload = await getAdminSecuritySettings()
        if (!cancelled) {
          settings = payload.settings
        }
      } catch (caughtError) {
        if (!cancelled) {
          settings = null
          error = formatError(caughtError, 'Failed to load instance admin settings.')
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

  $effect(() => {
    const authMode = authStore.authMode
    const authenticated = authStore.authenticated

    if (authMode !== 'oidc' || !authenticated) {
      rbacLoading = false
      rbacError = ''
      instancePermissions = null
      instanceBindings = []
      return
    }

    let cancelled = false

    const load = async () => {
      rbacLoading = true
      rbacError = ''
      try {
        const [permissions, bindings] = await Promise.all([
          getEffectivePermissions({}),
          listInstanceRoleBindings(),
        ])
        if (cancelled) {
          return
        }
        instancePermissions = permissions
        instanceBindings = bindings
      } catch (caughtError) {
        if (!cancelled) {
          rbacError = formatError(caughtError, 'Failed to load instance RBAC state.')
        }
      } finally {
        if (!cancelled) {
          rbacLoading = false
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  })

  async function reloadInstanceRBAC() {
    const [permissions, bindings] = await Promise.all([
      getEffectivePermissions({}),
      listInstanceRoleBindings(),
    ])
    instancePermissions = permissions
    instanceBindings = bindings
  }

  async function handleCreateBinding() {
    const key = 'instance:create'
    mutationKey = key
    rbacError = ''

    try {
      await createInstanceRoleBinding(createBindingPayload('instance', instanceDraft))
      await reloadInstanceRBAC()
      instanceDraft = defaultBindingDraftForScope('instance')
      toastStore.success('Instance role binding added.')
    } catch (caughtError) {
      const message =
        caughtError instanceof Error ? caughtError.message : 'Failed to create role binding.'
      rbacError = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }

  async function handleDeleteBinding(bindingId: string) {
    const key = `instance:delete:${bindingId}`
    mutationKey = key
    rbacError = ''

    try {
      await deleteInstanceRoleBinding(bindingId)
      await reloadInstanceRBAC()
      toastStore.success('Instance role binding deleted.')
    } catch (caughtError) {
      const message = formatError(caughtError, 'Failed to delete role binding.')
      rbacError = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }
</script>

<PageScaffold
  title="Instance Admin"
  description="Global authentication, bootstrap access, instance RBAC, sessions, and cached users live here."
>
  <div class="space-y-6">
    {#if loading}
      <div class="space-y-3">
        <div class="bg-muted h-24 animate-pulse rounded-lg"></div>
        <div class="bg-muted h-64 animate-pulse rounded-lg"></div>
      </div>
    {:else if error}
      <div class="text-destructive rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm">
        {error}
      </div>
    {:else if settings}
      <div class="flex flex-wrap items-center gap-2">
        <Badge variant="outline">/{'admin'}</Badge>
        <Badge variant={settings.auth.active_mode === 'oidc' ? 'default' : 'secondary'}>
          Active: {settings.auth.active_mode}
        </Badge>
        <Badge variant="outline">Configured: {settings.auth.configured_mode}</Badge>
      </div>

      <InstanceAdminAuthSetupPanel
        auth={settings.auth}
        onSettingsChange={(nextSettings) => (settings = nextSettings)}
      />

      <SecuritySettingsHumanAuthGuideLinks docs={settings.auth.docs} />

      <Separator />

      {#if authStore.authMode !== 'oidc'}
        <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-4 text-sm">
          Disabled mode keeps the local bootstrap operator in control. Enable OIDC here before you
          expect real instance RBAC, session governance, user directory, or org membership lifecycle
          tooling.
        </div>
      {:else if rbacLoading}
        <div class="space-y-3">
          <div class="bg-muted h-20 animate-pulse rounded-lg"></div>
          <div class="bg-muted h-32 animate-pulse rounded-lg"></div>
        </div>
      {:else}
        <SecuritySettingsHumanAuthAccessCard
          title="Instance effective access"
          subtitle="Global control plane"
          roles={instancePermissions?.roles ?? []}
          permissions={instancePermissions?.permissions ?? []}
          emptyRoles="No instance roles"
          emptyPermissions="No instance permissions"
        />

        {#if rbacError}
          <div class="text-destructive text-sm">{rbacError}</div>
        {/if}

        <SecuritySettingsHumanAuthSessions />

        <SecuritySettingsUserDirectory
          canRead={canReadUserDirectory}
          canManage={canManageUserDirectory}
        />

        <SecuritySettingsHumanAuthBindingSection
          scope="instance"
          bindings={instanceBindings}
          canManage={canManageInstanceBindings}
          draft={instanceDraft}
          {mutationKey}
          onSubjectKind={(_scope, value) =>
            (instanceDraft = { ...instanceDraft, subjectKind: value })}
          onSubjectKey={(_scope, value) =>
            (instanceDraft = { ...instanceDraft, subjectKey: value })}
          onRoleKey={(_scope, value) => (instanceDraft = { ...instanceDraft, roleKey: value })}
          onExpiresAt={(_scope, value) =>
            (instanceDraft = { ...instanceDraft, expiresAtLocal: value })}
          onCreate={() => void handleCreateBinding()}
          onDelete={(_scope, bindingId) => void handleDeleteBinding(bindingId)}
        />
      {/if}
    {/if}
  </div>
</PageScaffold>
