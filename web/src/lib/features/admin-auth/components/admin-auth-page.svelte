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
  import { SecuritySettingsHumanAuthGuideLinks } from '$lib/features/settings'
  import * as Collapsible from '$ui/collapsible'
  import { ChevronDown } from '@lucide/svelte'
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
  let errorCode = $state('')
  let actionKey = $state('')
  let auth = $state<SecurityAuthSettings | null>(null)
  let transition = $state<AdminAuthModeTransitionResponse['transition'] | null>(null)
  let lastDraftSignature = $state('')
  let runtimeOpen = $state(false)
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
        const payload = await saveAdminOIDCDraft(oidcDraftPayload())
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
        const payload = await testAdminOIDCDraft(oidcDraftPayload())
        applyValidationResult(payload)
        transition = null
        toastStore.success(
          payload.issuer_url
            ? `Validation passed — ${payload.issuer_url}`
            : 'Validation passed.',
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
        const payload = await enableAdminOIDC(oidcDraftPayload())
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
        toastStore.success('Switched to local bootstrap.')
      },
      'Failed to disable OIDC.',
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

      <!-- 1. Hero: current auth status -->
      <AdminAuthOverview {auth} user={authStore.user} />

      <!-- 2. OIDC configuration form -->
      <AdminAuthForm
        {auth}
        bind:form={oidcForm}
        {actionKey}
        onSave={() => void handleSave()}
        onTest={() => void handleTest()}
        onEnable={() => void handleEnable()}
        onDisable={() => void handleDisable()}
      />

      <!-- 3. Diagnostics (collapsible) -->
      <AdminAuthDiagnostics {auth} {transition} />

      <!-- 4. Runtime details (collapsible) -->
      <Collapsible.Root bind:open={runtimeOpen}>
        <div class="border-border bg-card rounded-2xl border">
          <Collapsible.Trigger class="flex w-full items-center justify-between px-5 py-4 text-left">
            <span class="text-sm font-semibold">Runtime details</span>
            <ChevronDown
              class="text-muted-foreground size-4 shrink-0 transition-transform duration-200 {runtimeOpen
                ? 'rotate-180'
                : ''}"
            />
          </Collapsible.Trigger>
          <Collapsible.Content>
            <div class="space-y-4 border-t px-5 pb-5 pt-4">
              <div class="grid gap-4 sm:grid-cols-3">
                <div>
                  <div class="text-muted-foreground text-xs">Session TTL</div>
                  <div class="mt-1 text-sm font-medium">{auth.session_policy.session_ttl}</div>
                </div>
                <div>
                  <div class="text-muted-foreground text-xs">Idle TTL</div>
                  <div class="mt-1 text-sm font-medium">{auth.session_policy.session_idle_ttl}</div>
                </div>
                <div>
                  <div class="text-muted-foreground text-xs">Config file</div>
                  <div class="mt-1 font-mono text-xs break-all">
                    {auth.config_path || 'Not available'}
                  </div>
                </div>
              </div>

              {#if auth.next_steps.length > 0}
                <div>
                  <div class="text-muted-foreground mb-2 text-xs font-medium">Next steps</div>
                  <ol
                    class="text-muted-foreground list-inside list-decimal space-y-1.5 text-sm leading-relaxed"
                  >
                    {#each auth.next_steps as step (step)}
                      <li>{step}</li>
                    {/each}
                  </ol>
                </div>
              {/if}
            </div>
          </Collapsible.Content>
        </div>
      </Collapsible.Root>

      <!-- 5. Guide links -->
      {#if auth.docs.length > 0}
        <SecuritySettingsHumanAuthGuideLinks docs={auth.docs} />
      {/if}
    </div>
  {/if}
</PageScaffold>
