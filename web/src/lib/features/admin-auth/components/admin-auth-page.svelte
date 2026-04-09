<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import { ApiError } from '$lib/api/client'
  import type {
    AdminAuthModeTransitionResponse,
    OIDCDraftTestResponse,
    SecurityAuthSettings,
  } from '$lib/api/contracts'
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

  type OIDCFormState = {
    issuerURL: string
    clientID: string
    clientSecret: string
    redirectMode: 'auto' | 'fixed'
    fixedRedirectURL: string
    scopesText: string
    allowedDomainsText: string
    bootstrapAdminEmailsText: string
  }

  let loading = $state(false)
  let error = $state('')
  let actionKey = $state('')
  let auth = $state<SecurityAuthSettings | null>(null)
  let transition = $state<AdminAuthModeTransitionResponse['transition'] | null>(null)
  let lastDraftSignature = $state('')
  let oidcForm = $state<OIDCFormState>({
    issuerURL: '',
    clientID: '',
    clientSecret: '',
    redirectMode: 'auto',
    fixedRedirectURL: '',
    scopesText: '',
    allowedDomainsText: '',
    bootstrapAdminEmailsText: '',
  })

  function parseListInput(value: string) {
    return value
      .split(/[\n,]/)
      .map((item) => item.trim())
      .filter(Boolean)
  }

  function syncForm(nextAuth: SecurityAuthSettings) {
    const nextSignature = JSON.stringify(nextAuth.oidc_draft)
    if (nextSignature === lastDraftSignature) {
      return
    }
    oidcForm = {
      issuerURL: nextAuth.oidc_draft.issuer_url,
      clientID: nextAuth.oidc_draft.client_id,
      clientSecret: '',
      redirectMode: nextAuth.oidc_draft.redirect_mode === 'fixed' ? 'fixed' : 'auto',
      fixedRedirectURL: nextAuth.oidc_draft.fixed_redirect_url,
      scopesText: nextAuth.oidc_draft.scopes.join('\n'),
      allowedDomainsText: nextAuth.oidc_draft.allowed_email_domains.join('\n'),
      bootstrapAdminEmailsText: nextAuth.oidc_draft.bootstrap_admin_emails.join('\n'),
    }
    lastDraftSignature = nextSignature
  }

  function oidcDraftPayload() {
    return {
      issuer_url: oidcForm.issuerURL.trim(),
      client_id: oidcForm.clientID.trim(),
      client_secret: oidcForm.clientSecret.trim(),
      redirect_mode: oidcForm.redirectMode,
      fixed_redirect_url: oidcForm.fixedRedirectURL.trim(),
      scopes: parseListInput(oidcForm.scopesText),
      allowed_email_domains: parseListInput(oidcForm.allowedDomainsText),
      bootstrap_admin_emails: parseListInput(oidcForm.bootstrapAdminEmailsText),
    }
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
    try {
      await runner()
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : failureFallback
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
        const payload = await saveAdminOIDCDraft(oidcDraftPayload())
        auth = payload.auth
        syncForm(payload.auth)
        transition = null
        toastStore.success('OIDC draft saved for the instance. Active auth mode stays unchanged.')
      },
      'Failed to save the instance auth draft.',
    )
  }

  async function handleTest() {
    await runAction(
      'test',
      async () => {
        const payload = await testAdminOIDCDraft(oidcDraftPayload())
        applyValidationResult(payload)
        transition = null
        toastStore.success('OIDC provider discovery succeeded.')
      },
      'Failed to validate the OIDC provider.',
    )
    if (error) {
      await refreshAuth()
    }
  }

  async function handleEnable() {
    await runAction(
      'enable',
      async () => {
        const payload = await enableAdminOIDC(oidcDraftPayload())
        auth = payload.auth
        syncForm(payload.auth)
        transition = payload.transition
        toastStore.success('OIDC is now the configured auth mode for the instance.')
      },
      'Failed to enable OIDC for the instance.',
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
  description="Current browser auth method, OIDC draft state, validation diagnostics, and activation controls."
>
  {#if loading}
    <div class="space-y-4">
      <div class="bg-muted h-24 animate-pulse rounded-xl"></div>
      <div class="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <div class="bg-muted h-96 animate-pulse rounded-xl"></div>
        <div class="bg-muted h-96 animate-pulse rounded-xl"></div>
      </div>
    </div>
  {:else if error}
    <div class="text-destructive rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm">
      {error}
    </div>
  {:else if auth}
    <div class="space-y-6">
      <div class="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
        <AdminAuthOverview {auth} user={authStore.user} />
        <AdminAuthDiagnostics {auth} {transition} />
      </div>

      <AdminAuthForm
        {auth}
        bind:form={oidcForm}
        {actionKey}
        onSave={() => void handleSave()}
        onTest={() => void handleTest()}
        onEnable={() => void handleEnable()}
        onDisable={() => void handleDisable()}
      />
    </div>
  {/if}
</PageScaffold>
