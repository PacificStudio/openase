<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import { ApiError } from '$lib/api/client'
  import type {
    AdminAuthModeTransitionResponse,
    OIDCDraftTestResponse,
    SecurityAuthSettings,
  } from '$lib/api/contracts'
  import {
    createOIDCFormState,
    oidcDraftFormFromAuth,
    oidcDraftPayloadFromForm,
    oidcDraftSignature,
    type OIDCFormState,
  } from '$lib/features/auth'
  import {
    disableAdminAuth,
    enableAdminOIDC,
    getAdminAuth,
    saveAdminOIDCDraft,
    testAdminOIDCDraft,
  } from '$lib/api/openase'
  import { authStore } from '$lib/stores/auth.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import AdminAuthDiagnostics from './admin-auth-diagnostics.svelte'
  import AdminAuthForm from './admin-auth-form.svelte'
  import AdminAuthOverview from './admin-auth-overview.svelte'
  import AdminAuthRuntimeDetails from './admin-auth-runtime-details.svelte'

  let loading = $state(false)
  let error = $state('')
  let errorCode = $state('')
  let actionKey = $state('')
  let auth = $state<SecurityAuthSettings | null>(null)
  let transition = $state<AdminAuthModeTransitionResponse['transition'] | null>(null)
  let lastDraftSignature = $state('')
  let oidcForm = $state<OIDCFormState>(createOIDCFormState())

  function syncForm(nextAuth: SecurityAuthSettings) {
    const nextSignature = oidcDraftSignature(nextAuth)
    if (nextSignature === lastDraftSignature) {
      return
    }
    oidcForm = oidcDraftFormFromAuth(nextAuth)
    lastDraftSignature = nextSignature
  }

  async function refreshAuth() {
    loading = true
    error = ''
    try {
      const payload = await getAdminAuth()
      auth = payload.auth
      syncForm(payload.auth)
    } catch (caughtError) {
      auth = null
      error =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load admin auth state.'
    } finally {
      loading = false
    }
  }

  async function runAction(
    key: 'save' | 'test' | 'enable' | 'disable',
    runner: () => Promise<void>,
    failureFallback: string,
  ) {
    actionKey = key
    error = ''
    errorCode = ''
    try {
      await runner()
    } catch (caughtError) {
      if (caughtError instanceof ApiError) {
        error = caughtError.detail
        errorCode = caughtError.code ?? ''
      } else {
        error = failureFallback
      }
      toastStore.error(error)
    } finally {
      actionKey = ''
    }
  }

  function applyValidationResult(result: OIDCDraftTestResponse) {
    if (!auth) {
      return
    }
    auth = {
      ...auth,
      last_validation: {
        status: result.status,
        message: result.message,
        checked_at: new Date().toISOString(),
        issuer_url: result.issuer_url,
        authorization_endpoint: result.authorization_endpoint,
        token_endpoint: result.token_endpoint,
        redirect_url: result.redirect_url,
        warnings: result.warnings,
      },
    }
  }

  async function handleSave() {
    await runAction(
      'save',
      async () => {
        const payload = await saveAdminOIDCDraft(oidcDraftPayloadFromForm(oidcForm))
        auth = payload.auth
        syncForm(payload.auth)
        transition = null
        toastStore.success('Draft saved.')
      },
      'Failed to save draft.',
    )
  }

  async function handleTest() {
    await runAction(
      'test',
      async () => {
        const payload = await testAdminOIDCDraft(oidcDraftPayloadFromForm(oidcForm))
        applyValidationResult(payload)
        transition = null
        toastStore.success(
          payload.issuer_url ? `Validation passed — ${payload.issuer_url}` : 'Validation passed.',
        )
      },
      'Validation failed.',
    )
    if (error) {
      await refreshAuth()
    }
  }

  async function handleEnable() {
    await runAction(
      'enable',
      async () => {
        const payload = await enableAdminOIDC(oidcDraftPayloadFromForm(oidcForm))
        auth = payload.auth
        syncForm(payload.auth)
        transition = payload.transition
        toastStore.success('OIDC activated.')
      },
      'Failed to activate OIDC.',
    )
    if (error) {
      await refreshAuth()
    }
  }

  async function handleDisable() {
    await runAction(
      'disable',
      async () => {
        const payload = await disableAdminAuth()
        auth = payload.auth
        syncForm(payload.auth)
        transition = payload.transition
        toastStore.success(
          'OIDC is now inactive. Use local bootstrap until you are ready to retry rollout.',
        )
      },
      'Failed to switch the instance back to local bootstrap access.',
    )
  }

  $effect(() => {
    void refreshAuth()
  })
</script>

<PageScaffold
  title="Admin Auth"
  description="Instance browser authentication and OIDC provider settings."
>
  {#if loading}
    <div class="space-y-4">
      <div class="bg-muted h-32 animate-pulse rounded-2xl"></div>
      <div class="bg-muted h-64 animate-pulse rounded-2xl"></div>
    </div>
  {:else if error && !auth}
    <div class="text-destructive rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm">
      {error}
    </div>
  {:else if auth}
    <div class="space-y-4">
      {#if error}
        <div class="rounded-xl border border-red-200 bg-red-50 px-4 py-3">
          <div class="text-sm text-red-900">{error}</div>
          {#if errorCode}
            <div class="mt-1 font-mono text-xs text-red-700">{errorCode}</div>
          {/if}
        </div>
      {/if}

      <AdminAuthOverview {auth} user={authStore.user} />

      <AdminAuthForm
        {auth}
        bind:form={oidcForm}
        {actionKey}
        onSave={() => void handleSave()}
        onTest={() => void handleTest()}
        onEnable={() => void handleEnable()}
        onDisable={() => void handleDisable()}
      />

      <AdminAuthDiagnostics {auth} {transition} />

      <AdminAuthRuntimeDetails {auth} />
    </div>
  {/if}
</PageScaffold>
