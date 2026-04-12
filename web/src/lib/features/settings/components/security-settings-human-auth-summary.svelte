<script lang="ts">
  import type { HumanAuthUser } from '$lib/stores/auth.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    authMode,
    configuredMode = '',
    issuerURL = '',
    user = null,
    bootstrapSummary = '',
    publicExposureRisk = '',
    localPrincipal = '',
  }: {
    authMode: string
    configuredMode?: string
    issuerURL?: string
    user?: HumanAuthUser | null
    bootstrapSummary?: string
    publicExposureRisk?: string
    localPrincipal?: string
  } = $props()
</script>

<div class="bg-muted/30 grid gap-3 rounded-lg px-4 py-3 text-xs md:grid-cols-2 xl:grid-cols-5">
  <div>
    <div class="text-muted-foreground">
      {i18nStore.t('settings.security.humanAuthSummary.labels.authMode')}
    </div>
    <div class="text-foreground mt-1 font-medium uppercase">{authMode}</div>
    {#if configuredMode && configuredMode !== authMode}
      <div class="text-muted-foreground">
        {i18nStore.t('settings.security.humanAuthSummary.labels.configuredMode', {
          mode: configuredMode,
        })}
      </div>
    {/if}
  </div>
  <div>
    <div class="text-muted-foreground">
      {i18nStore.t('settings.security.humanAuthSummary.labels.issuer')}
    </div>
    <div class="text-foreground mt-1 font-mono break-all">
      {issuerURL || i18nStore.t('settings.security.humanAuthSummary.messages.notConfigured')}
    </div>
  </div>
  <div>
    <div class="text-muted-foreground">
      {i18nStore.t(
        user
          ? 'settings.security.humanAuthSummary.labels.currentUser'
          : 'settings.security.humanAuthSummary.labels.localPrincipal',
      )}
    </div>
    <div class="text-foreground mt-1 font-medium">
      {user?.displayName ||
        localPrincipal ||
        i18nStore.t('settings.security.humanAuthSummary.messages.anonymous')}
    </div>
    {#if user?.primaryEmail}
      <div class="text-muted-foreground break-all">{user.primaryEmail}</div>
    {:else if localPrincipal}
      <div class="text-muted-foreground">
        {i18nStore.t('settings.security.humanAuthSummary.messages.localBootstrapHint')}
      </div>
    {/if}
  </div>
  <div>
    <div class="text-muted-foreground">
      {i18nStore.t('settings.security.humanAuthSummary.labels.bootstrapState')}
    </div>
    <div class="text-foreground mt-1">
      {bootstrapSummary || i18nStore.t('settings.security.humanAuthSummary.messages.noBootstrapAdmins')}
    </div>
  </div>
  <div>
    <div class="text-muted-foreground">
      {i18nStore.t('settings.security.humanAuthSummary.labels.sessionBoundary')}
    </div>
    <div class="text-foreground mt-1">
      {i18nStore.t('settings.security.humanAuthSummary.messages.sessionBoundaryDetails')}
    </div>
    <div class="text-muted-foreground">
      {publicExposureRisk === 'high'
        ? i18nStore.t('settings.security.humanAuthSummary.messages.publicExposure')
        : i18nStore.t('settings.security.humanAuthSummary.messages.tokensServerSide')}
    </div>
  </div>
</div>
