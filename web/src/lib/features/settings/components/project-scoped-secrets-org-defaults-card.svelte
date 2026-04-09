<script lang="ts">
  import type { ScopedSecretRecord } from '$lib/api/contracts'
  import { organizationPath } from '$lib/stores/app-context'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Skeleton } from '$ui/skeleton'
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

<div class="space-y-3">
  <h4 class="text-sm font-medium">Inherited organization defaults</h4>

  {#if loading}
    <div class="space-y-2">
      {#each Array(2) as _, i (i)}
        <div class="border-border bg-muted/20 rounded-lg border p-4">
          <div class="flex items-start justify-between gap-3">
            <div class="flex-1 space-y-2">
              <Skeleton class="h-4 w-28" />
              <Skeleton class="h-3 w-44" />
              <Skeleton class="h-3 w-52" />
            </div>
            <Skeleton class="h-7 w-28 rounded-md" />
          </div>
        </div>
      {/each}
    </div>
  {:else if organizationSecrets.length === 0}
    <div class="border-border rounded-lg border border-dashed p-6 text-center">
      <p class="text-muted-foreground text-sm">No inherited org secrets</p>
      <p class="text-muted-foreground mt-0.5 text-xs">
        Ask your org admin to create organization-level secrets — they will appear here
        automatically.
      </p>
    </div>
  {:else}
    <div class="space-y-2">
      {#each organizationSecrets as secret (secret.id)}
        <div class="border-border bg-muted/20 rounded-lg border p-4">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="space-y-1.5">
              <div class="flex flex-wrap items-center gap-2">
                <span class="text-sm font-medium">{secret.name}</span>
                <Badge variant="secondary">Organization</Badge>
                {#if isOverriddenInProject(secret, projectOverrides)}
                  <Badge variant="outline">Currently overridden</Badge>
                {/if}
                {#if secret.disabled}
                  <Badge variant="destructive">Disabled at org</Badge>
                {/if}
              </div>
              <p class="text-muted-foreground text-sm">
                {secret.description || 'No description yet.'}
              </p>
              <div
                class="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs"
              >
                <code class="bg-muted rounded px-1 py-0.5 font-mono">
                  {secret.encryption.value_preview}
                </code>
                <span>{usageIndicator(secret)}</span>
                <span>Updated {formatSecretTimestamp(secret.updated_at)}</span>
              </div>
            </div>

            <div class="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" onclick={() => onPrimeOverride(secret)}>
                Use as override draft
              </Button>
              {#if organizationId}
                <Button
                  variant="ghost"
                  size="sm"
                  href={`${organizationPath(organizationId)}/admin/settings`}
                >
                  Manage in org admin
                </Button>
              {/if}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
