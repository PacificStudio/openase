<script lang="ts">
  import type { ScopedSecretRecord } from '$lib/api/contracts'
  import { organizationPath } from '$lib/stores/app-context'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { formatSecretTimestamp, isOverriddenInProject, usageIndicator } from '../scoped-secrets'

  let {
    loading,
    organizationSecrets,
    projectOverrides,
    organizationId,
    onPrimeOverride,
  }: {
    loading: boolean
    organizationSecrets: ScopedSecretRecord[]
    projectOverrides: ScopedSecretRecord[]
    organizationId: string
    onPrimeOverride: (secret: ScopedSecretRecord) => void
  } = $props()
</script>

<Card.Root class="rounded-2xl">
  <Card.Header>
    <Card.Title>Inherited organization defaults</Card.Title>
    <Card.Description>
      Central org secrets stay visible here so operators can see which values are inherited and
      which names already have a project override.
    </Card.Description>
  </Card.Header>
  <Card.Content>
    {#if loading}
      <div class="text-sm text-slate-500">Loading inherited organization defaults…</div>
    {:else if organizationSecrets.length === 0}
      <div class="text-sm text-slate-500">
        No organization defaults are available for this project.
      </div>
    {:else}
      <div class="space-y-3">
        {#each organizationSecrets as secret (secret.id)}
          <div class="rounded-2xl border border-slate-200 bg-slate-50/70 p-4">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <div class="text-sm font-semibold text-slate-950">{secret.name}</div>
                  <Badge variant="secondary">Organization</Badge>
                  {#if isOverriddenInProject(secret, projectOverrides)}
                    <Badge variant="outline">Currently overridden</Badge>
                  {/if}
                  {#if secret.disabled}
                    <Badge variant="destructive">Disabled at org</Badge>
                  {/if}
                </div>
                <div class="text-sm text-slate-600">
                  {secret.description || 'No description yet.'}
                </div>
                <div class="text-xs text-slate-500">
                  Preview {secret.encryption.value_preview} · {usageIndicator(secret)} · updated {formatSecretTimestamp(
                    secret.updated_at,
                  )}
                </div>
              </div>

              <div class="flex flex-wrap gap-2">
                <Button variant="outline" onclick={() => onPrimeOverride(secret)}>
                  Use as override draft
                </Button>
                {#if organizationId}
                  <a
                    href={`${organizationPath(organizationId)}/admin/settings`}
                    class="inline-flex items-center rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100 hover:text-slate-950"
                  >
                    Manage in org admin
                  </a>
                {/if}
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </Card.Content>
</Card.Root>
