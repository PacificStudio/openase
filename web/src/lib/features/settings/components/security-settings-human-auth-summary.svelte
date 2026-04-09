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

<div class="bg-muted/30 rounded-lg border">
  <div class="grid gap-4 p-4 sm:grid-cols-2">
    <!-- Auth config -->
    <div class="space-y-3">
      <div>
        <div class="text-muted-foreground text-xs">Auth mode</div>
        <div class="text-foreground mt-1 font-semibold uppercase">{authMode}</div>
        {#if configuredMode && configuredMode !== authMode}
          <div class="text-muted-foreground mt-0.5 text-xs">Draft configured: {configuredMode}</div>
        {/if}
      </div>
      <div>
        <div class="text-muted-foreground text-xs">Issuer</div>
        <div class="text-foreground mt-1 font-mono text-xs break-all">
          {issuerURL || 'Not configured'}
        </div>
      </div>
    </div>

    <!-- Identity -->
    <div class="space-y-3">
      <div>
        <div class="text-muted-foreground text-xs">{user ? 'Current user' : 'Local principal'}</div>
        <div class="text-foreground mt-1 text-sm font-medium">
          {user?.displayName || localPrincipal || 'Anonymous'}
        </div>
        {#if user?.primaryEmail}
          <div class="text-muted-foreground text-xs break-all">{user.primaryEmail}</div>
        {:else if localPrincipal}
          <div class="text-muted-foreground mt-0.5 text-xs">
            Disabled mode keeps this local admin available.
          </div>
        {/if}
      </div>
      <div>
        <div class="text-muted-foreground text-xs">Bootstrap state</div>
        <div class="text-foreground mt-1 text-xs leading-relaxed">
          {bootstrapSummary || 'No bootstrap admins configured'}
        </div>
      </div>
    </div>
  </div>

  <!-- Session boundary footer -->
  <div
    class={`border-t px-4 py-2.5 text-xs ${
      publicExposureRisk === 'high'
        ? 'border-amber-200 bg-amber-50 text-amber-800'
        : 'text-muted-foreground'
    }`}
  >
    <span class="font-medium">Session boundary:</span>
    <code class="mx-1">httpOnly cookie + CSRF header</code>
    <span class="mx-1">·</span>
    {#if publicExposureRisk === 'high'}
      <span class="font-medium"
        >Public exposure detected — configure OIDC before wider rollout.</span
      >
    {:else}
      <span>OIDC tokens stay server-side.</span>
    {/if}
  </div>
</div>
