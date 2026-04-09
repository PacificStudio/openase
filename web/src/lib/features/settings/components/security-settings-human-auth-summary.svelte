<script lang="ts">
  import type { HumanAuthUser } from '$lib/stores/auth.svelte'

  let {
    authMode,
    configuredMode = '',
    issuerURL = '',
    user = null,
    bootstrapSummary = '',
    publicExposureRisk = '',
    localPrincipal = '',
  }: {
    authMode: string
    configuredMode?: string
    issuerURL?: string
    user?: HumanAuthUser | null
    bootstrapSummary?: string
    publicExposureRisk?: string
    localPrincipal?: string
  } = $props()
</script>

<div class="bg-muted/30 grid gap-3 rounded-lg px-4 py-3 text-xs md:grid-cols-2 xl:grid-cols-5">
  <div>
    <div class="text-muted-foreground">Auth mode</div>
    <div class="text-foreground mt-1 font-medium uppercase">{authMode}</div>
    {#if configuredMode && configuredMode !== authMode}
      <div class="text-muted-foreground">Configured: {configuredMode}</div>
    {/if}
  </div>
  <div>
    <div class="text-muted-foreground">Issuer</div>
    <div class="text-foreground mt-1 font-mono break-all">{issuerURL || 'Not configured'}</div>
  </div>
  <div>
    <div class="text-muted-foreground">{user ? 'Current user' : 'Local principal'}</div>
    <div class="text-foreground mt-1 font-medium">
      {user?.displayName || localPrincipal || 'Anonymous'}
    </div>
    {#if user?.primaryEmail}
      <div class="text-muted-foreground break-all">{user.primaryEmail}</div>
    {:else if localPrincipal}
      <div class="text-muted-foreground">Local bootstrap keeps this admin path available.</div>
    {/if}
  </div>
  <div>
    <div class="text-muted-foreground">Bootstrap state</div>
    <div class="text-foreground mt-1">{bootstrapSummary || 'No bootstrap admins configured'}</div>
  </div>
  <div>
    <div class="text-muted-foreground">Session boundary</div>
    <div class="text-foreground mt-1">httpOnly cookie + CSRF header</div>
    <div class="text-muted-foreground">
      {publicExposureRisk === 'high'
        ? 'Public exposure detected: configure OIDC before wider rollout.'
        : 'OIDC tokens stay server-side.'}
    </div>
  </div>
</div>
