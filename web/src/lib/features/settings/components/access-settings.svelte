<script lang="ts">
  import {
    getEffectivePermissions,
    listProjectRoleBindings,
    type EffectivePermissionsResponse,
    type RoleBinding,
  } from '$lib/api/auth'
  import { ApiError } from '$lib/api/client'
  import { appStore } from '$lib/stores/app.svelte'
  import { authStore } from '$lib/stores/auth.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Badge } from '$ui/badge'
  import * as Collapsible from '$ui/collapsible'
  import { Separator } from '$ui/separator'
  import { ChevronRight, Users } from '@lucide/svelte'
  import AccessSettingsDisabledCard from './access-settings-disabled-card.svelte'
  import SecuritySettingsHumanAuthAccessCard from './security-settings-human-auth-access-card.svelte'
  import SecuritySettingsHumanAuthBindingSection from './security-settings-human-auth-binding-section.svelte'
  import SecuritySettingsHumanAuthSignInHint from './security-settings-human-auth-sign-in-hint.svelte'
  import {
    createRoleBindingForScope,
    deleteRoleBindingForScope,
  } from './security-settings-human-auth.data'
  import {
    defaultBindingDraftForScope,
    formatError,
    type BindingDraft,
  } from './security-settings-human-auth.model'
  let accessLoading = $state(false)
  let accessError = $state('')
  let mutationKey = $state('')
  let projectPermissions = $state<EffectivePermissionsResponse | null>(null)
  let projectBindings = $state<RoleBinding[]>([])
  let projectDraft = $state<BindingDraft>(defaultBindingDraftForScope('project'))
  let detailsOpen = $state(false)

  const currentOrgId = $derived(appStore.currentOrg?.id ?? '')
  const currentProjectId = $derived(appStore.currentProject?.id ?? '')
  const currentProjectName = $derived(appStore.currentProject?.name ?? '')
  const currentGroups = $derived(projectPermissions?.groups ?? [])
  const canManageProjectBindings = $derived(
    projectPermissions?.permissions.includes('rbac.manage') ?? false,
  )

  async function loadProjectAccess(projectId: string) {
    const [nextPermissions, nextBindings] = await Promise.all([
      getEffectivePermissions({ projectId }),
      listProjectRoleBindings(projectId),
    ])
    projectPermissions = nextPermissions
    projectBindings = nextBindings
  }

  $effect(() => {
    const projectId = currentProjectId
    if (!authStore.usesOIDC || !authStore.authenticated || !projectId) {
      accessLoading = false
      accessError = ''
      projectPermissions = null
      projectBindings = []
      return
    }

    let cancelled = false

    const load = async () => {
      accessLoading = true
      accessError = ''
      try {
        await loadProjectAccess(projectId)
      } catch (caughtError) {
        if (cancelled) {
          return
        }
        accessError = formatError(
          caughtError,
          i18nStore.t('settings.access.errors.loadProjectAccess'),
        )
      } finally {
        if (!cancelled) {
          accessLoading = false
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  })

  function patchDraft(patch: Partial<BindingDraft>) {
    projectDraft = { ...projectDraft, ...patch }
  }

  async function handleCreateBinding() {
    const orgId = currentOrgId
    const projectId = currentProjectId
    if (!projectId) {
      return
    }

    mutationKey = 'project:create'
    accessError = ''
    try {
      await createRoleBindingForScope('project', orgId, projectId, projectDraft)
      await loadProjectAccess(projectId)
      projectDraft = defaultBindingDraftForScope('project')
      toastStore.success(i18nStore.t('settings.access.messages.bindingAdded'))
    } catch (caughtError) {
      const message =
        caughtError instanceof Error
          ? caughtError.message
          : i18nStore.t('settings.access.errors.createBinding')
      accessError = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }

  async function handleDeleteBinding(bindingId: string) {
    const orgId = currentOrgId
    const projectId = currentProjectId
    if (!projectId) {
      return
    }

    mutationKey = `project:delete:${bindingId}`
    accessError = ''
    try {
      await deleteRoleBindingForScope('project', orgId, projectId, bindingId)
      await loadProjectAccess(projectId)
      toastStore.success(i18nStore.t('settings.access.messages.bindingRemoved'))
    } catch (caughtError) {
      const message =
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('settings.access.errors.deleteBinding')
      accessError = message
      toastStore.error(message)
    } finally {
      mutationKey = ''
    }
  }
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">
      {i18nStore.t('settings.access.heading')}
    </h2>
    <p class="text-muted-foreground mt-1 text-sm">
      {i18nStore.t('settings.access.description')}
    </p>
  </div>

  {#if authStore.usesLocalBootstrap}
    <AccessSettingsDisabledCard />
  {:else if !authStore.authenticated}
    <SecuritySettingsHumanAuthSignInHint />
  {:else if accessLoading}
    <div class="space-y-4">
      <div class="bg-muted h-10 w-48 animate-pulse rounded-lg"></div>
      <div class="bg-muted h-32 animate-pulse rounded-lg"></div>
    </div>
  {:else}
    {#if accessError}
      <div class="text-destructive text-sm">{accessError}</div>
    {/if}

    <SecuritySettingsHumanAuthBindingSection
      scope="project"
      bindings={projectBindings}
      canManage={canManageProjectBindings}
      draft={projectDraft}
      {mutationKey}
      onSubjectKind={(_, value) => patchDraft({ subjectKind: value })}
      onSubjectKey={(_, value) => patchDraft({ subjectKey: value })}
      onRoleKey={(_, value) => patchDraft({ roleKey: value })}
      onExpiresAt={(_, value) => patchDraft({ expiresAtLocal: value })}
      onCreate={() => void handleCreateBinding()}
      onDelete={(_, bindingId) => void handleDeleteBinding(bindingId)}
    />

    <Separator />

    <Collapsible.Root bind:open={detailsOpen}>
      <Collapsible.Trigger>
        {#snippet child({ props })}
          <button
            {...props}
            type="button"
            class="text-muted-foreground hover:text-foreground flex w-full items-center gap-2 text-sm transition-colors"
          >
            <ChevronRight class="size-4 transition-transform {detailsOpen ? 'rotate-90' : ''}" />
            {i18nStore.t('settings.access.buttons.showIdentity')}
          </button>
        {/snippet}
      </Collapsible.Trigger>
      <Collapsible.Content>
        <div class="mt-4 space-y-4">
          <SecuritySettingsHumanAuthAccessCard
            title={i18nStore.t('settings.security.humanAuth.accessCardTitles.project')}
            subtitle={currentProjectName}
            roles={projectPermissions?.roles ?? []}
            permissions={projectPermissions?.permissions ?? []}
            emptyRoles={i18nStore.t('settings.security.humanAuth.messages.noProjectRoles')}
            emptyPermissions={i18nStore.t(
              'settings.security.humanAuth.messages.noProjectPermissions',
            )}
          />

          <div class="border-border bg-card space-y-3 rounded-lg border p-4">
            <div class="flex items-center gap-2">
              <Users class="text-muted-foreground size-4" />
              <div class="text-sm font-semibold">
                {i18nStore.t('settings.access.headings.currentPrincipal')}
              </div>
            </div>
            <div class="grid gap-4 text-sm sm:grid-cols-2">
              <div>
                <div class="text-muted-foreground text-xs font-medium">
                  {i18nStore.t('settings.access.labels.principal')}
                </div>
                <div class="mt-1.5 font-medium">
                  {authStore.user?.displayName ||
                    authStore.user?.primaryEmail ||
                    i18nStore.t('settings.access.fallbacks.authenticatedUser')}
                </div>
                <div class="text-muted-foreground mt-0.5 text-xs">
                  {authStore.user?.primaryEmail ||
                    authStore.user?.id ||
                    i18nStore.t('settings.access.fallbacks.oidcSession')}
                </div>
              </div>
              <div>
                <div class="text-muted-foreground text-xs font-medium">
                  {i18nStore.t('settings.access.labels.syncedGroups')}
                </div>
                <div class="mt-1.5 flex flex-wrap gap-1.5">
                  {#if currentGroups.length > 0}
                    {#each currentGroups as group (group.issuer + ':' + group.group_key)}
                      <Badge variant="outline">{group.group_name || group.group_key}</Badge>
                    {/each}
                  {:else}
                    <span class="text-muted-foreground text-xs">
                      {i18nStore.t('settings.access.messages.noSyncedGroups')}
                    </span>
                  {/if}
                </div>
              </div>
            </div>
          </div>
        </div>
      </Collapsible.Content>
    </Collapsible.Root>

    <p class="text-muted-foreground text-xs">
      {i18nStore.t('settings.access.description.bottom')}
    </p>
  {/if}
</div>
