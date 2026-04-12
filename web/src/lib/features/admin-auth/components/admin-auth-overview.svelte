<script lang="ts">
  import type { HumanAuthUser } from '$lib/stores/auth.svelte'
  import { authStore } from '$lib/stores/auth.svelte'
  import type { SecurityAuthSettings } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Separator } from '$ui/separator'
  import { AlertTriangle, Shield, ShieldCheck } from '@lucide/svelte'
  import { adminAuthT } from './i18n'

  let {
    auth,
    user = null,
  }: {
    auth: SecurityAuthSettings
    user?: HumanAuthUser | null
  } = $props()

  function modeLabel(mode: SecurityAuthSettings['active_mode']) {
    return mode === 'oidc'
      ? adminAuthT('adminAuth.overview.modeLabel.oidc')
      : adminAuthT('adminAuth.overview.modeLabel.disabled')
  }

  const currentAuthMethodLabel = $derived(
    authStore.currentAuthMethod === 'local_bootstrap_link'
      ? adminAuthT('adminAuth.overview.methods.localBootstrapLink')
      : authStore.currentAuthMethod === 'oidc'
        ? adminAuthT('adminAuth.overview.methods.oidcLogin')
        : auth.active_mode === 'oidc'
          ? adminAuthT('adminAuth.overview.methods.oidcLogin')
          : adminAuthT('adminAuth.overview.methods.localBootstrapLink'),
  )

  const isOIDCActive = $derived(auth.active_mode === 'oidc')
</script>

<div class="border-border bg-card rounded-2xl border">
  <!-- High-risk alert -->
  {#if auth.public_exposure_risk === 'high'}
    <div class="rounded-t-2xl border-b border-amber-300 bg-amber-50 px-6 py-4 text-amber-950">
      <div class="flex items-start gap-3">
        <AlertTriangle class="mt-0.5 size-5 shrink-0 text-amber-600" />
        <div class="space-y-1">
        <div class="text-sm font-semibold">
          {adminAuthT('adminAuth.overview.publicExposureRisk')}
        </div>
          {#each auth.warnings as warning (warning)}
            <p class="text-xs leading-relaxed">{warning}</p>
          {/each}
        </div>
      </div>
    </div>
  {/if}

  <div class="p-6">
    <!-- Primary status: large auth mode display -->
    <div class="flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between">
      <div class="flex items-start gap-4">
        <div
          class={`flex size-12 items-center justify-center rounded-xl ${isOIDCActive ? 'bg-emerald-100 text-emerald-700' : 'bg-muted text-muted-foreground'}`}
        >
          {#if isOIDCActive}
            <ShieldCheck class="size-6" />
          {:else}
            <Shield class="size-6" />
          {/if}
        </div>
        <div>
          <div class="text-2xl font-bold tracking-tight uppercase">{modeLabel(auth.active_mode)}</div>
          <div class="text-muted-foreground mt-0.5 text-sm">{currentAuthMethodLabel}</div>
        </div>
      </div>
      <div class="flex flex-wrap gap-2">
        <Badge variant={isOIDCActive ? 'default' : 'secondary'}>
          {modeLabel(auth.active_mode)}
        </Badge>
        {#if auth.configured_mode !== auth.active_mode}
          <Badge variant="outline">
            {adminAuthT('adminAuth.overview.draftLabel', {
              modeLabel: modeLabel(auth.configured_mode),
            })}
          </Badge>
        {/if}
      </div>
    </div>

    <Separator class="my-5" />

    <!-- Key info grid -->
    <div class="grid gap-x-8 gap-y-4 sm:grid-cols-3">
      <div>
        <div class="text-muted-foreground text-xs font-medium">
          {adminAuthT('adminAuth.diagnostics.issuerLabel')}
        </div>
        <div class="mt-1 font-mono text-sm break-all">
          {auth.issuer_url ||
            adminAuthT('adminAuth.diagnostics.notConfigured')}
        </div>
      </div>
      <div>
        <div class="text-muted-foreground text-xs font-medium">
          {user
            ? adminAuthT('adminAuth.overview.currentUser')
            : adminAuthT('adminAuth.overview.localPrincipal')}
        </div>
        <div class="mt-1 text-sm font-medium">
          {user?.displayName ||
            auth.local_principal ||
            adminAuthT('adminAuth.overview.anonymous')}
        </div>
        {#if user?.primaryEmail}
          <div class="text-muted-foreground mt-0.5 text-xs break-all">{user.primaryEmail}</div>
        {/if}
      </div>
      <div>
        <div class="text-muted-foreground text-xs font-medium">
          {adminAuthT('adminAuth.overview.bootstrap')}
        </div>
        <div class="text-foreground mt-1 text-sm leading-relaxed">
          {auth.bootstrap_state.summary ||
            adminAuthT('adminAuth.overview.noAdminsConfigured')}
        </div>
      </div>
    </div>
  </div>

  <!-- Bootstrap link hint -->
  {#if currentAuthMethodLabel === 'Local bootstrap link'}
    <div class="border-t px-6 py-4">
      <details class="group">
        <summary
          class="text-muted-foreground hover:text-foreground flex cursor-pointer items-center gap-1.5 text-xs select-none"
        >
          <span
            class="border-muted-foreground/40 group-open:border-foreground/40 flex size-3.5 items-center justify-center rounded-full border text-[9px] leading-none font-bold transition-colors"
            >?</span
          >
          {adminAuthT('adminAuth.overview.browserAccess')}
        </summary>
        <div class="bg-muted/50 mt-2 rounded-lg px-3 py-3">
          <code class="bg-muted block rounded px-2 py-1.5 font-mono text-xs">
            openase auth bootstrap create-link
          </code>
        </div>
      </details>
    </div>
  {/if}
</div>
