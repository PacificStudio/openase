<script lang="ts">
  import type { SecurityAuthSettings } from '$lib/api/contracts'
  import * as Collapsible from '$ui/collapsible'
  import { ChevronDown } from '@lucide/svelte'
  import { adminAuthT } from './i18n'

  type Props = {
    auth: SecurityAuthSettings
  }

  let { auth }: Props = $props()

  let open = $state(false)
</script>

<Collapsible.Root bind:open>
  <div class="border-border bg-card rounded-2xl border">
    <Collapsible.Trigger class="flex w-full items-center justify-between px-5 py-4 text-left">
    <span class="text-sm font-semibold">
      {adminAuthT('adminAuth.runtime.title')}
    </span>
      <ChevronDown
        class="text-muted-foreground size-4 shrink-0 transition-transform duration-200 {open
          ? 'rotate-180'
          : ''}"
      />
    </Collapsible.Trigger>
    <Collapsible.Content>
      <div class="space-y-4 border-t px-5 pt-4 pb-5">
        <div class="grid gap-4 sm:grid-cols-3">
          <div>
            <div class="text-muted-foreground text-xs">
              {adminAuthT('adminAuth.labels.sessionTtl')}
            </div>
            <div class="mt-1 text-sm font-medium">{auth.session_policy.session_ttl}</div>
          </div>
          <div>
            <div class="text-muted-foreground text-xs">
              {adminAuthT('adminAuth.labels.idleTtl')}
            </div>
            <div class="mt-1 text-sm font-medium">{auth.session_policy.session_idle_ttl}</div>
          </div>
          <div>
            <div class="text-muted-foreground text-xs">
              {adminAuthT('adminAuth.runtime.configFile')}
            </div>
            <div class="mt-1 font-mono text-xs break-all">
              {auth.config_path ||
                adminAuthT('adminAuth.runtime.notAvailable')}
            </div>
          </div>
        </div>

        {#if auth.next_steps.length > 0}
        <div>
          <div class="text-muted-foreground mb-2 text-xs font-medium">
            {adminAuthT('adminAuth.diagnostics.nextStepsTitle')}
          </div>
            <ol
              class="text-muted-foreground list-inside list-decimal space-y-1.5 text-sm leading-relaxed"
            >
              {#each auth.next_steps as step (step)}
                <li>{step}</li>
              {/each}
            </ol>
          </div>
        {/if}
      </div>
    </Collapsible.Content>
  </div>
</Collapsible.Root>
