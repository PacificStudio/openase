<script lang="ts">
  import { goto } from '$app/navigation'
  import { onMount } from 'svelte'

  import { ApiError } from '$lib/api/client'
  import { redeemLocalBootstrapAuthorization } from '$lib/api/auth'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()

  let pending = $state(false)
  let status = $state<'idle' | 'success' | 'error'>('idle')
  let message = $state('Authorize this browser as the local instance admin.')

  async function redeem() {
    if (!data.requestID || !data.code || !data.nonce) {
      status = 'error'
      message =
        'This link is incomplete. Generate a fresh local bootstrap authorization link from the CLI.'
      return
    }

    pending = true
    status = 'idle'
    message = 'Authorizing this browser session...'
    try {
      await redeemLocalBootstrapAuthorization({
        requestID: data.requestID,
        code: data.code,
        nonce: data.nonce,
      })
      status = 'success'
      message = 'Authorization succeeded. Redirecting into OpenASE...'
      await goto(data.returnTo, { replaceState: true })
    } catch (error) {
      status = 'error'
      message =
        error instanceof ApiError
          ? error.detail || 'Local bootstrap authorization failed.'
          : 'Local bootstrap authorization failed.'
    } finally {
      pending = false
    }
  }

  onMount(() => {
    void redeem()
  })
</script>

<svelte:head>
  <title>Local Authorization - OpenASE</title>
</svelte:head>

<div class="bg-background flex min-h-screen items-center justify-center px-6 py-12">
  <Card.Root class="w-full max-w-xl gap-6 border shadow-sm">
    <Card.Header class="gap-2">
      <Card.Title class="text-2xl">Local Authorization</Card.Title>
      <Card.Description>
        Complete a one-time local bootstrap authorization before entering the control plane.
      </Card.Description>
    </Card.Header>

    <Card.Content class="space-y-4">
      <div class="bg-muted/40 border-border rounded-xl border px-4 py-3 text-sm">
        <div class="text-muted-foreground">Status</div>
        <div class="text-foreground mt-1 font-medium">{message}</div>
      </div>

      <div class="flex gap-3">
        <Button onclick={() => void redeem()} disabled={pending} class="w-full">
          {pending ? 'Authorizing...' : 'Retry authorization'}
        </Button>
      </div>

      <p class="text-muted-foreground text-xs">
        The URL carries only short-lived, single-use authorization material. OpenASE stores the
        resulting browser session in an httpOnly cookie after redemption.
      </p>
    </Card.Content>
  </Card.Root>
</div>
