<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type {
    AdminOIDCEnableResponse,
    AdminSecuritySettingsResponse,
    OIDCDraftTestResponse,
    SecurityAuthSettings,
  } from '$lib/api/contracts'
  import { enableAdminOIDC, saveAdminOIDCDraft, testAdminOIDCDraft } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import SecuritySettingsHumanAuthDisabledSetup from '$lib/features/settings/components/security-settings-human-auth-disabled-setup.svelte'

  type AdminSettings = AdminSecuritySettingsResponse['settings']

  type OIDCFormState = {
    issuerURL: string
    clientID: string
    clientSecret: string
    redirectURL: string
    scopesText: string
    allowedDomainsText: string
    bootstrapAdminEmailsText: string
  }

  let {
    auth,
    onSettingsChange,
  }: {
    auth: SecurityAuthSettings
    onSettingsChange?: (settings: AdminSettings) => void
  } = $props()

  let oidcForm = $state<OIDCFormState>({
    issuerURL: '',
    clientID: '',
    clientSecret: '',
    redirectURL: '',
    scopesText: '',
    allowedDomainsText: '',
    bootstrapAdminEmailsText: '',
  })
  let oidcActionKey = $state('')
  let oidcError = $state('')
  let oidcTestResult = $state<OIDCDraftTestResponse | null>(null)
  let oidcEnableResult = $state<AdminOIDCEnableResponse['activation'] | null>(null)
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
      redirectURL: auth.oidc_draft.redirect_url,
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
      redirect_url: oidcForm.redirectURL.trim(),
      scopes: parseListInput(oidcForm.scopesText),
      allowed_email_domains: parseListInput(oidcForm.allowedDomainsText),
      bootstrap_admin_emails: parseListInput(oidcForm.bootstrapAdminEmailsText),
    }
  }

  async function runOIDCAction(action: 'save' | 'test' | 'enable', runner: () => Promise<void>) {
    oidcActionKey = action
    oidcError = ''

    try {
      await runner()
    } catch (caughtError) {
      const message = caughtError instanceof ApiError ? caughtError.detail : 'OIDC update failed.'
      oidcError = message
      toastStore.error(message)
    } finally {
      oidcActionKey = ''
    }
  }

  async function handleSaveOIDCDraft() {
    await runOIDCAction('save', async () => {
      const payload = await saveAdminOIDCDraft(oidcDraftPayload())
      onSettingsChange?.(payload.settings)
      oidcEnableResult = null
      toastStore.success(
        'OIDC draft saved. Disabled mode remains active until you explicitly enable OIDC.',
      )
    })
  }

  async function handleTestOIDCDraft() {
    await runOIDCAction('test', async () => {
      oidcTestResult = await testAdminOIDCDraft(oidcDraftPayload())
      oidcEnableResult = null
      toastStore.success('OIDC provider discovery succeeded.')
    })
  }

  async function handleEnableOIDC() {
    await runOIDCAction('enable', async () => {
      const payload = await enableAdminOIDC(oidcDraftPayload())
      onSettingsChange?.(payload.settings)
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
  onRedirectURL={(value) => (oidcForm.redirectURL = value)}
  onScopes={(value) => (oidcForm.scopesText = value)}
  onAllowedDomains={(value) => (oidcForm.allowedDomainsText = value)}
  onBootstrapAdmins={(value) => (oidcForm.bootstrapAdminEmailsText = value)}
  onSave={() => void handleSaveOIDCDraft()}
  onTest={() => void handleTestOIDCDraft()}
  onEnable={() => void handleEnableOIDC()}
/>
