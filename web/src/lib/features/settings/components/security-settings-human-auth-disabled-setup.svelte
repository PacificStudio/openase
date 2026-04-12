<script lang="ts">
  import { AlertTriangle } from '@lucide/svelte'
  import { Badge } from '$ui/badge'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import SecuritySettingsHumanAuthDisabledSetupDraftForm from './security-settings-human-auth-disabled-setup-draft-form.svelte'
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
</script>

<div class="space-y-4">
  <div class="border-border bg-card space-y-4 rounded-lg border p-4">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
      <div class="space-y-2">
        <div class="flex flex-wrap items-center gap-2">
          <h4 class="text-sm font-semibold">
            {i18nStore.t('settings.security.authDisabled.heading')}
          </h4>
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

  <SecuritySettingsHumanAuthDisabledSetupDraftForm
    {auth}
    {form}
    {actionKey}
    {error}
    {testResult}
    {enableResult}
    {onIssuerURL}
    {onClientID}
    {onClientSecret}
    {onRedirectMode}
    {onFixedRedirectURL}
    {onScopes}
    {onAllowedDomains}
    {onBootstrapAdmins}
    {onSessionTTL}
    {onSessionIdleTTL}
    {onSave}
    {onTest}
    {onEnable}
  />
</div>
