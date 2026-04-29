<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { pageTitle } from '$lib/i18n'
  import { goto } from '$app/navigation'
  import {
    buildLocalBootstrapRedeemPath,
    parseLocalBootstrapAuthorizationBundle,
  } from '$lib/features/auth/local-bootstrap'
  import type { HumanAuthSession } from '$lib/stores/auth.svelte'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import { LogIn, Terminal } from '@lucide/svelte'

  type LoginGatePageData = {
    returnTo: string
    authSession: HumanAuthSession
  }

  let { data }: { data: LoginGatePageData } = $props()
  let localBootstrapBundle = $state('')
  let localBootstrapError = $state('')

  const loginHref = $derived(
    `/api/v1/auth/oidc/start?return_to=${encodeURIComponent(data.returnTo)}`,
  )
  const availableAuthMethods = $derived(data.authSession.authCapabilities.availableAuthMethods)
  const supportsOIDC = $derived(availableAuthMethods.includes('oidc'))
  const supportsLocalBootstrap = $derived(availableAuthMethods.includes('local_bootstrap_link'))

  async function continueWithLocalBootstrap() {
    const bundle = parseLocalBootstrapAuthorizationBundle(localBootstrapBundle, data.returnTo)
    if (!bundle) {
      localBootstrapError = i18nStore.t('auth.localBootstrapParseError')
      return
    }
    localBootstrapError = ''
    await goto(buildLocalBootstrapRedeemPath(bundle))
  }
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('auth.loginPageTitle'), i18nStore.locale)}</title>
</svelte:head>

<div class="bg-background flex min-h-screen items-center justify-center px-6 py-12">
  <div class="w-full max-w-sm space-y-8">
    <!-- Header -->
    <div class="text-center">
      <img src="/favicon.svg" alt="" class="mx-auto mb-4 size-10" />
      <h1 class="text-2xl font-bold tracking-tight">{i18nStore.t('common.appName')}</h1>
      <p class="text-muted-foreground mt-2 text-sm">{i18nStore.t('auth.signInToContinue')}</p>
    </div>

    <!-- OIDC login -->
    {#if supportsOIDC}
      <div class="space-y-3">
        <a href={loginHref} class="block">
          <Button class="w-full gap-2" size="lg">
            <LogIn class="size-4" />
            {i18nStore.t('auth.continueWithOIDC')}
          </Button>
        </a>
        {#if data.authSession.issuerURL}
          <p class="text-muted-foreground text-center text-xs">
            {data.authSession.issuerURL}
          </p>
        {/if}
      </div>
    {/if}

    <!-- Separator when both methods available -->
    {#if supportsOIDC && supportsLocalBootstrap}
      <div class="flex items-center gap-3">
        <div class="bg-border h-px flex-1"></div>
        <span class="text-muted-foreground text-xs">{i18nStore.t('auth.or')}</span>
        <div class="bg-border h-px flex-1"></div>
      </div>
    {/if}

    <!-- Local bootstrap -->
    {#if supportsLocalBootstrap}
      <div class="space-y-4">
        {#if !supportsOIDC}
          <div class="flex items-center justify-center gap-2">
            <Terminal class="text-muted-foreground size-4" />
            <span class="text-sm font-medium">{i18nStore.t('auth.localBootstrap')}</span>
          </div>
        {/if}

        <div class="bg-muted rounded-lg px-3 py-2.5 font-mono text-xs break-all">
          openase auth bootstrap create-link
        </div>

        <div class="space-y-2">
          <Textarea
            id="local-bootstrap-bundle"
            value={localBootstrapBundle}
            class="min-h-24 font-mono text-xs"
            placeholder={i18nStore.t('auth.pasteBundlePlaceholder')}
            oninput={(event) =>
              (localBootstrapBundle = (event.currentTarget as HTMLTextAreaElement).value)}
          />
          {#if localBootstrapError}
            <div class="text-destructive text-xs">{localBootstrapError}</div>
          {/if}
          <Button
            class="w-full"
            variant={supportsOIDC ? 'outline' : 'default'}
            onclick={() => void continueWithLocalBootstrap()}
          >
            {i18nStore.t('auth.authorize')}
          </Button>
        </div>
      </div>
    {/if}

    <!-- No methods available -->
    {#if !supportsOIDC && !supportsLocalBootstrap}
      <div
        class="text-destructive rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-center text-sm"
      >
        {i18nStore.t('auth.noAuthMethodAvailable')}
      </div>
    {/if}
  </div>
</div>
