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
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Label } from '$ui/label'
  import { Separator } from '$ui/separator'
  import { Textarea } from '$ui/textarea'
  import {
    KeyRound,
    LoaderCircle,
    LockKeyhole,
    RefreshCw,
    ShieldCheck,
    Trash2,
    Upload,
    Webhook,
  } from '@lucide/svelte'

  type Security = SecuritySettingsResponse['security']
  type GitHubScope = 'organization' | 'project'
  type GitHubSlot = Security['github']['organization']

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

  const deviceFlowSummary = $derived(
    security?.deferred.find((item) => item.key === 'github-device-flow')?.summary ??
      'GitHub Device Flow remains deferred.',
  )

  const scopeCards = $derived(
    security
      ? [
          {
            scope: 'organization' as const,
            title: 'Organization default',
            description:
              'Shared platform-managed GH_TOKEN used by default across this organization unless a project override is configured.',
            slot: security.github.organization,
          },
          {
            scope: 'project' as const,
            title: 'Project override',
            description:
              'Project-specific GH_TOKEN that shadows the organization default for this project only.',
            slot: security.github.project_override,
          },
        ]
      : [],
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

  function scopeLabel(scope: string | undefined) {
    if (scope === 'organization') return 'Organization'
    if (scope === 'project') return 'Project override'
    return 'Missing'
  }

  function probeTone(slot: GitHubSlot) {
    if (!slot.configured) return 'secondary'
    if (slot.probe.valid) return 'outline'
    if (slot.probe.state === 'error' || slot.probe.state === 'revoked') return 'destructive'
    return 'secondary'
  }

  function probeLabel(slot: GitHubSlot) {
    if (!slot.configured) return 'Missing'
    return slot.probe.state.replaceAll('_', ' ')
  }

  function formatCheckedAt(value: string | null | undefined) {
    if (!value) return 'Not checked yet'
    const parsed = new Date(value)
    if (Number.isNaN(parsed.getTime())) return value
    return parsed.toLocaleString()
  }

  function slotHint(scope: GitHubScope, slot: GitHubSlot) {
    if (scope === 'project' && !slot.configured && security?.github.organization.configured) {
      return 'No project override is configured. This project currently falls back to the organization default.'
    }
    if (!slot.configured) {
      return 'No platform-managed credential is stored at this scope yet.'
    }
    return 'This scope is stored in platform secret storage and immediately probed after save or import.'
  }

  function isBusy(key: string) {
    return actionKey === key
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
        toastStore.success(`Saved ${scopeLabel(scope).toLowerCase()} GitHub credential.`)
      } else if (action === 'import') {
        payload = await importGitHubOutboundCredentialFromGHCLI(projectId, { scope })
        toastStore.success(`Imported ${scopeLabel(scope).toLowerCase()} credential from gh.`)
      } else if (action === 'retest') {
        payload = await retestGitHubOutboundCredential(projectId, { scope })
        toastStore.success(`Retested ${scopeLabel(scope).toLowerCase()} GitHub credential.`)
      } else {
        payload = await deleteGitHubOutboundCredential(projectId, scope)
        manualTokens[scope] = ''
        toastStore.success(`Deleted ${scopeLabel(scope).toLowerCase()} GitHub credential.`)
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
      <Card.Root class="xl:col-span-2">
        <Card.Header>
          <Card.Title class="flex items-center gap-2">
            <ShieldCheck class="size-4" />
            GitHub outbound credentials
          </Card.Title>
          <Card.Description>
            One platform-managed GH_TOKEN is resolved per project. Project overrides shadow the
            organization default, and every save/import is immediately probed for validity and repo
            access.
          </Card.Description>
        </Card.Header>
        <Card.Content class="space-y-6">
          <div class="bg-muted/30 border-border rounded-xl border p-4">
            <div class="flex flex-wrap items-center gap-2">
              <div class="text-sm font-medium">Effective credential</div>
              <Badge variant={probeTone(security.github.effective)}>
                {probeLabel(security.github.effective)}
              </Badge>
              <Badge variant="secondary">{scopeLabel(security.github.effective.scope)}</Badge>
              {#if security.github.effective.source}
                <Badge variant="outline">{security.github.effective.source}</Badge>
              {/if}
            </div>

            <div
              class="text-muted-foreground mt-3 grid gap-3 text-sm md:grid-cols-2 xl:grid-cols-4"
            >
              <div>
                <div class="text-foreground font-medium">Token preview</div>
                <div>{security.github.effective.token_preview || 'Not configured'}</div>
              </div>
              <div>
                <div class="text-foreground font-medium">Repo access</div>
                <div>{security.github.effective.probe.repo_access.replaceAll('_', ' ')}</div>
              </div>
              <div>
                <div class="text-foreground font-medium">Checked at</div>
                <div>{formatCheckedAt(security.github.effective.probe.checked_at)}</div>
              </div>
              <div>
                <div class="text-foreground font-medium">Permissions</div>
                <div>
                  {security.github.effective.probe.permissions.length
                    ? security.github.effective.probe.permissions.join(', ')
                    : 'No scopes reported'}
                </div>
              </div>
            </div>

            {#if security.github.effective.probe.last_error}
              <div class="text-destructive mt-3 text-sm">
                Last error: {security.github.effective.probe.last_error}
              </div>
            {/if}
          </div>

          <div class="grid gap-4 lg:grid-cols-2">
            {#each scopeCards as card (card.scope)}
              <div class="border-border bg-card rounded-2xl border p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="space-y-1">
                    <div class="flex flex-wrap items-center gap-2">
                      <div class="font-medium">{card.title}</div>
                      <Badge variant={probeTone(card.slot)}>{probeLabel(card.slot)}</Badge>
                      {#if card.slot.source}
                        <Badge variant="outline">{card.slot.source}</Badge>
                      {/if}
                    </div>
                    <p class="text-muted-foreground text-sm">{card.description}</p>
                  </div>
                </div>

                <div class="text-muted-foreground mt-4 space-y-2 text-sm">
                  <p>{slotHint(card.scope, card.slot)}</p>
                  <p>Token preview: {card.slot.token_preview || 'Not configured'}</p>
                  <p>Repo access: {card.slot.probe.repo_access.replaceAll('_', ' ')}</p>
                  <p>Checked at: {formatCheckedAt(card.slot.probe.checked_at)}</p>
                  {#if card.slot.probe.permissions.length}
                    <p>Permissions: {card.slot.probe.permissions.join(', ')}</p>
                  {/if}
                  {#if card.slot.probe.last_error}
                    <p class="text-destructive">Last error: {card.slot.probe.last_error}</p>
                  {/if}
                </div>

                <div class="mt-4 space-y-2">
                  <Label for={`github-token-${card.scope}`}>
                    {card.slot.configured ? 'Rotate token' : 'Paste token'}
                  </Label>
                  <Textarea
                    id={`github-token-${card.scope}`}
                    bind:value={manualTokens[card.scope]}
                    rows={3}
                    placeholder="ghu_xxx or github_pat_xxx"
                    disabled={actionKey !== ''}
                  />
                </div>

                <div class="mt-4 flex flex-wrap gap-2">
                  <Button
                    onclick={() => mutateScope(card.scope, 'save')}
                    disabled={actionKey !== ''}
                  >
                    {#if isBusy(`${card.scope}:save`)}
                      <LoaderCircle class="mr-2 size-4 animate-spin" />
                    {:else}
                      <KeyRound class="mr-2 size-4" />
                    {/if}
                    {card.slot.configured ? 'Save rotation' : 'Save token'}
                  </Button>

                  <Button
                    variant="outline"
                    onclick={() => mutateScope(card.scope, 'import')}
                    disabled={actionKey !== ''}
                  >
                    {#if isBusy(`${card.scope}:import`)}
                      <LoaderCircle class="mr-2 size-4 animate-spin" />
                    {:else}
                      <Upload class="mr-2 size-4" />
                    {/if}
                    Import from gh
                  </Button>

                  <Button
                    variant="outline"
                    onclick={() => mutateScope(card.scope, 'retest')}
                    disabled={!card.slot.configured || actionKey !== ''}
                  >
                    {#if isBusy(`${card.scope}:retest`)}
                      <LoaderCircle class="mr-2 size-4 animate-spin" />
                    {:else}
                      <RefreshCw class="mr-2 size-4" />
                    {/if}
                    Retest
                  </Button>

                  <Button
                    variant="destructive"
                    onclick={() => mutateScope(card.scope, 'delete')}
                    disabled={!card.slot.configured || actionKey !== ''}
                  >
                    {#if isBusy(`${card.scope}:delete`)}
                      <LoaderCircle class="mr-2 size-4 animate-spin" />
                    {:else}
                      <Trash2 class="mr-2 size-4" />
                    {/if}
                    Delete
                  </Button>
                </div>
              </div>
            {/each}
          </div>

          <div
            class="text-muted-foreground border-border rounded-xl border border-dashed p-4 text-sm"
          >
            <div class="text-foreground font-medium">Device Flow</div>
            <p class="mt-2">{deviceFlowSummary}</p>
          </div>
        </Card.Content>
      </Card.Root>

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
