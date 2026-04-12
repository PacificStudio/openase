<script lang="ts">
  import { oidcSessionFieldCopy, type OIDCFormState } from '$lib/features/auth'
  import type { SecurityAuthSettings } from '$lib/api/contracts'
  import { AlertTriangle, CheckCircle2, Rocket, Save, TestTube2 } from '@lucide/svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import OIDCRedirectFields from './oidc-redirect-fields.svelte'

  type OIDCTestResult = {
    status: string
    message: string
    issuer_url: string
    authorization_endpoint: string
    token_endpoint: string
    redirect_url: string
    warnings: string[]
  }

  type OIDCEnableResult = {
    status: string
    message: string
    restart_required: boolean
    next_steps: string[]
  }

  let {
    auth,
    form,
    actionKey = '',
    error = '',
    testResult = null,
    enableResult = null,
    onIssuerURL,
    onClientID,
    onClientSecret,
    onRedirectMode,
    onFixedRedirectURL,
    onScopes,
    onAllowedDomains,
    onBootstrapAdmins,
    onSessionTTL,
    onSessionIdleTTL,
    onSave,
    onTest,
    onEnable,
  }: {
    auth: SecurityAuthSettings
    form: OIDCFormState
    actionKey?: string
    error?: string
    testResult?: OIDCTestResult | null
    enableResult?: OIDCEnableResult | null
    onIssuerURL: (value: string) => void
    onClientID: (value: string) => void
    onClientSecret: (value: string) => void
    onRedirectMode: (value: 'auto' | 'fixed') => void
    onFixedRedirectURL: (value: string) => void
    onScopes: (value: string) => void
    onAllowedDomains: (value: string) => void
    onBootstrapAdmins: (value: string) => void
    onSessionTTL: (value: string) => void
    onSessionIdleTTL: (value: string) => void
    onSave: () => void
    onTest: () => void
    onEnable: () => void
  } = $props()
</script>

