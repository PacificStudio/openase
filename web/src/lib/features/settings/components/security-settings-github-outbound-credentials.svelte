<script lang="ts">
  import type { SecuritySettingsResponse } from '$lib/api/contracts'
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
      slot: security.github.organization,
    },
    {
      scope: 'project' as const,
      title: 'Project override',
      slot: security.github.project_override,
    },
  ])

  function effectiveStatusDot(slot: GitHubSlot): string {
    if (!slot.configured) return 'bg-slate-400'
    if (slot.probe.valid) return 'bg-emerald-500'
    if (slot.probe.state === 'error' || slot.probe.state === 'revoked') return 'bg-rose-500'
    return 'bg-amber-500'
  }

  function effectiveLabel(slot: GitHubSlot): string {
    if (!slot.configured) return 'Not configured'
    return slot.probe.state.replaceAll('_', ' ')
  }

  function scopeSourceLabel(slot: GitHubSlot): string {
    if (slot.scope === 'organization') return 'Organization'
    if (slot.scope === 'project') return 'Project override'
    return ''
  }

  function displayLogin(slot: GitHubSlot): string {
    const login = (slot.probe as typeof slot.probe & { login?: string }).login?.trim()
    if (!login) return ''
    return login.startsWith('@') ? login : `@${login}`
  }

  function formatCheckedAt(value: string | null | undefined): string {
    if (!value) return ''
    const parsed = new Date(value)
    if (Number.isNaN(parsed.getTime())) return value
    return parsed.toLocaleString()
  }
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <ShieldCheck class="text-muted-foreground size-4" />
    <h3 class="text-sm font-semibold">GitHub outbound credentials</h3>
  </div>

  <!-- Effective credential status bar -->
  <div class="bg-muted/40 flex flex-wrap items-center gap-x-4 gap-y-1 rounded-lg px-4 py-3">
    <div class="flex items-center gap-2">
      <span class="text-muted-foreground text-xs font-medium tracking-wide uppercase"
        >Effective credential</span
      >
      <span
        class={`inline-block size-2 rounded-full ${effectiveStatusDot(security.github.effective)}`}
      ></span>
      <span class="text-sm font-medium capitalize">{effectiveLabel(security.github.effective)}</span
      >
    </div>

    {#if security.github.effective.configured}
      <span class="text-muted-foreground text-xs">
        {scopeSourceLabel(security.github.effective)}
        {#if security.github.effective.source}
          · {security.github.effective.source.replaceAll('_', ' ')}
        {/if}
      </span>
      {#if displayLogin(security.github.effective)}
        <span class="text-muted-foreground text-xs">
          User {displayLogin(security.github.effective)}
        </span>
      {/if}
      <code class="text-muted-foreground text-xs">{security.github.effective.token_preview}</code>
      {#if security.github.effective.probe.permissions.length}
        <span class="text-muted-foreground text-xs">
          {security.github.effective.probe.permissions.join(', ')}
        </span>
      {:else}
        <span class="text-muted-foreground text-xs">No scopes reported</span>
      {/if}
      {#if formatCheckedAt(security.github.effective.probe.checked_at)}
        <span class="text-muted-foreground text-xs">
          Checked {formatCheckedAt(security.github.effective.probe.checked_at)}
        </span>
      {/if}
    {/if}

    {#if security.github.effective.probe.last_error}
      <span class="text-destructive text-xs">
        {security.github.effective.probe.last_error}
      </span>
    {/if}
  </div>

  <!-- Scope cards -->
  <div class="grid gap-4 lg:grid-cols-2">
    {#each scopeCards as card (card.scope)}
      <GitHubCredentialScopeCard
        scope={card.scope}
        title={card.title}
        slot={card.slot}
        tokenValue={manualTokens[card.scope]}
        {actionKey}
        organizationConfigured={security.github.organization.configured}
        {onAction}
        onTokenChange={onManualTokenChange}
      />
    {/each}
  </div>

  <!-- Device Flow deferred -->
  <p class="text-muted-foreground text-xs">
    <span class="font-medium">Device Flow</span> — {deviceFlowSummary}
  </p>
</div>
