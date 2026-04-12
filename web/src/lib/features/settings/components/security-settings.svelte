<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { SecuritySettingsResponse } from '$lib/api/contracts'
  import {
    deleteGitHubOutboundCredential,
    getSecuritySettings,
    importGitHubOutboundCredentialFromGHCLI,
    retestGitHubOutboundCredential,
    saveGitHubOutboundCredential,
  } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { TranslationKey } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Separator } from '$ui/separator'

  import GitHubOutboundCredentialsPanel from './security-settings-github-outbound-credentials.svelte'
  import ProjectAIPlatformAccessPanel from './project-ai-platform-access-panel.svelte'
  import ProjectScopedSecretsPanel from './project-scoped-secrets-panel.svelte'
  import SecurityPlatformDetails from './security-settings-platform-details.svelte'
  import SecuritySettingsSecretBindingsSection from './security-settings-secret-bindings-section.svelte'
  import { normalizeSecuritySettings } from '../security-settings'

  type Security = SecuritySettingsResponse['security']

  let security = $state<Security | null>(null)
  let loading = $state(false)
  let error = $state('')
  let actionKey = $state('')
  let manualToken = $state('')

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      security = null
      error = ''
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''
      try {
        const payload = await getSecuritySettings(projectId)
        if (cancelled) return
        security = normalizeSecuritySettings(payload.security)
      } catch (caughtError) {
        if (cancelled) return
        security = null
        error = formatError(caughtError, 'settings.security.errors.load')
      } finally {
        if (!cancelled) loading = false
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  })

  function formatError(caughtError: unknown, fallbackKey: TranslationKey) {
    return caughtError instanceof ApiError ? caughtError.detail : i18nStore.t(fallbackKey)
  }

  async function mutate(action: 'save' | 'import' | 'retest' | 'delete') {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    actionKey = action
    error = ''

    try {
      let payload: SecuritySettingsResponse
      if (action === 'save') {
        const token = manualToken.trim()
        if (!token) {
          toastStore.error(i18nStore.t('settings.security.errors.githubTokenRequired'))
          return
        }
        payload = await saveGitHubOutboundCredential(projectId, { token })
        manualToken = ''
        toastStore.success(i18nStore.t('settings.security.notifications.githubCredentialSaved'))
      } else if (action === 'import') {
        payload = await importGitHubOutboundCredentialFromGHCLI(projectId)
        toastStore.success(i18nStore.t('settings.security.notifications.githubCredentialImported'))
      } else if (action === 'retest') {
        payload = await retestGitHubOutboundCredential(projectId)
        toastStore.success(i18nStore.t('settings.security.notifications.githubCredentialRetested'))
      } else {
        payload = await deleteGitHubOutboundCredential(projectId)
        manualToken = ''
        toastStore.success(i18nStore.t('settings.security.notifications.githubCredentialDeleted'))
      }
      security = normalizeSecuritySettings(payload.security)
    } catch (caughtError) {
      const message = formatError(
        caughtError,
        'settings.security.errors.githubCredentialUpdateFailed',
      )
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">
      {i18nStore.t('settings.security.heading')}
    </h2>
    <p class="text-muted-foreground mt-1 text-sm">
      {i18nStore.t('settings.security.description')}
    </p>
  </div>

  {#if loading}
    <div class="space-y-6">
      <div class="space-y-3">
        <div class="bg-muted h-4 w-44 animate-pulse rounded"></div>
        <div class="border-border bg-card rounded-lg border p-4">
          <div class="flex items-center gap-3">
            <div class="bg-muted size-8 shrink-0 animate-pulse rounded-lg"></div>
            <div class="flex-1 space-y-1.5">
              <div class="bg-muted h-4 w-28 animate-pulse rounded"></div>
              <div class="bg-muted h-3 w-40 animate-pulse rounded"></div>
            </div>
            <div class="bg-muted h-7 w-20 animate-pulse rounded-md"></div>
          </div>
        </div>
      </div>
      <div class="bg-border h-px"></div>
      <div class="space-y-3">
        <div class="bg-muted h-4 w-32 animate-pulse rounded"></div>
        <div class="grid grid-cols-2 gap-3">
          {#each { length: 4 } as _}
            <div class="flex items-center gap-2">
              <div class="bg-muted h-3 w-20 animate-pulse rounded"></div>
              <div class="bg-muted h-3 w-24 animate-pulse rounded"></div>
            </div>
          {/each}
        </div>
      </div>
    </div>
  {:else if error}
    <div class="text-destructive text-sm">{error}</div>
  {:else if security}
    <Separator />

    <ProjectScopedSecretsPanel
      projectId={appStore.currentProject?.id ?? ''}
      organizationId={appStore.currentOrg?.id ?? ''}
    />

    <Separator />

    <GitHubOutboundCredentialsPanel
      {security}
      {actionKey}
      {manualToken}
      onAction={mutate}
      onManualTokenChange={(value) => (manualToken = value)}
    />

    <Separator />

    <SecuritySettingsSecretBindingsSection />

    <Separator />

    <ProjectAIPlatformAccessPanel {security} />

    <Separator />

    <SecurityPlatformDetails {security} />
  {/if}
</div>
