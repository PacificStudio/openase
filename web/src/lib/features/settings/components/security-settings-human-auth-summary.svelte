<script lang="ts">
  import type { HumanAuthUser } from '$lib/stores/auth.svelte'

  let {
    authMode,
    issuerURL = '',
    user = null,
  }: {
    authMode: string
    issuerURL?: string
    user?: HumanAuthUser | null
  } = $props()
</script>

<div class="bg-muted/30 grid gap-3 rounded-lg px-4 py-3 text-xs md:grid-cols-2 xl:grid-cols-4">
  <div>
    <div class="text-muted-foreground">Auth mode</div>
    <div class="text-foreground mt-1 font-medium uppercase">{authMode}</div>
  </div>
  <div>
    <div class="text-muted-foreground">Issuer</div>
    <div class="text-foreground mt-1 font-mono break-all">{issuerURL || 'Not configured'}</div>
  </div>
  <div>
    <div class="text-muted-foreground">Current user</div>
    <div class="text-foreground mt-1 font-medium">{user?.displayName || 'Anonymous'}</div>
    {#if user?.primaryEmail}
      <div class="text-muted-foreground">{user.primaryEmail}</div>
    {/if}
  </div>
  <div>
    <div class="text-muted-foreground">Session boundary</div>
    <div class="text-foreground mt-1">httpOnly cookie + CSRF header</div>
    <div class="text-muted-foreground">OIDC tokens stay server-side.</div>
  </div>
</div>
