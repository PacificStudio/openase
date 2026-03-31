<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { SecuritySettingsResponse } from '$lib/api/contracts'
  import { getSecuritySettings } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import * as Card from '$ui/card'
  import { Separator } from '$ui/separator'
  import { KeyRound, LockKeyhole, ShieldCheck, Webhook } from '@lucide/svelte'

  let security = $state<SecuritySettingsResponse['security'] | null>(null)
  let loading = $state(false)
  let error = $state('')

  const signatureLabel = $derived(
    security?.webhooks.legacy_github_signature_required ? 'Required' : 'Optional until configured',
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      security = null
      error = ''
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const payload = await getSecuritySettings(projectId)
        if (cancelled) return
        security = payload.security
      } catch (caughtError) {
        if (cancelled) return
        security = null
        error =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load security settings.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Security</h2>
    <p class="text-muted-foreground mt-1 max-w-3xl text-sm">
      Runtime security boundaries and access controls.
    </p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading security settings…</div>
  {:else if error}
    <div class="text-destructive text-sm">{error}</div>
  {:else if security}
    <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr),minmax(0,1fr)]">
      <Card.Root>
        <Card.Header>
          <Card.Title class="flex items-center gap-2">
            <KeyRound class="size-4" />
            Agent runtime access
          </Card.Title>
          <Card.Description>
            Project agents authenticate to the platform with scoped short-lived tokens instead of
            inheriting ambient machine credentials.
          </Card.Description>
        </Card.Header>
        <Card.Content class="space-y-4">
          <div class="border-border bg-muted/20 rounded-lg border p-4 text-sm">
            <div class="font-medium">Delivery contract</div>
            <div class="text-muted-foreground mt-2 space-y-1">
              <p>Transport: {security.agent_tokens.transport}</p>
              <p>Environment variable: <code>{security.agent_tokens.environment_variable}</code></p>
              <p>Token prefix: <code>{security.agent_tokens.token_prefix}</code></p>
            </div>
          </div>

          <div class="space-y-2">
            <div class="text-sm font-medium">Default scopes</div>
            <div class="flex flex-wrap gap-2">
              {#each security.agent_tokens.default_scopes as scope (scope)}
                <code class="bg-muted inline-flex rounded px-2 py-1 text-xs">{scope}</code>
              {/each}
            </div>
          </div>

          <div class="space-y-2">
            <div class="text-sm font-medium">Project-level scopes the runtime can mint</div>
            <div class="flex flex-wrap gap-2">
              {#each security.agent_tokens.supported_project_scopes as scope (scope)}
                <code class="bg-muted inline-flex rounded px-2 py-1 text-xs">{scope}</code>
              {/each}
            </div>
          </div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header>
          <Card.Title class="flex items-center gap-2">
            <Webhook class="size-4" />
            Inbound webhooks
          </Card.Title>
          <Card.Description>
            The current settings surface documents the webhook routes that exist today and whether
            legacy GitHub delivery signatures are enforced in this deployment.
          </Card.Description>
        </Card.Header>
        <Card.Content class="space-y-4">
          <div class="border-border bg-muted/20 rounded-lg border p-4 text-sm">
            <div class="font-medium">Legacy GitHub route</div>
            <p class="text-muted-foreground mt-2">
              <code>{security.webhooks.legacy_github_endpoint}</code>
            </p>
            <p class="text-muted-foreground mt-2">Signature verification: {signatureLabel}</p>
          </div>

          <div class="border-border bg-muted/20 rounded-lg border p-4 text-sm">
            <div class="font-medium">Connector ingress route</div>
            <p class="text-muted-foreground mt-2">
              <code>{security.webhooks.connector_endpoint}</code>
            </p>
            <p class="text-muted-foreground mt-2">
              Connector-specific verification still lives with each registered inbound receiver.
            </p>
          </div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header>
          <Card.Title class="flex items-center gap-2">
            <ShieldCheck class="size-4" />
            Secret hygiene
          </Card.Title>
          <Card.Description>
            This boundary calls out the secret-handling behavior already exported by the current app
            surface.
          </Card.Description>
        </Card.Header>
        <Card.Content>
          <div class="border-border bg-muted/20 rounded-lg border p-4 text-sm">
            <div class="flex items-center gap-2 font-medium">
              <LockKeyhole class="size-4" />
              Response-safe notification configs
            </div>
            <p class="text-muted-foreground mt-2">
              Notification channel payloads return masked secrets instead of raw values.
            </p>
            <p class="text-muted-foreground mt-2">
              Current state:{' '}
              {security.secret_hygiene.notification_channel_configs_redacted
                ? 'Enabled'
                : 'Disabled'}
            </p>
          </div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header>
          <Card.Title>Explicitly deferred</Card.Title>
          <Card.Description>
            This settings slice is concrete, but it does not pretend the broader security control
            plane is already shipped.
          </Card.Description>
        </Card.Header>
        <Card.Content class="space-y-3">
          {#each security.deferred as item (item.key)}
            <div class="border-border bg-muted/20 rounded-lg border p-4 text-sm">
              <div class="font-medium">{item.title}</div>
              <p class="text-muted-foreground mt-2">{item.summary}</p>
            </div>
          {/each}
        </Card.Content>
      </Card.Root>
    </div>
  {/if}
</div>
