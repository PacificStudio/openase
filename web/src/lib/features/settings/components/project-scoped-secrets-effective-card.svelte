<script lang="ts">
  import type { ScopedSecretRecord } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import * as Card from '$ui/card'
  import {
    formatSecretTimestamp,
    isProjectOverride,
    normalizeUsageScopes,
    usageIndicator,
  } from '../scoped-secrets'

  let {
    loading,
    effectiveSecrets,
    organizationSecrets,
  }: {
    loading: boolean
    effectiveSecrets: ScopedSecretRecord[]
    organizationSecrets: ScopedSecretRecord[]
  } = $props()
</script>

<Card.Root class="rounded-2xl">
  <Card.Header>
    <Card.Title>Effective inventory</Card.Title>
    <Card.Description>
      The secrets that currently resolve for this project after org inheritance and project
      override precedence.
    </Card.Description>
  </Card.Header>
  <Card.Content>
    {#if loading}
      <div class="text-sm text-slate-500">Loading effective secrets…</div>
    {:else if effectiveSecrets.length === 0}
      <div class="text-sm text-slate-500">No effective secrets are available yet.</div>
    {:else}
      <div class="space-y-3">
        {#each effectiveSecrets as secret (secret.id)}
          <div class="rounded-2xl border border-slate-200 bg-white p-4">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <div class="text-sm font-semibold text-slate-950">{secret.name}</div>
                  <Badge variant={secret.scope === 'project' ? 'default' : 'secondary'}>
                    {secret.scope === 'project' ? 'Project override' : 'Inherited from org'}
                  </Badge>
                  {#if secret.scope === 'project' && isProjectOverride(secret, organizationSecrets)}
                    <Badge variant="outline">Overrides org secret</Badge>
                  {/if}
                </div>
                <div class="text-sm text-slate-600">{secret.description || 'No description yet.'}</div>
                <div class="text-xs text-slate-500">
                  Preview {secret.encryption.value_preview} · rotated {formatSecretTimestamp(
                    secret.encryption.rotated_at,
                  )} · updated {formatSecretTimestamp(secret.updated_at)}
                </div>
              </div>

              <div class="space-y-2 text-right">
                <div class="text-xs font-medium tracking-[0.16em] text-slate-500 uppercase">
                  Usage
                </div>
                <div class="text-sm font-semibold text-slate-900">{usageIndicator(secret)}</div>
                <div class="flex flex-wrap justify-end gap-1">
                  {#each normalizeUsageScopes(secret) as scope (scope)}
                    <Badge variant="outline">{scope}</Badge>
                  {/each}
                </div>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </Card.Content>
</Card.Root>
