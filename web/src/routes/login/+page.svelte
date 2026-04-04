<script lang="ts">
  import * as Card from '$ui/card'
  import { Button } from '$ui/button'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()

  const loginHref = $derived(
    `/api/v1/auth/oidc/start?return_to=${encodeURIComponent(data.returnTo)}`,
  )
</script>

<svelte:head>
  <title>Login - OpenASE</title>
</svelte:head>

<div class="bg-background flex min-h-screen items-center justify-center px-6 py-12">
  <Card.Root class="w-full max-w-lg gap-6 border shadow-sm">
    <Card.Header class="gap-2">
      <Card.Title class="text-2xl">Sign In</Card.Title>
      <Card.Description>
        OpenASE uses your organization identity provider for control-plane access.
      </Card.Description>
    </Card.Header>

    <Card.Content class="space-y-4">
      <div class="bg-muted/40 border-border rounded-xl border px-4 py-3 text-sm">
        <div class="text-muted-foreground">Auth mode</div>
        <div class="text-foreground mt-1 font-medium uppercase">{data.authSession.authMode}</div>
        {#if data.authSession.issuerURL}
          <div class="text-muted-foreground mt-3">Issuer</div>
          <div class="text-foreground mt-1 font-mono text-xs break-all">
            {data.authSession.issuerURL}
          </div>
        {/if}
      </div>

      <a href={loginHref} class="block">
        <Button class="w-full">Continue with OIDC</Button>
      </a>

      <p class="text-muted-foreground text-xs">
        After sign-in, OpenASE stores only an httpOnly session cookie in the browser.
      </p>
    </Card.Content>
  </Card.Root>
</div>
