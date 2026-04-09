<script lang="ts">
  import { goto } from '$app/navigation'
  import {
    buildLocalBootstrapRedeemPath,
    parseLocalBootstrapAuthorizationBundle,
  } from '$lib/features/auth/local-bootstrap'
  import type { HumanAuthSession } from '$lib/stores/auth.svelte'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Textarea } from '$ui/textarea'

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
  const currentAuthMethodLabel = $derived(
    data.authSession.authCapabilities.currentAuthMethod === 'local_bootstrap_link'
      ? 'Local bootstrap link'
      : data.authSession.authCapabilities.currentAuthMethod === 'oidc'
        ? 'OIDC login'
        : 'Unknown',
  )

  async function continueWithLocalBootstrap() {
    const bundle = parseLocalBootstrapAuthorizationBundle(localBootstrapBundle, data.returnTo)
    if (!bundle) {
      localBootstrapError =
        'Paste the CLI JSON, text output, or URL from `openase auth bootstrap create-link`.'
      return
    }
    localBootstrapError = ''
    await goto(buildLocalBootstrapRedeemPath(bundle))
  }
</script>

<svelte:head>
  <title>Login - OpenASE</title>
</svelte:head>

<div class="bg-background flex min-h-screen items-center justify-center px-6 py-12">
  <Card.Root class="w-full max-w-2xl gap-6 border shadow-sm">
    <Card.Header class="gap-2">
      <Card.Title class="text-2xl">Authorize Browser</Card.Title>
      <Card.Description>
        OpenASE always routes browser access through an auth gate before entering the control plane.
      </Card.Description>
    </Card.Header>

    <Card.Content class="space-y-4">
      <div
        class="bg-muted/40 border-border grid gap-3 rounded-xl border px-4 py-3 text-sm md:grid-cols-2"
      >
        <div>
          <div class="text-muted-foreground">Current interactive auth</div>
          <div class="text-foreground mt-1 font-medium">{currentAuthMethodLabel}</div>
        </div>
        <div>
          <div class="text-muted-foreground">Available methods</div>
          <div class="text-foreground mt-1 font-medium">
            {#if availableAuthMethods.length > 0}
              {availableAuthMethods.join(', ')}
            {:else}
              None
            {/if}
          </div>
        </div>
        {#if data.authSession.issuerURL}
          <div class="md:col-span-2">
            <div class="text-muted-foreground">Issuer</div>
            <div class="text-foreground mt-1 font-mono text-xs break-all">
              {data.authSession.issuerURL}
            </div>
          </div>
        {/if}
      </div>

      {#if supportsOIDC}
        <div class="space-y-3 rounded-xl border px-4 py-4">
          <div>
            <div class="text-sm font-semibold">OIDC sign-in</div>
            <p class="text-muted-foreground mt-1 text-xs leading-5">
              Continue with the configured identity provider. Local bootstrap links are not offered
              while OIDC is the active browser auth method.
            </p>
          </div>

          <a href={loginHref} class="block">
            <Button class="w-full">Continue with OIDC</Button>
          </a>
        </div>
      {/if}

      {#if supportsLocalBootstrap}
        <div class="space-y-4 rounded-xl border px-4 py-4">
          <div>
            <div class="text-sm font-semibold">Local bootstrap authorization</div>
            <p class="text-muted-foreground mt-1 text-xs leading-5">
              Generate a short-lived authorization bundle on the machine running OpenASE, then paste
              the CLI output here to redeem a browser session through this browser's current
              entrypoint.
            </p>
          </div>

          <div class="bg-muted rounded-lg px-3 py-3 font-mono text-xs break-all">
            openase auth bootstrap create-link --return-to {data.returnTo}
          </div>

          <div class="space-y-2">
            <label class="text-xs font-medium" for="local-bootstrap-bundle">
              Paste local bootstrap bundle
            </label>
            <div class="space-y-2">
              <Textarea
                id="local-bootstrap-bundle"
                value={localBootstrapBundle}
                class="min-h-32 font-mono text-xs"
                placeholder={'{\n  "request_id": "...",\n  "code": "...",\n  "nonce": "..."\n}'}
                oninput={(event) =>
                  (localBootstrapBundle = (event.currentTarget as HTMLTextAreaElement).value)}
              />
              <Button class="w-full sm:w-auto" onclick={() => void continueWithLocalBootstrap()}>
                Continue with local bootstrap
              </Button>
            </div>
            {#if localBootstrapError}
              <div class="text-destructive text-xs">{localBootstrapError}</div>
            {/if}
            <div class="text-muted-foreground text-xs leading-5">
              Accepted formats: the CLI JSON output, the text-mode URL output, or a copied
              `/local-bootstrap?...` link from another entrypoint. OpenASE will redeem the
              short-lived request through this browser's current origin.
            </div>
          </div>
        </div>
      {/if}

      {#if !supportsOIDC && !supportsLocalBootstrap}
        <div class="text-destructive rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm">
          No browser authorization method is currently available.
        </div>
      {/if}

      <p class="text-muted-foreground text-xs">
        After sign-in, OpenASE stores only an httpOnly session cookie in the browser.
      </p>
    </Card.Content>
  </Card.Root>
</div>
