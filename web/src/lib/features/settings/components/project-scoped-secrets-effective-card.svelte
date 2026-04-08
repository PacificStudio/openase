<script lang="ts">
  import type { ScopedSecretRecord } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import * as Card from '$ui/card'
  import { Skeleton } from '$ui/skeleton'
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
      The secrets that currently resolve for this project after org inheritance and project override
      precedence.
    </Card.Description>
  </Card.Header>
  <Card.Content>
    {#if loading}
      <div class="space-y-3">
        {#each Array(3) as _, i (i)}
          <div class="rounded-2xl border border-slate-200 p-4">
            <div class="flex items-start justify-between gap-3">
              <div class="flex-1 space-y-2">
                <Skeleton class="h-4 w-28" />
                <Skeleton class="h-3 w-44" />
                <Skeleton class="h-3 w-56" />
              </div>
              <div class="space-y-1 text-right">
                <Skeleton class="ml-auto h-3 w-10" />
                <Skeleton class="ml-auto h-5 w-16" />
              </div>
            </div>
          </div>
        {/each}
      </div>
    {:else if effectiveSecrets.length === 0}
      <div class="rounded-2xl border border-dashed border-slate-200 p-8 text-center">
        <p class="text-sm font-medium text-slate-700">No secrets in scope yet</p>
        <p class="mt-1 text-sm text-slate-500">
          Create a project override above or ask your org admin to add an organization secret.
        </p>
      </div>
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
                <div class="text-sm text-slate-600">
                  {secret.description || 'No description yet.'}
                </div>
                <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
                  <code class="rounded bg-slate-100 px-1 py-0.5 font-mono text-slate-700">
                    {secret.encryption.value_preview}
                  </code>
                  <span>Rotated {formatSecretTimestamp(secret.encryption.rotated_at)}</span>
                  <span>Updated {formatSecretTimestamp(secret.updated_at)}</span>
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
