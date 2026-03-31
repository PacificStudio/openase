<script lang="ts">
  import type { SecuritySettingsResponse } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import * as Card from '$ui/card'
  import { ShieldCheck } from '@lucide/svelte'

  import GitHubCredentialScopeCard from './security-settings-github-scope-card.svelte'

  type Security = SecuritySettingsResponse['security']
  type GitHubScope = 'organization' | 'project'
  type GitHubSlot = Security['github']['organization']

  let {
    security,
    actionKey,
    manualTokens,
    onAction,
    onManualTokenChange,
  }: {
    security: Security
    actionKey: string
    manualTokens: Record<GitHubScope, string>
    onAction: (scope: GitHubScope, action: 'save' | 'import' | 'retest' | 'delete') => void
    onManualTokenChange: (scope: GitHubScope, value: string) => void
  } = $props()

  const deviceFlowSummary = $derived(
    security.deferred.find((item) => item.key === 'github-device-flow')?.summary ??
      'GitHub Device Flow remains deferred.',
  )

  const scopeCards = $derived([
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
  ])

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
</script>

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

      <div class="text-muted-foreground mt-3 grid gap-3 text-sm md:grid-cols-2 xl:grid-cols-4">
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
        <GitHubCredentialScopeCard
          scope={card.scope}
          title={card.title}
          description={card.description}
          slot={card.slot}
          tokenValue={manualTokens[card.scope]}
          {actionKey}
          organizationConfigured={security.github.organization.configured}
          {onAction}
          onTokenChange={onManualTokenChange}
        />
      {/each}
    </div>

    <div class="text-muted-foreground border-border rounded-xl border border-dashed p-4 text-sm">
      <div class="text-foreground font-medium">Device Flow</div>
      <p class="mt-2">{deviceFlowSummary}</p>
    </div>
  </Card.Content>
</Card.Root>
