<script lang="ts">
  import type { HumanAuthUser } from '$lib/stores/auth.svelte'
  import { authStore } from '$lib/stores/auth.svelte'
  import type { SecurityAuthSettings } from '$lib/api/contracts'
  import { SecuritySettingsHumanAuthSummary } from '$lib/features/settings'
  import { Badge } from '$ui/badge'
  import { AlertTriangle } from '@lucide/svelte'

  let {
    auth,
    user = null,
  }: {
    auth: SecurityAuthSettings
    user?: HumanAuthUser | null
  } = $props()

  const currentAuthMethodLabel = $derived(
    authStore.currentAuthMethod === 'local_bootstrap_link'
      ? 'Local bootstrap link'
      : authStore.currentAuthMethod === 'oidc'
        ? 'OIDC login'
        : auth.active_mode === 'oidc'
          ? 'OIDC login'
          : 'Local bootstrap link',
  )
</script>

<div class="border-border bg-card space-y-4 rounded-2xl border p-5">
  <div class="flex flex-wrap items-center gap-2">
    <Badge variant="outline">Instance scope</Badge>
    <Badge variant={authStore.usesOIDC || auth.active_mode === 'oidc' ? 'default' : 'secondary'}>
      Current interactive auth: {currentAuthMethodLabel}
    </Badge>
    <Badge variant="outline">Active runtime: {auth.active_mode}</Badge>
    {#if auth.configured_mode !== auth.active_mode}
      <Badge variant="outline">Draft ready: {auth.configured_mode}</Badge>
    {/if}
    <Badge variant={auth.public_exposure_risk === 'high' ? 'destructive' : 'secondary'}>
      {auth.public_exposure_risk === 'high' ? 'Public exposure risk' : 'Local-ready posture'}
    </Badge>
  </div>

  <SecuritySettingsHumanAuthSummary
    authMode={auth.active_mode}
    configuredMode={auth.configured_mode}
    issuerURL={auth.issuer_url ?? ''}
    {user}
    bootstrapSummary={auth.bootstrap_state.summary}
    publicExposureRisk={auth.public_exposure_risk}
    localPrincipal={auth.local_principal}
  />

  <div class="grid gap-3 md:grid-cols-2">
    {#each auth.warnings as warning (warning)}
      <div
        class={`rounded-xl border px-3 py-2 text-xs leading-relaxed ${
          auth.public_exposure_risk === 'high'
            ? 'border-amber-300 bg-amber-50 text-amber-950'
            : 'border-sky-200 bg-sky-50 text-sky-950'
        }`}
      >
        <div class="flex items-start gap-2">
          <AlertTriangle class="mt-0.5 size-4 shrink-0" />
          <span>{warning}</span>
        </div>
      </div>
    {/each}
  </div>

  <div class="grid gap-3 md:grid-cols-2">
    <div class="rounded-xl border border-dashed p-3">
      <div class="text-muted-foreground text-xs">Configured session TTL</div>
      <div class="mt-1 text-sm font-medium">{auth.session_policy.session_ttl}</div>
    </div>
    <div class="rounded-xl border border-dashed p-3">
      <div class="text-muted-foreground text-xs">Idle session TTL</div>
      <div class="mt-1 text-sm font-medium">{auth.session_policy.session_idle_ttl}</div>
    </div>
    <div class="rounded-xl border border-dashed p-3 md:col-span-2">
      <div class="text-muted-foreground text-xs">Source of truth</div>
      <div class="mt-1 font-mono text-xs break-all">{auth.config_path || 'Not available'}</div>
    </div>
  </div>

  {#if currentAuthMethodLabel === 'Local bootstrap link'}
    <div class="rounded-xl border border-sky-200 bg-sky-50 px-4 py-3 text-xs text-sky-950">
      Browser access currently enters through the local bootstrap auth gate. Generate a one-time
      authorization bundle from the host machine with <code>openase auth bootstrap create-link</code
      >, then paste the CLI output into the browser gate.
    </div>
  {/if}

  <div class="space-y-2">
    <div class="text-sm font-semibold">Next steps</div>
    <ol class="list-inside list-decimal space-y-1 text-sm leading-relaxed">
      {#each auth.next_steps as step (step)}
        <li>{step}</li>
      {/each}
    </ol>
  </div>
</div>
