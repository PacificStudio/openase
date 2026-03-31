<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { SecuritySettingsResponse } from '$lib/api/contracts'
  import {
    deleteGitHubOutboundCredential,
    getSecuritySettings,
    importGitHubOutboundCredentialFromGHCLI,
    retestGitHubOutboundCredential,
    saveGitHubOutboundCredential,
  } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import * as Card from '$ui/card'
  import { Separator } from '$ui/separator'
  import { KeyRound, LockKeyhole, Webhook } from '@lucide/svelte'

  import GitHubOutboundCredentialsPanel from './security-settings-github-outbound-credentials.svelte'

  type Security = SecuritySettingsResponse['security']
  type GitHubScope = 'organization' | 'project'

  let security = $state<Security | null>(null)
  let loading = $state(false)
  let error = $state('')
  let actionKey = $state('')
  let manualTokens = $state<Record<GitHubScope, string>>({
    organization: '',
    project: '',
  })

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
        error = formatError(caughtError, 'Failed to load security settings.')
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

  function formatError(caughtError: unknown, fallback: string) {
    return caughtError instanceof ApiError ? caughtError.detail : fallback
  }

  function scopeLabel(scope: GitHubScope) {
    return scope === 'organization' ? 'organization' : 'project override'
  }

  function handleManualTokenChange(scope: GitHubScope, value: string) {
    manualTokens[scope] = value
  }

  async function mutateScope(scope: GitHubScope, action: 'save' | 'import' | 'retest' | 'delete') {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    const key = `${scope}:${action}`
    actionKey = key
    error = ''

    try {
      let payload: SecuritySettingsResponse
      if (action === 'save') {
        const token = manualTokens[scope].trim()
        if (!token) {
          toastStore.error('GitHub token is required.')
          return
        }
        payload = await saveGitHubOutboundCredential(projectId, { scope, token })
        manualTokens[scope] = ''
        toastStore.success(`Saved ${scopeLabel(scope)} GitHub credential.`)
      } else if (action === 'import') {
        payload = await importGitHubOutboundCredentialFromGHCLI(projectId, { scope })
        toastStore.success(`Imported ${scopeLabel(scope)} credential from gh.`)
      } else if (action === 'retest') {
        payload = await retestGitHubOutboundCredential(projectId, { scope })
        toastStore.success(`Retested ${scopeLabel(scope)} GitHub credential.`)
      } else {
        payload = await deleteGitHubOutboundCredential(projectId, scope)
        manualTokens[scope] = ''
        toastStore.success(`Deleted ${scopeLabel(scope)} GitHub credential.`)
      }
      security = payload.security
    } catch (caughtError) {
      const message = formatError(caughtError, 'GitHub credential update failed.')
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Security</h2>
    <p class="text-muted-foreground mt-1 max-w-3xl text-sm">
      Manage the platform-managed GH_TOKEN control plane alongside the runtime boundaries already
      enforced by OpenASE.
    </p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading security settings…</div>
  {:else if error}
    <div class="text-destructive text-sm">{error}</div>
  {:else if security}
    <div class="grid gap-6 xl:grid-cols-[minmax(0,1.35fr),minmax(0,1fr)]">
      <GitHubOutboundCredentialsPanel
        {security}
        {actionKey}
        {manualTokens}
        onAction={mutateScope}
        onManualTokenChange={handleManualTokenChange}
      />

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

      <div class="space-y-6">
        <Card.Root>
          <Card.Header>
            <Card.Title class="flex items-center gap-2">
              <Webhook class="size-4" />
              Inbound webhooks
            </Card.Title>
            <Card.Description>
              Legacy GitHub delivery verification remains separate from the outbound GH_TOKEN
              control plane.
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
              <LockKeyhole class="size-4" />
              Secret hygiene
            </Card.Title>
            <Card.Description>
              Security-sensitive response surfaces stay redacted even while operators manage
              platform credentials.
            </Card.Description>
          </Card.Header>
          <Card.Content>
            <div class="border-border bg-muted/20 rounded-lg border p-4 text-sm">
              <div class="font-medium">Notification channel configs</div>
              <p class="text-muted-foreground mt-2">
                Current state:{' '}
                {security.secret_hygiene.notification_channel_configs_redacted
                  ? 'Secrets are redacted'
                  : 'Secrets may be exposed'}
              </p>
            </div>
          </Card.Content>
        </Card.Root>
      </div>
    </div>
  {/if}
</div>
