<script lang="ts">
  import type { SecuritySettingsResponse } from '$lib/api/contracts'
  import { organizationPath } from '$lib/stores/app-context'
  import { appStore } from '$lib/stores/app.svelte'
  import { ArrowRight, ShieldCheck } from '@lucide/svelte'

  import GitHubCredentialScopeCard from './security-settings-github-scope-card.svelte'

  type Security = SecuritySettingsResponse['security']
  type GitHubSlot = Security['github']['organization']

  let {
    security,
    actionKey,
    manualToken,
    onAction,
    onManualTokenChange,
  }: {
    security: Security
    actionKey: string
    manualToken: string
    onAction: (action: 'save' | 'import' | 'retest' | 'delete') => void
    onManualTokenChange: (value: string) => void
  } = $props()

  const orgAdminHref = $derived(
    appStore.currentOrg?.id
      ? `${organizationPath(appStore.currentOrg.id)}/admin/credentials`
      : null,
  )

  const deviceFlowSummary = $derived(
    security.deferred.find((item) => item.key === 'github-device-flow')?.summary ??
      'GitHub Device Flow remains deferred.',
  )

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
    if (slot.scope === 'organization') return 'Org default'
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

  function orgSlotStatusDot(): string {
    const slot = security.github.organization
    if (!slot.configured) return 'bg-slate-300'
    if (slot.probe.valid) return 'bg-emerald-500'
    if (slot.probe.state === 'error' || slot.probe.state === 'revoked') return 'bg-rose-500'
    return 'bg-amber-500'
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
        <span class="text-muted-foreground text-xs"
          >User {displayLogin(security.github.effective)}</span
        >
      {/if}
      <code class="text-muted-foreground text-xs">{security.github.effective.token_preview}</code>
      {#if security.github.effective.probe.permissions.length}
        <span class="text-muted-foreground text-xs"
          >{security.github.effective.probe.permissions.join(', ')}</span
        >
      {/if}
      {#if formatCheckedAt(security.github.effective.probe.checked_at)}
        <span class="text-muted-foreground text-xs"
          >Checked {formatCheckedAt(security.github.effective.probe.checked_at)}</span
        >
      {/if}
    {/if}
    {#if security.github.effective.probe.last_error}
      <span class="text-destructive text-xs">{security.github.effective.probe.last_error}</span>
    {/if}
  </div>

  <div class="grid gap-4 lg:grid-cols-2">
    <!-- Org default: read-only reference -->
    <div class="border-border bg-muted/20 rounded-lg border p-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class={`inline-block size-2 rounded-full ${orgSlotStatusDot()}`}></span>
          <span class="text-sm font-medium">Org default</span>
          <span class="text-muted-foreground text-xs capitalize">
            {security.github.organization.configured
              ? security.github.organization.probe.state.replaceAll('_', ' ')
              : 'Not configured'}
          </span>
        </div>
      </div>
      {#if security.github.organization.configured}
        <div class="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
          {#if (security.github.organization.probe as typeof security.github.organization.probe & { login?: string }).login}
            <div>
              <span class="text-muted-foreground">User</span>
              <div>
                @{(
                  security.github.organization
                    .probe as typeof security.github.organization.probe & { login?: string }
                ).login}
              </div>
            </div>
          {/if}
          <div>
            <span class="text-muted-foreground">Token</span>
            <div class="font-mono">{security.github.organization.token_preview}</div>
          </div>
          {#if security.github.organization.probe.permissions.length}
            <div class="col-span-2">
              <span class="text-muted-foreground">Permissions</span>
              <div>{security.github.organization.probe.permissions.join(', ')}</div>
            </div>
          {/if}
        </div>
      {:else}
        <p class="text-muted-foreground mt-2 text-xs">No org default configured.</p>
      {/if}
      {#if orgAdminHref}
        <a
          href={orgAdminHref}
          class="mt-3 inline-flex items-center gap-1 text-xs font-medium text-sky-600 hover:text-sky-700 dark:text-sky-400 dark:hover:text-sky-300"
        >
          Manage in org admin <ArrowRight class="size-3" />
        </a>
      {/if}
    </div>

    <!-- Project override: full edit card -->
    <GitHubCredentialScopeCard
      slot={security.github.project_override}
      tokenValue={manualToken}
      {actionKey}
      organizationConfigured={security.github.organization.configured}
      {onAction}
      onTokenChange={onManualTokenChange}
    />
  </div>

  <p class="text-muted-foreground text-xs">
    <span class="font-medium">Device Flow</span> — {deviceFlowSummary}
  </p>
</div>
