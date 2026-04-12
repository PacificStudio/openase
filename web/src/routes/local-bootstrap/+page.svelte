<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { pageTitle } from '$lib/i18n'
  import { goto } from '$app/navigation'
  import { onMount } from 'svelte'

  import {
    describeLocalBootstrapRedeemError,
    redeemLocalBootstrapBrowserSession,
  } from '$lib/features/auth/local-bootstrap'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()

  let pending = $state(false)
  let message = $state(i18nStore.t('auth.localAuthorizationInitialStatus'))

  async function redeem() {
    if (!data.requestID || !data.code || !data.nonce) {
      message = i18nStore.t('auth.localAuthorizationIncomplete')
      return
    }

    pending = true
    message = i18nStore.t('auth.localAuthorizationAuthorizing')
    try {
      await redeemLocalBootstrapBrowserSession({
        requestID: data.requestID,
        code: data.code,
        nonce: data.nonce,
      })
      message = i18nStore.t('auth.localAuthorizationSucceeded')
      await goto(data.returnTo, { replaceState: true })
    } catch (error) {
      message = describeLocalBootstrapRedeemError(error)
    } finally {
      pending = false
    }
  }

  onMount(() => {
    void redeem()
  })
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('auth.localAuthorizationPageTitle'), i18nStore.locale)}</title>
</svelte:head>

<div class="bg-background flex min-h-screen items-center justify-center px-6 py-12">
  <Card.Root class="w-full max-w-xl gap-6 border shadow-sm">
    <Card.Header class="gap-2">
      <Card.Title class="text-2xl">{i18nStore.t('auth.localAuthorizationTitle')}</Card.Title>
      <Card.Description>
        {i18nStore.t('auth.localAuthorizationDescription')}
      </Card.Description>
    </Card.Header>

    <Card.Content class="space-y-4">
      <div class="bg-muted/40 border-border rounded-xl border px-4 py-3 text-sm">
        <div class="text-muted-foreground">{i18nStore.t('common.status')}</div>
        <div class="text-foreground mt-1 font-medium">{message}</div>
      </div>

      <div class="flex gap-3">
        <Button onclick={() => void redeem()} disabled={pending} class="w-full">
          {pending
            ? i18nStore.t('auth.localAuthorizationRetrying')
            : i18nStore.t('auth.localAuthorizationRetry')}
        </Button>
      </div>

      <p class="text-muted-foreground text-xs">
        {i18nStore.t('auth.localAuthorizationNotice')}
      </p>
    </Card.Content>
  </Card.Root>
</div>
