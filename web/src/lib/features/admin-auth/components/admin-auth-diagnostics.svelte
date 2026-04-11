<script lang="ts">
  import type { AdminAuthModeTransitionResponse, SecurityAuthSettings } from '$lib/api/contracts'
  import { formatTimestamp, SecuritySettingsHumanAuthGuideLinks } from '$lib/features/settings'
  import { Badge } from '$ui/badge'
  import * as Collapsible from '$ui/collapsible'
  import { CheckCircle2, ChevronDown, LockKeyhole } from '@lucide/svelte'

  let {
    auth,
    transition = null,
  }: {
    auth: SecurityAuthSettings
    transition?: AdminAuthModeTransitionResponse['transition'] | null
  } = $props()

  // Auto-expand when there's a transition or a failure
  const shouldAutoOpen = $derived(!!transition || auth.last_validation.status === 'failed')
  let open = $state(false)

  $effect(() => {
    if (shouldAutoOpen) {
      open = true
    }
  })
</script>

<Collapsible.Root bind:open>
  <div class="border-border bg-card rounded-2xl border">
    <Collapsible.Trigger class="flex w-full items-center justify-between px-5 py-4 text-left">
      <div class="flex items-center gap-3">
        <span class="text-sm font-semibold">Validation</span>
        <Badge
          variant={auth.last_validation.status === 'ok'
            ? 'secondary'
            : auth.last_validation.status === 'failed'
              ? 'destructive'
              : 'outline'}
        >
          {auth.last_validation.status}
        </Badge>
        {#if auth.last_validation.checked_at}
          <span class="text-muted-foreground hidden text-xs sm:inline">
            {formatTimestamp(auth.last_validation.checked_at)}
          </span>
        {/if}
      </div>
      <ChevronDown
        class="text-muted-foreground size-4 shrink-0 transition-transform duration-200 {open
          ? 'rotate-180'
          : ''}"
      />
    </Collapsible.Trigger>

    <Collapsible.Content>
      <div class="space-y-4 border-t px-5 pt-4 pb-5">
        <!-- Validation message -->
        <div class="rounded-xl border px-4 py-3">
          <div class="flex items-start gap-2">
            <LockKeyhole class="text-muted-foreground mt-0.5 size-4 shrink-0" />
            <div class="text-sm font-medium">{auth.last_validation.message}</div>
          </div>
        </div>

        <!-- Endpoint details -->
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <div class="text-muted-foreground text-xs">Issuer</div>
            <div class="mt-1 text-sm break-all">
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

        <!-- Guide links -->
        <SecuritySettingsHumanAuthGuideLinks docs={auth.docs} />
      </div>
    </Collapsible.Content>
  </div>
</Collapsible.Root>
