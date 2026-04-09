<script lang="ts">
  import { goto } from '$app/navigation'
  import { parseLocalBootstrapRedeemURL } from '$lib/features/auth/local-bootstrap'
  import { Input } from '$ui/input'
  import * as Card from '$ui/card'
  import { Button } from '$ui/button'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()
  let localBootstrapURL = $state('')
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

  async function openLocalBootstrapLink() {
    const destination = parseLocalBootstrapRedeemURL(localBootstrapURL, window.location.origin)
    if (!destination) {
      localBootstrapError =
        'Paste a full local bootstrap URL from `openase auth bootstrap create-link`.'
      return
    }
    localBootstrapError = ''
    await goto(destination)
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
        OpenASE always routes browser access through an auth gate before entering the control
        plane.
      </Card.Description>
    </Card.Header>

    <Card.Content class="space-y-4">
      <div class="bg-muted/40 border-border grid gap-3 rounded-xl border px-4 py-3 text-sm md:grid-cols-2">
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
              Generate a short-lived authorization link from the machine running OpenASE, then open
              it here to redeem a browser session.
            </p>
          </div>

          <div class="bg-muted rounded-lg px-3 py-3 font-mono text-xs break-all">
            openase auth bootstrap create-link --return-to {data.returnTo}
          </div>

          <div class="space-y-2">
            <label class="text-xs font-medium" for="local-bootstrap-url">
              Paste local bootstrap URL
            </label>
            <div class="flex flex-col gap-2 sm:flex-row">
              <Input
                id="local-bootstrap-url"
                value={localBootstrapURL}
                placeholder="http://127.0.0.1:19836/local-bootstrap?request_id=..."
                oninput={(event) =>
                  (localBootstrapURL = (event.currentTarget as HTMLInputElement).value)}
              />
              <Button class="sm:min-w-44" onclick={() => void openLocalBootstrapLink()}>
                Open authorization link
              </Button>
            </div>
            {#if localBootstrapError}
              <div class="text-destructive text-xs">{localBootstrapError}</div>
            {/if}
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
