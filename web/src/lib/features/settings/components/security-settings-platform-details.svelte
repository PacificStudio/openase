<script lang="ts">
  import type { SecuritySettingsResponse } from '$lib/api/contracts'
  import { KeyRound, LockKeyhole, Webhook } from '@lucide/svelte'

  type Security = SecuritySettingsResponse['security']

  let { security }: { security: Security } = $props()
</script>

<div class="space-y-4">
  <!-- Agent runtime tokens -->
  <div class="space-y-2">
    <div class="flex items-center gap-2">
      <KeyRound class="text-muted-foreground size-3.5" />
      <h3 class="text-sm font-semibold">Agent runtime tokens</h3>
    </div>
    <div class="bg-muted/30 rounded-lg px-4 py-3">
      <div class="grid gap-x-6 gap-y-2 text-xs sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <span class="text-muted-foreground">Transport</span>
          <div>{security.agent_tokens.transport}</div>
        </div>
        <div>
          <span class="text-muted-foreground">Env variable</span>
          <div class="font-mono">{security.agent_tokens.environment_variable}</div>
        </div>
        <div>
          <span class="text-muted-foreground">Token prefix</span>
          <div class="font-mono">{security.agent_tokens.token_prefix}</div>
        </div>
        <div>
          <span class="text-muted-foreground">Default scopes</span>
          <div>{security.agent_tokens.default_scopes.join(', ') || 'None'}</div>
        </div>
      </div>
      {#if security.agent_tokens.supported_project_scopes.length}
        <div class="mt-2 text-xs">
          <span class="text-muted-foreground">Mintable project scopes</span>
          <div class="mt-0.5 flex flex-wrap gap-1">
            {#each security.agent_tokens.supported_project_scopes as scope (scope)}
              <code class="bg-background rounded px-1.5 py-0.5 text-[10px]">{scope}</code>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  </div>

  <!-- Inbound webhooks -->
  <div class="space-y-2">
    <div class="flex items-center gap-2">
      <Webhook class="text-muted-foreground size-3.5" />
      <h3 class="text-sm font-semibold">Inbound webhooks</h3>
    </div>
    <div class="bg-muted/30 rounded-lg px-4 py-3">
      <div class="grid gap-x-6 gap-y-2 text-xs">
        <div>
          <span class="text-muted-foreground">Connector ingress route</span>
          <div class="font-mono">{security.webhooks.connector_endpoint}</div>
        </div>
      </div>
    </div>
  </div>

  <!-- Secret hygiene -->
  <div class="space-y-2">
    <div class="flex items-center gap-2">
      <LockKeyhole class="text-muted-foreground size-3.5" />
      <h3 class="text-sm font-semibold">Secret hygiene</h3>
    </div>
    <div class="bg-muted/30 space-y-4 rounded-lg px-4 py-3 text-xs">
      <div class="grid gap-x-6 gap-y-2 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <span class="text-muted-foreground">Notification configs</span>
          <div>
            {security.secret_hygiene.notification_channel_configs_redacted
              ? 'Secrets redacted'
              : 'Secrets may be exposed'}
          </div>
        </div>
        <div>
          <span class="text-muted-foreground">Machine env vars</span>
          <div>
            {security.secret_hygiene.machine_env_vars_redacted
              ? 'Secret-like values masked'
              : 'Raw values may leak'}
          </div>
        </div>
        <div>
          <span class="text-muted-foreground">Legacy provider bindings</span>
          <div>
            {security.secret_hygiene.legacy_provider_inline_secret_bindings} inline secrets across{' '}
            {security.secret_hygiene.legacy_providers_requiring_migration} providers
          </div>
        </div>
        <div>
          <span class="text-muted-foreground">Legacy machine env vars</span>
          <div>
            {security.secret_hygiene.legacy_machine_secret_env_vars} secret-like vars across{' '}
            {security.secret_hygiene.legacy_machines_requiring_migration} machines
          </div>
        </div>
      </div>

      <div class="space-y-2">
        <div class="text-muted-foreground">Rollout checklist</div>
        <div class="space-y-2">
          {#each security.secret_hygiene.rollout_checklist as item (item.key)}
            <div class="bg-background/80 rounded-md px-3 py-2">
              <div class="flex items-center justify-between gap-3">
                <div class="font-medium">{item.title}</div>
                <span
                  class={`rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase ${
                    item.status === 'done'
                      ? 'bg-emerald-500/15 text-emerald-700'
                      : 'bg-amber-500/15 text-amber-700'
                  }`}
                >
                  {item.status}
                </span>
              </div>
              <div class="text-muted-foreground mt-1">{item.summary}</div>
            </div>
          {/each}
        </div>
      </div>
    </div>
  </div>
</div>
