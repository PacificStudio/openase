<script lang="ts">
  import type { AdminAuthModeTransitionResponse, SecurityAuthSettings } from '$lib/api/contracts'
  import { formatTimestamp, SecuritySettingsHumanAuthGuideLinks } from '$lib/features/settings'
  import { Badge } from '$ui/badge'
  import { CheckCircle2, LockKeyhole } from '@lucide/svelte'

  let {
    auth,
    transition = null,
  }: {
    auth: SecurityAuthSettings
    transition?: AdminAuthModeTransitionResponse['transition'] | null
  } = $props()
</script>

<div class="border-border bg-card space-y-4 rounded-2xl border p-5">
  <div class="flex items-center justify-between gap-3">
    <div class="text-sm font-semibold">Last validation diagnostics</div>
    <Badge
      variant={auth.last_validation.status === 'ok'
        ? 'secondary'
        : auth.last_validation.status === 'failed'
          ? 'destructive'
          : 'outline'}
    >
      {auth.last_validation.status}
    </Badge>
  </div>

  <div class="rounded-xl border px-4 py-3">
    <div class="flex items-start gap-2">
      <LockKeyhole class="text-muted-foreground mt-0.5 size-4 shrink-0" />
      <div class="space-y-2 text-sm">
        <div class="font-medium">{auth.last_validation.message}</div>
        {#if auth.last_validation.checked_at}
          <div class="text-muted-foreground text-xs">
            Last checked {formatTimestamp(auth.last_validation.checked_at)}
          </div>
        {/if}
      </div>
    </div>
  </div>

  <div class="grid gap-3">
    <div>
      <div class="text-muted-foreground text-xs">Issuer</div>
      <div class="mt-1 text-sm font-medium break-all">
        {auth.last_validation.issuer_url || 'Not recorded'}
      </div>
    </div>
    <div>
      <div class="text-muted-foreground text-xs">Authorization endpoint</div>
      <div class="mt-1 text-sm break-all">
        {auth.last_validation.authorization_endpoint || 'Not recorded'}
      </div>
    </div>
    <div>
      <div class="text-muted-foreground text-xs">Token endpoint</div>
      <div class="mt-1 text-sm break-all">
        {auth.last_validation.token_endpoint || 'Not recorded'}
      </div>
    </div>
    <div>
      <div class="text-muted-foreground text-xs">Redirect URL</div>
      <div class="mt-1 text-sm break-all">
        {auth.last_validation.redirect_url ||
          auth.oidc_draft.fixed_redirect_url ||
          'Not configured'}
      </div>
    </div>
    {#if auth.last_validation.warnings.length > 0}
      <div
        class="space-y-2 rounded-xl border border-amber-200 bg-amber-50 px-3 py-3 text-xs text-amber-950"
      >
        {#each auth.last_validation.warnings as warning (warning)}
          <div>{warning}</div>
        {/each}
      </div>
    {/if}
  </div>

  {#if transition}
    <div
      class="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-950"
    >
      <div class="flex items-start gap-2">
        <CheckCircle2 class="mt-0.5 size-4 shrink-0" />
        <div class="space-y-2">
          <div class="font-medium">{transition.message}</div>
          {#if transition.restart_required}
            <div class="text-xs font-medium tracking-wide text-emerald-800 uppercase">
              Restart required
            </div>
          {/if}
          <ol class="list-inside list-decimal space-y-1 text-xs leading-relaxed">
            {#each transition.next_steps as step (step)}
              <li>{step}</li>
            {/each}
          </ol>
        </div>
      </div>
    </div>
  {/if}

  <SecuritySettingsHumanAuthGuideLinks docs={auth.docs} />
</div>
