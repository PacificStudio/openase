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
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import {
    AlertTriangle,
    CheckCircle2,
    LockKeyhole,
    RefreshCcw,
    Rocket,
    Save,
    TestTube2,
  } from '@lucide/svelte'
  import SecuritySettingsHumanAuthGuideLinks from '$lib/features/settings/components/security-settings-human-auth-guide-links.svelte'
  import SecuritySettingsHumanAuthSummary from '$lib/features/settings/components/security-settings-human-auth-summary.svelte'
  import { formatTimestamp } from '$lib/features/settings/components/security-settings-human-auth.model'

  type OIDCFormState = {
    issuerURL: string
    clientID: string
    clientSecret: string
    redirectURL: string
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
    redirectURL: '',
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
      redirectURL: nextAuth.oidc_draft.redirect_url,
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
      redirect_url: oidcForm.redirectURL.trim(),
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
        toastStore.success('Disabled mode is now the configured fallback for the instance.')
      },
      'Failed to revert the instance auth mode to disabled.',
    )
  }

  $effect(() => {
    void refreshAuth()
  })
</script>

<PageScaffold
  title="Admin Auth"
  description="Instance-level authentication, OIDC rollout, bootstrap admins, and validation diagnostics."
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
        <div class="border-border bg-card space-y-4 rounded-2xl border p-5">
          <div class="flex flex-wrap items-center gap-2">
            <Badge variant="outline">Instance scope</Badge>
            <Badge variant={auth.active_mode === 'oidc' ? 'default' : 'secondary'}>
              Active {auth.active_mode}
            </Badge>
            {#if auth.configured_mode !== auth.active_mode}
              <Badge variant="outline">Configured {auth.configured_mode}</Badge>
            {/if}
            <Badge variant={auth.public_exposure_risk === 'high' ? 'destructive' : 'secondary'}>
              {auth.public_exposure_risk === 'high'
                ? 'Public exposure risk'
                : 'Local-ready posture'}
            </Badge>
          </div>

          <SecuritySettingsHumanAuthSummary
            authMode={auth.active_mode}
            configuredMode={auth.configured_mode}
            issuerURL={auth.issuer_url ?? ''}
            user={authStore.user}
            bootstrapSummary={auth.bootstrap_state.summary}
            publicExposureRisk={auth.public_exposure_risk}
            localPrincipal={auth.local_principal}
          />

          <div class="grid gap-3 md:grid-cols-2">
            {#each auth.warnings as warning (warning)}
              <div
                class={`rounded-xl border px-3 py-2 text-xs leading-relaxed ${
                  auth.public_exposure_risk === 'high'
                    ? 'border-amber-300 bg-amber-50 text-amber-950'
                    : 'border-sky-200 bg-sky-50 text-sky-950'
                }`}
              >
                <div class="flex items-start gap-2">
                  <AlertTriangle class="mt-0.5 size-4 shrink-0" />
                  <span>{warning}</span>
                </div>
              </div>
            {/each}
          </div>

          <div class="grid gap-3 md:grid-cols-2">
            <div class="rounded-xl border border-dashed p-3">
              <div class="text-muted-foreground text-xs">Configured session TTL</div>
              <div class="mt-1 text-sm font-medium">{auth.session_policy.session_ttl}</div>
            </div>
            <div class="rounded-xl border border-dashed p-3">
              <div class="text-muted-foreground text-xs">Idle session TTL</div>
              <div class="mt-1 text-sm font-medium">{auth.session_policy.session_idle_ttl}</div>
            </div>
            <div class="rounded-xl border border-dashed p-3 md:col-span-2">
              <div class="text-muted-foreground text-xs">Config path</div>
              <div class="mt-1 font-mono text-xs break-all">
                {auth.config_path || 'Not available'}
              </div>
            </div>
          </div>

          <div class="space-y-2">
            <div class="text-sm font-semibold">Operator checklist</div>
            <ol class="list-inside list-decimal space-y-1 text-sm leading-relaxed">
              {#each auth.next_steps as step (step)}
                <li>{step}</li>
              {/each}
            </ol>
          </div>
        </div>

        <div class="border-border bg-card space-y-4 rounded-2xl border p-5">
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="text-sm font-semibold">Last validation diagnostics</div>
              <div class="text-muted-foreground mt-1 text-xs">
                The most recent OIDC discovery result is persisted server-side for rollout
                troubleshooting.
              </div>
            </div>
            <Badge
              variant={auth.last_validation.status === 'ok'
                ? 'secondary'
                : auth.last_validation.status === 'failed'
                  ? 'destructive'
                  : 'outline'}
            >
              {auth.last_validation.status}
            </Badge>
          </div>

          <div class="rounded-xl border px-4 py-3">
            <div class="flex items-start gap-2">
              <LockKeyhole class="text-muted-foreground mt-0.5 size-4 shrink-0" />
              <div class="space-y-2 text-sm">
                <div class="font-medium">{auth.last_validation.message}</div>
                {#if auth.last_validation.checked_at}
                  <div class="text-muted-foreground text-xs">
                    Last checked {formatTimestamp(auth.last_validation.checked_at)}
                  </div>
                {/if}
              </div>
            </div>
          </div>

          <div class="grid gap-3">
            <div>
              <div class="text-muted-foreground text-xs">Issuer</div>
              <div class="mt-1 text-sm font-medium break-all">
                {auth.last_validation.issuer_url || 'Not recorded'}
              </div>
            </div>
            <div>
              <div class="text-muted-foreground text-xs">Authorization endpoint</div>
              <div class="mt-1 text-sm break-all">
                {auth.last_validation.authorization_endpoint || 'Not recorded'}
              </div>
            </div>
            <div>
              <div class="text-muted-foreground text-xs">Token endpoint</div>
              <div class="mt-1 text-sm break-all">
                {auth.last_validation.token_endpoint || 'Not recorded'}
              </div>
            </div>
            <div>
              <div class="text-muted-foreground text-xs">Redirect URL</div>
              <div class="mt-1 text-sm break-all">
                {auth.last_validation.redirect_url ||
                  auth.oidc_draft.redirect_url ||
                  'Not configured'}
              </div>
            </div>
            {#if auth.last_validation.warnings.length > 0}
              <div
                class="space-y-2 rounded-xl border border-amber-200 bg-amber-50 px-3 py-3 text-xs text-amber-950"
              >
                {#each auth.last_validation.warnings as warning (warning)}
                  <div>{warning}</div>
                {/each}
              </div>
            {/if}
          </div>

          {#if transition}
            <div
              class="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-950"
            >
              <div class="flex items-start gap-2">
                <CheckCircle2 class="mt-0.5 size-4 shrink-0" />
                <div class="space-y-2">
                  <div class="font-medium">{transition.message}</div>
                  {#if transition.restart_required}
                    <div class="text-xs font-medium tracking-wide text-emerald-800 uppercase">
                      Restart required
                    </div>
                  {/if}
                  <ol class="list-inside list-decimal space-y-1 text-xs leading-relaxed">
                    {#each transition.next_steps as step (step)}
                      <li>{step}</li>
                    {/each}
                  </ol>
                </div>
              </div>
            </div>
          {/if}

          <SecuritySettingsHumanAuthGuideLinks docs={auth.docs} />
        </div>
      </div>

      <div class="border-border bg-card space-y-5 rounded-2xl border p-5">
        <div class="flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <div class="text-sm font-semibold">OIDC configuration</div>
            <div class="text-muted-foreground mt-1 text-sm leading-relaxed">
              Save stores the server-side draft, Test validates discovery and refreshes diagnostics,
              Enable OIDC flips the configured mode, and Revert to disabled keeps the saved OIDC
              draft while restoring the break-glass local path.
            </div>
          </div>

          <div class="flex flex-wrap gap-2">
            <Button variant="outline" onclick={() => void handleSave()} disabled={actionKey !== ''}>
              <Save class="size-4" />
              {actionKey === 'save' ? 'Saving…' : 'Save configuration'}
            </Button>
            <Button variant="outline" onclick={() => void handleTest()} disabled={actionKey !== ''}>
              <TestTube2 class="size-4" />
              {actionKey === 'test' ? 'Testing…' : 'Test configuration'}
            </Button>
            <Button onclick={() => void handleEnable()} disabled={actionKey !== ''}>
              <Rocket class="size-4" />
              {actionKey === 'enable' ? 'Enabling…' : 'Enable OIDC'}
            </Button>
            <Button
              variant="outline"
              onclick={() => void handleDisable()}
              disabled={actionKey !== ''}
            >
              <RefreshCcw class="size-4" />
              {actionKey === 'disable' ? 'Reverting…' : 'Revert to disabled'}
            </Button>
          </div>
        </div>

        <div class="grid gap-4 lg:grid-cols-2">
          <div class="space-y-2">
            <Label for="admin-oidc-issuer-url">Issuer URL</Label>
            <Input
              id="admin-oidc-issuer-url"
              value={oidcForm.issuerURL}
              placeholder="https://idp.example.com/realms/openase"
              oninput={(event) =>
                (oidcForm.issuerURL = (event.currentTarget as HTMLInputElement).value)}
            />
          </div>
          <div class="space-y-2">
            <Label for="admin-oidc-client-id">Client ID</Label>
            <Input
              id="admin-oidc-client-id"
              value={oidcForm.clientID}
              placeholder="openase"
              oninput={(event) =>
                (oidcForm.clientID = (event.currentTarget as HTMLInputElement).value)}
            />
          </div>
          <div class="space-y-2">
            <Label for="admin-oidc-client-secret">Client secret</Label>
            <Input
              id="admin-oidc-client-secret"
              type="password"
              value={oidcForm.clientSecret}
              placeholder={auth.oidc_draft.client_secret_configured
                ? 'Leave blank to keep the saved secret'
                : 'Paste the current client secret'}
              oninput={(event) =>
                (oidcForm.clientSecret = (event.currentTarget as HTMLInputElement).value)}
            />
            <p class="text-muted-foreground text-[11px]">
              {auth.oidc_draft.client_secret_configured
                ? 'A client secret is already stored server-side. Leave this blank to preserve it.'
                : 'The client secret is accepted server-side and never echoed back in plain text.'}
            </p>
          </div>
          <div class="space-y-2">
            <Label for="admin-oidc-redirect-url">Redirect URL</Label>
            <Input
              id="admin-oidc-redirect-url"
              value={oidcForm.redirectURL}
              placeholder="http://127.0.0.1:19836/api/v1/auth/oidc/callback"
              oninput={(event) =>
                (oidcForm.redirectURL = (event.currentTarget as HTMLInputElement).value)}
            />
          </div>
          <div class="space-y-2 lg:col-span-2">
            <Label for="admin-oidc-scopes">Scopes</Label>
            <Textarea
              id="admin-oidc-scopes"
              rows={3}
              value={oidcForm.scopesText}
              placeholder="openid, profile, email, groups"
              oninput={(event) =>
                (oidcForm.scopesText = (event.currentTarget as HTMLTextAreaElement).value)}
            />
            <p class="text-muted-foreground text-[11px]">Use commas or new lines.</p>
          </div>
          <div class="space-y-2">
            <Label for="admin-oidc-allowed-domains">Allowed domains</Label>
            <Textarea
              id="admin-oidc-allowed-domains"
              rows={3}
              value={oidcForm.allowedDomainsText}
              placeholder="example.com"
              oninput={(event) =>
                (oidcForm.allowedDomainsText = (event.currentTarget as HTMLTextAreaElement).value)}
            />
          </div>
          <div class="space-y-2">
            <Label for="admin-oidc-bootstrap-admins">Bootstrap admin emails</Label>
            <Textarea
              id="admin-oidc-bootstrap-admins"
              rows={3}
              value={oidcForm.bootstrapAdminEmailsText}
              placeholder="admin@example.com"
              oninput={(event) =>
                (oidcForm.bootstrapAdminEmailsText = (
                  event.currentTarget as HTMLTextAreaElement
                ).value)}
            />
          </div>
        </div>
      </div>
    </div>
  {/if}
</PageScaffold>