<div class="space-y-4">
  <div class="border-border bg-card space-y-4 rounded-lg border p-4">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
      <div class="space-y-2">
      <div class="flex flex-wrap items-center gap-2">
        <h4 class="text-sm font-semibold">{i18nStore.t('settings.security.authDisabled.heading')}</h4>
        <Badge variant="outline">{auth.active_mode}</Badge>
        <Badge variant={auth.public_exposure_risk === 'high' ? 'destructive' : 'secondary'}>
          {auth.public_exposure_risk === 'high'
            ? i18nStore.t('settings.security.authDisabled.badges.highRisk')
            : i18nStore.t('settings.security.authDisabled.badges.localReady')}
        </Badge>
      </div>
      <p class="text-muted-foreground text-sm leading-relaxed">{auth.mode_summary}</p>
      <p class="text-muted-foreground text-xs leading-relaxed">
        {i18nStore.t('settings.security.authDisabled.localPrincipal.prefix')}
        <code>{auth.local_principal}</code>.
      </p>
      <p class="text-muted-foreground text-xs leading-relaxed">
        {i18nStore.t('settings.security.authDisabled.localPrincipal.suffix')}
      </p>
      </div>

      <div class="grid gap-2 text-xs sm:min-w-64 sm:grid-cols-2 lg:w-80">
        <div>
          <div class="text-muted-foreground">
            {i18nStore.t('settings.security.authDisabled.stats.configuredMode')}
          </div>
          <div class="mt-1 font-medium uppercase">{auth.configured_mode}</div>
        </div>
        <div>
          <div class="text-muted-foreground">
            {i18nStore.t('settings.security.authDisabled.stats.bootstrapAdmins')}
          </div>
          <div class="mt-1 font-medium">{auth.bootstrap_state.summary}</div>
        </div>
        <div class="sm:col-span-2">
          <div class="text-muted-foreground">
            {i18nStore.t('settings.security.authDisabled.stats.storedIn')}
          </div>
          <div class="mt-1 font-mono break-all">
            {auth.config_path ?? i18nStore.t('settings.security.authDisabled.stats.notAvailable')}
          </div>
        </div>
      </div>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      {#each auth.warnings as warning (warning)}
        <div
          class={`rounded-lg border px-3 py-2 text-xs leading-relaxed ${
            auth.public_exposure_risk === 'high'
              ? 'border-amber-300 bg-amber-50 text-amber-900'
              : 'border-sky-200 bg-sky-50 text-sky-900'
          }`}
        >
          <div class="flex items-start gap-2">
            <AlertTriangle class="mt-0.5 size-4 shrink-0" />
            <span>{warning}</span>
          </div>
        </div>
      {/each}
    </div>
  </div>

  <div class="border-border bg-card space-y-4 rounded-lg border p-4">
    <div>
      <h4 class="text-sm font-semibold">{i18nStore.t('settings.security.authDisabled.draft.heading')}</h4>
      <p class="text-muted-foreground mt-1 text-xs leading-relaxed">
        {i18nStore.t('settings.security.authDisabled.draft.description')}
      </p>
    </div>

    <div class="grid gap-4 lg:grid-cols-2">
      <div class="space-y-2">
        <Label for="oidc-issuer-url">{i18nStore.t('settings.security.authDisabled.labels.issuerURL')}</Label>
        <Input
          id="oidc-issuer-url"
          value={form.issuerURL}
          placeholder={i18nStore.t('settings.security.authDisabled.placeholders.issuerURL')}
          oninput={(event) => onIssuerURL((event.currentTarget as HTMLInputElement).value)}
        />
      </div>
      <div class="space-y-2">
        <Label for="oidc-client-id">{i18nStore.t('settings.security.authDisabled.labels.clientID')}</Label>
        <Input
          id="oidc-client-id"
          value={form.clientID}
          placeholder={i18nStore.t('settings.security.authDisabled.placeholders.clientID')}
          oninput={(event) => onClientID((event.currentTarget as HTMLInputElement).value)}
        />
      </div>
      <div class="space-y-2">
        <Label for="oidc-client-secret">
          {i18nStore.t('settings.security.authDisabled.labels.clientSecret')}
        </Label>
        <Input
          id="oidc-client-secret"
          type="password"
          value={form.clientSecret}
          placeholder={
            auth.oidc_draft.client_secret_configured
              ? i18nStore.t(
                  'settings.security.authDisabled.placeholders.clientSecretConfigured',
                )
              : i18nStore.t('settings.security.authDisabled.placeholders.clientSecretNew')
          }
          oninput={(event) => onClientSecret((event.currentTarget as HTMLInputElement).value)}
        />
        <p class="text-muted-foreground text-[11px]">
          {auth.oidc_draft.client_secret_configured
            ? i18nStore.t(
                'settings.security.authDisabled.hints.clientSecretConfigured',
              )
            : i18nStore.t('settings.security.authDisabled.hints.clientSecretNew')}
        </p>
      </div>
      <OIDCRedirectFields
        redirectMode={form.redirectMode}
        fixedRedirectURL={form.fixedRedirectURL}
        {onRedirectMode}
        {onFixedRedirectURL}
      />
      <div class="space-y-2">
        <Label for="oidc-session-ttl">
          {i18nStore.t('settings.security.authDisabled.labels.sessionTTL')}
        </Label>
        <Input
          id="oidc-session-ttl"
          value={form.sessionTTL}
          placeholder={i18nStore.t('settings.security.authDisabled.placeholders.sessionTTL')}
          oninput={(event) => onSessionTTL((event.currentTarget as HTMLInputElement).value)}
        />
        <p class="text-muted-foreground text-[11px]">
          {oidcSessionFieldCopy.sessionTTLDescription}
        </p>
      </div>
      <div class="space-y-2">
        <Label for="oidc-session-idle-ttl">
          {i18nStore.t('settings.security.authDisabled.labels.idleTTL')}
        </Label>
        <Input
          id="oidc-session-idle-ttl"
          value={form.sessionIdleTTL}
          placeholder={i18nStore.t('settings.security.authDisabled.placeholders.idleTTL')}
          oninput={(event) => onSessionIdleTTL((event.currentTarget as HTMLInputElement).value)}
        />
        <p class="text-muted-foreground text-[11px]">
          {oidcSessionFieldCopy.sessionIdleTTLDescription}
        </p>
      </div>
      <div class="space-y-2 lg:col-span-2">
        <Label for="oidc-scopes">
          {i18nStore.t('settings.security.authDisabled.labels.scopes')}
        </Label>
        <Textarea
          id="oidc-scopes"
          rows={3}
          value={form.scopesText}
          placeholder={i18nStore.t('settings.security.authDisabled.placeholders.scopes')}
          oninput={(event) => onScopes((event.currentTarget as HTMLTextAreaElement).value)}
        />
        <p class="text-muted-foreground text-[11px]">
          {i18nStore.t('settings.security.authDisabled.hints.scopes')}
        </p>
      </div>
      <div class="space-y-2">
        <Label for="oidc-allowed-domains">
          {i18nStore.t('settings.security.authDisabled.labels.allowedDomains')}
        </Label>
        <Textarea
          id="oidc-allowed-domains"
          rows={3}
          value={form.allowedDomainsText}
          placeholder={i18nStore.t('settings.security.authDisabled.placeholders.allowedDomains')}
          oninput={(event) => onAllowedDomains((event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>
      <div class="space-y-2">
        <Label for="oidc-bootstrap-admins">
          {i18nStore.t('settings.security.authDisabled.labels.bootstrapAdmins')}
        </Label>
        <Textarea
          id="oidc-bootstrap-admins"
          rows={3}
          value={form.bootstrapAdminEmailsText}
          placeholder={i18nStore.t('settings.security.authDisabled.placeholders.bootstrapAdmins')}
          oninput={(event) => onBootstrapAdmins((event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>
    </div>

    <div class="flex flex-wrap gap-3">
      <Button variant="outline" onclick={onSave} disabled={actionKey !== ''}>
        <Save class="size-4" />
        {actionKey === 'save'
          ? i18nStore.t('settings.security.authDisabled.buttons.savingDraft')
          : i18nStore.t('settings.security.authDisabled.buttons.saveDraft')}
      </Button>
      <Button variant="outline" onclick={onTest} disabled={actionKey !== ''}>
        <TestTube2 class="size-4" />
        {actionKey === 'test'
          ? i18nStore.t('settings.security.authDisabled.buttons.testing')
          : i18nStore.t('settings.security.authDisabled.buttons.testConfiguration')}
      </Button>
      <Button onclick={onEnable} disabled={actionKey !== ''}>
        <Rocket class="size-4" />
        {actionKey === 'enable'
          ? i18nStore.t('settings.security.authDisabled.buttons.enabling')
          : i18nStore.t('settings.security.authDisabled.buttons.enableOIDC')}
      </Button>
    </div>

    {#if error}
      <div class="text-destructive rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm">
        {error}
      </div>
    {/if}

    {#if testResult}
      <div
        class="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-950"
      >
        <div class="flex items-start gap-2">
          <CheckCircle2 class="mt-0.5 size-4 shrink-0" />
          <div class="space-y-2">
            <div class="font-medium">{testResult.message}</div>
            <div class="grid gap-2 text-xs md:grid-cols-2">
              <div>
                <div class="text-emerald-800/80">
                  {i18nStore.t('settings.security.authDisabled.testResult.labels.issuer')}
                </div>
                <div class="font-mono break-all">{testResult.issuer_url}</div>
              </div>
              <div>
                <div class="text-emerald-800/80">
                  {i18nStore.t('settings.security.authDisabled.testResult.labels.redirect')}
                </div>
                <div class="font-mono break-all">{testResult.redirect_url}</div>
              </div>
              <div>
                <div class="text-emerald-800/80">
                  {i18nStore.t(
                    'settings.security.authDisabled.testResult.labels.authorizationEndpoint',
                  )}
                </div>
                <div class="font-mono break-all">{testResult.authorization_endpoint}</div>
              </div>
              <div>
                <div class="text-emerald-800/80">
                  {i18nStore.t('settings.security.authDisabled.testResult.labels.tokenEndpoint')}
                </div>
                <div class="font-mono break-all">{testResult.token_endpoint}</div>
              </div>
            </div>
            {#if testResult.warnings.length > 0}
              <div class="space-y-1 text-xs">
                {#each testResult.warnings as warning (warning)}
                  <div>{warning}</div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      </div>
    {/if}

    {#if enableResult}
      <div
        class="rounded-lg border border-indigo-200 bg-indigo-50 px-4 py-3 text-sm text-indigo-950"
      >
        <div class="flex items-start gap-2">
          <CheckCircle2 class="mt-0.5 size-4 shrink-0" />
          <div class="space-y-2">
            <div class="font-medium">{enableResult.message}</div>
            {#if enableResult.restart_required}
              <div class="text-xs font-medium tracking-wide text-indigo-700 uppercase">
                {i18nStore.t('settings.security.authDisabled.enableResult.restartRequired')}
              </div>
            {/if}
            <ol class="list-inside list-decimal space-y-1 text-xs leading-relaxed">
              {#each enableResult.next_steps as step (step)}
                <li>{step}</li>
              {/each}
            </ol>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>
