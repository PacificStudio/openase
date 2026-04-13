<script lang="ts">
  import { CheckCircle2, Rocket, Save, TestTube2 } from '@lucide/svelte'
  import { oidcSessionFieldCopy } from '$lib/features/auth'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import OIDCRedirectFields from './oidc-redirect-fields.svelte'
  import type { SecuritySettingsHumanAuthDisabledSetupProps } from './security-settings-human-auth-disabled-setup-model'

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
  }: SecuritySettingsHumanAuthDisabledSetupProps = $props()

  const t = i18nStore.t
  const buttonDisabled = actionKey !== ''
</script>

<div class="border-border bg-card space-y-4 rounded-lg border p-4">
  <div>
    <h4 class="text-sm font-semibold">
      {t('settings.security.authDisabled.draft.heading')}
    </h4>
    <p class="text-muted-foreground mt-1 text-xs leading-relaxed">
      {t('settings.security.authDisabled.draft.description')}
    </p>
  </div>

  <div class="grid gap-4 lg:grid-cols-2">
    <div class="space-y-2">
      <Label for="oidc-issuer-url">{t('settings.security.authDisabled.labels.issuerURL')}</Label>
      <Input
        id="oidc-issuer-url"
        value={form.issuerURL}
        placeholder={t('settings.security.authDisabled.placeholders.issuerURL')}
        oninput={(event) => onIssuerURL((event.currentTarget as HTMLInputElement).value)}
      />
    </div>
    <div class="space-y-2">
      <Label for="oidc-client-id">{t('settings.security.authDisabled.labels.clientID')}</Label>
      <Input
        id="oidc-client-id"
        value={form.clientID}
        placeholder={t('settings.security.authDisabled.placeholders.clientID')}
        oninput={(event) => onClientID((event.currentTarget as HTMLInputElement).value)}
      />
    </div>
    <div class="space-y-2">
      <Label for="oidc-client-secret"
        >{t('settings.security.authDisabled.labels.clientSecret')}</Label
      >
      <Input
        id="oidc-client-secret"
        type="password"
        value={form.clientSecret}
        placeholder={auth.oidc_draft.client_secret_configured
          ? t('settings.security.authDisabled.placeholders.clientSecretConfigured')
          : t('settings.security.authDisabled.placeholders.clientSecretNew')}
        oninput={(event) => onClientSecret((event.currentTarget as HTMLInputElement).value)}
      />
      <p class="text-muted-foreground text-[11px]">
        {auth.oidc_draft.client_secret_configured
          ? t('settings.security.authDisabled.hints.clientSecretConfigured')
          : t('settings.security.authDisabled.hints.clientSecretNew')}
      </p>
    </div>
    <OIDCRedirectFields
      redirectMode={form.redirectMode}
      fixedRedirectURL={form.fixedRedirectURL}
      {onRedirectMode}
      {onFixedRedirectURL}
    />
    <div class="space-y-2">
      <Label for="oidc-session-ttl">{t('settings.security.authDisabled.labels.sessionTTL')}</Label>
      <Input
        id="oidc-session-ttl"
        value={form.sessionTTL}
        placeholder={t('settings.security.authDisabled.placeholders.sessionTTL')}
        oninput={(event) => onSessionTTL((event.currentTarget as HTMLInputElement).value)}
      />
      <p class="text-muted-foreground text-[11px]">{oidcSessionFieldCopy.sessionTTLDescription}</p>
    </div>
    <div class="space-y-2">
      <Label for="oidc-session-idle-ttl">{t('settings.security.authDisabled.labels.idleTTL')}</Label
      >
      <Input
        id="oidc-session-idle-ttl"
        value={form.sessionIdleTTL}
        placeholder={t('settings.security.authDisabled.placeholders.idleTTL')}
        oninput={(event) => onSessionIdleTTL((event.currentTarget as HTMLInputElement).value)}
      />
      <p class="text-muted-foreground text-[11px]">
        {oidcSessionFieldCopy.sessionIdleTTLDescription}
      </p>
    </div>
    <div class="space-y-2 lg:col-span-2">
      <Label for="oidc-scopes">{t('settings.security.authDisabled.labels.scopes')}</Label>
      <Textarea
        id="oidc-scopes"
        rows={3}
        value={form.scopesText}
        placeholder={t('settings.security.authDisabled.placeholders.scopes')}
        oninput={(event) => onScopes((event.currentTarget as HTMLTextAreaElement).value)}
      />
      <p class="text-muted-foreground text-[11px]">
        {t('settings.security.authDisabled.hints.scopes')}
      </p>
    </div>
    <div class="space-y-2">
      <Label for="oidc-allowed-domains"
        >{t('settings.security.authDisabled.labels.allowedDomains')}</Label
      >
      <Textarea
        id="oidc-allowed-domains"
        rows={3}
        value={form.allowedDomainsText}
        placeholder={t('settings.security.authDisabled.placeholders.allowedDomains')}
        oninput={(event) => onAllowedDomains((event.currentTarget as HTMLTextAreaElement).value)}
      />
    </div>
    <div class="space-y-2">
      <Label for="oidc-bootstrap-admins"
        >{t('settings.security.authDisabled.labels.bootstrapAdmins')}</Label
      >
      <Textarea
        id="oidc-bootstrap-admins"
        rows={3}
        value={form.bootstrapAdminEmailsText}
        placeholder={t('settings.security.authDisabled.placeholders.bootstrapAdmins')}
        oninput={(event) => onBootstrapAdmins((event.currentTarget as HTMLTextAreaElement).value)}
      />
    </div>
  </div>

  <div class="flex flex-wrap gap-3">
    <Button variant="outline" onclick={onSave} disabled={buttonDisabled}>
      <Save class="size-4" />
      {actionKey === 'save'
        ? t('settings.security.authDisabled.buttons.savingDraft')
        : t('settings.security.authDisabled.buttons.saveDraft')}
    </Button>
    <Button variant="outline" onclick={onTest} disabled={buttonDisabled}>
      <TestTube2 class="size-4" />
      {actionKey === 'test'
        ? t('settings.security.authDisabled.buttons.testing')
        : t('settings.security.authDisabled.buttons.testConfiguration')}
    </Button>
    <Button onclick={onEnable} disabled={buttonDisabled}>
      <Rocket class="size-4" />
      {actionKey === 'enable'
        ? t('settings.security.authDisabled.buttons.enabling')
        : t('settings.security.authDisabled.buttons.enableOIDC')}
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
                {t('settings.security.authDisabled.testResult.labels.issuer')}
              </div>
              <div class="font-mono break-all">{testResult.issuer_url}</div>
            </div>
            <div>
              <div class="text-emerald-800/80">
                {t('settings.security.authDisabled.testResult.labels.redirect')}
              </div>
              <div class="font-mono break-all">{testResult.redirect_url}</div>
            </div>
            <div>
              <div class="text-emerald-800/80">
                {t('settings.security.authDisabled.testResult.labels.authorizationEndpoint')}
              </div>
              <div class="font-mono break-all">{testResult.authorization_endpoint}</div>
            </div>
            <div>
              <div class="text-emerald-800/80">
                {t('settings.security.authDisabled.testResult.labels.tokenEndpoint')}
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
    <div class="rounded-lg border border-indigo-200 bg-indigo-50 px-4 py-3 text-sm text-indigo-950">
      <div class="flex items-start gap-2">
        <CheckCircle2 class="mt-0.5 size-4 shrink-0" />
        <div class="space-y-2">
          <div class="font-medium">{enableResult.message}</div>
          {#if enableResult.restart_required}
            <div class="text-xs font-medium tracking-wide text-indigo-700 uppercase">
              {t('settings.security.authDisabled.enableResult.restartRequired')}
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
