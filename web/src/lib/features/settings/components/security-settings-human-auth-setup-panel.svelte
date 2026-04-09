<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type {
    OIDCDraftTestResponse,
    OIDCEnableResponse,
    SecurityAuthSettings,
    SecuritySettingsResponse,
  } from '$lib/api/contracts'
  import { enableOIDC, saveOIDCDraft, testOIDCDraft } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import SecuritySettingsHumanAuthDisabledSetup from './security-settings-human-auth-disabled-setup.svelte'

  type Security = SecuritySettingsResponse['security']

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

  let {
    auth,
    projectId = '',
    onSecurityChange,
  }: {
    auth: SecurityAuthSettings
    projectId?: string
    onSecurityChange?: (security: Security) => void
  } = $props()

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
  let oidcActionKey = $state('')
  let oidcError = $state('')
  let oidcTestResult = $state<OIDCDraftTestResponse | null>(null)
  let oidcEnableResult = $state<OIDCEnableResponse['activation'] | null>(null)
  let lastDraftSignature = $state('')

  $effect(() => {
    const nextSignature = JSON.stringify(auth.oidc_draft)
    if (nextSignature === lastDraftSignature) {
      return
    }
    oidcForm = {
      issuerURL: auth.oidc_draft.issuer_url,
      clientID: auth.oidc_draft.client_id,
      clientSecret: '',
      redirectMode: auth.oidc_draft.redirect_mode === 'fixed' ? 'fixed' : 'auto',
      fixedRedirectURL: auth.oidc_draft.fixed_redirect_url,
      scopesText: auth.oidc_draft.scopes.join('\n'),
      allowedDomainsText: auth.oidc_draft.allowed_email_domains.join('\n'),
      bootstrapAdminEmailsText: auth.oidc_draft.bootstrap_admin_emails.join('\n'),
    }
    lastDraftSignature = nextSignature
  })

  function parseListInput(value: string) {
    return value
      .split(/[\n,]/)
      .map((item) => item.trim())
      .filter(Boolean)
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

  async function runOIDCAction(
    action: 'save' | 'test' | 'enable',
    runner: (projectId: string) => Promise<void>,
  ) {
    if (!projectId) {
      return
    }

    oidcActionKey = action
    oidcError = ''

    try {
      await runner(projectId)
    } catch (caughtError) {
      const message = caughtError instanceof ApiError ? caughtError.detail : 'OIDC update failed.'
      oidcError = message
      toastStore.error(message)
    } finally {
      oidcActionKey = ''
    }
  }

  async function handleSaveOIDCDraft() {
    await runOIDCAction('save', async (resolvedProjectId) => {
      const payload = await saveOIDCDraft(resolvedProjectId, oidcDraftPayload())
      onSecurityChange?.(payload.security)
      oidcEnableResult = null
      toastStore.success(
        'OIDC draft saved. Local bootstrap remains the browser access path until you explicitly enable OIDC.',
      )
    })
  }

  async function handleTestOIDCDraft() {
    await runOIDCAction('test', async (resolvedProjectId) => {
      oidcTestResult = await testOIDCDraft(resolvedProjectId, oidcDraftPayload())
      oidcEnableResult = null
      toastStore.success('OIDC provider discovery succeeded.')
    })
  }

  async function handleEnableOIDC() {
    await runOIDCAction('enable', async (resolvedProjectId) => {
      const payload = await enableOIDC(resolvedProjectId, oidcDraftPayload())
      onSecurityChange?.(payload.security)
      oidcEnableResult = payload.activation
      toastStore.success(
        'OIDC is now the configured auth mode. Follow the rollout steps to activate it.',
      )
    })
  }
</script>

<SecuritySettingsHumanAuthDisabledSetup
  {auth}
  form={oidcForm}
  actionKey={oidcActionKey}
  error={oidcError}
  testResult={oidcTestResult}
  enableResult={oidcEnableResult}
  onIssuerURL={(value) => (oidcForm.issuerURL = value)}
  onClientID={(value) => (oidcForm.clientID = value)}
  onClientSecret={(value) => (oidcForm.clientSecret = value)}
  onRedirectMode={(value) => (oidcForm.redirectMode = value)}
  onFixedRedirectURL={(value) => (oidcForm.fixedRedirectURL = value)}
  onScopes={(value) => (oidcForm.scopesText = value)}
  onAllowedDomains={(value) => (oidcForm.allowedDomainsText = value)}
  onBootstrapAdmins={(value) => (oidcForm.bootstrapAdminEmailsText = value)}
  onSave={() => void handleSaveOIDCDraft()}
  onTest={() => void handleTestOIDCDraft()}
  onEnable={() => void handleEnableOIDC()}
/>
