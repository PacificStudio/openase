<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type {
    OIDCDraftTestResponse,
    OIDCEnableResponse,
    SecurityAuthSettings,
    SecuritySettingsResponse,
  } from '$lib/api/contracts'
  import { enableOIDC, saveOIDCDraft, testOIDCDraft } from '$lib/api/openase'
  import {
    createOIDCFormState,
    oidcDraftFormFromAuth,
    oidcDraftPayloadFromForm,
    oidcDraftSignature,
    type OIDCFormState,
  } from '$lib/features/auth'
  import { toastStore } from '$lib/stores/toast.svelte'
  import SecuritySettingsHumanAuthDisabledSetup from './security-settings-human-auth-disabled-setup.svelte'

  type Security = SecuritySettingsResponse['security']

  let {
    auth,
    projectId = '',
    onSecurityChange,
  }: {
    auth: SecurityAuthSettings
    projectId?: string
    onSecurityChange?: (security: Security) => void
  } = $props()

  let oidcForm = $state<OIDCFormState>(createOIDCFormState())
  let oidcActionKey = $state('')
  let oidcError = $state('')
  let oidcTestResult = $state<OIDCDraftTestResponse | null>(null)
  let oidcEnableResult = $state<OIDCEnableResponse['activation'] | null>(null)
  let lastDraftSignature = $state('')

  $effect(() => {
    const nextSignature = oidcDraftSignature(auth)
    if (nextSignature === lastDraftSignature) {
      return
    }
    oidcForm = oidcDraftFormFromAuth(auth)
    lastDraftSignature = nextSignature
  })

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
      const payload = await saveOIDCDraft(resolvedProjectId, oidcDraftPayloadFromForm(oidcForm))
      onSecurityChange?.(payload.security)
      oidcEnableResult = null
      toastStore.success(
        'OIDC draft saved. Local bootstrap remains the browser access path until you explicitly enable OIDC.',
      )
    })
  }

  async function handleTestOIDCDraft() {
    await runOIDCAction('test', async (resolvedProjectId) => {
      oidcTestResult = await testOIDCDraft(resolvedProjectId, oidcDraftPayloadFromForm(oidcForm))
      oidcEnableResult = null
      toastStore.success('OIDC provider discovery succeeded.')
    })
  }

  async function handleEnableOIDC() {
    await runOIDCAction('enable', async (resolvedProjectId) => {
      const payload = await enableOIDC(resolvedProjectId, oidcDraftPayloadFromForm(oidcForm))
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
  onSessionTTL={(value) => (oidcForm.sessionTTL = value)}
  onSessionIdleTTL={(value) => (oidcForm.sessionIdleTTL = value)}
  onSave={() => void handleSaveOIDCDraft()}
  onTest={() => void handleTestOIDCDraft()}
  onEnable={() => void handleEnableOIDC()}
/>
