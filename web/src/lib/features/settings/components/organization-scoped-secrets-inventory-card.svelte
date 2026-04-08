<script lang="ts">
  import type { ScopedSecretRecord } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Separator } from '$ui/separator'
  import {
    formatSecretTimestamp,
    normalizeUsageScopes,
    usageIndicator,
  } from '../scoped-secrets'

  let {
    loading,
    secrets,
    actionKey,
    openRotateId,
    rotateDrafts,
    onToggleRotate,
    onRotateInput,
    onRotate,
    onDisable,
    onDelete,
  }: {
    loading: boolean
    secrets: ScopedSecretRecord[]
    actionKey: string
    openRotateId: string
    rotateDrafts: Record<string, string>
    onToggleRotate: (secretId: string) => void
    onRotateInput: (secretId: string, value: string) => void
    onRotate: (secret: ScopedSecretRecord) => void
    onDisable: (secret: ScopedSecretRecord) => void
    onDelete: (secret: ScopedSecretRecord) => void
  } = $props()
</script>

<Card.Root class="rounded-2xl">
  <Card.Header>
    <Card.Title>Inventory</Card.Title>
    <Card.Description>
      Operators can inspect masked previews, usage signals, and the latest rotation metadata
      without exposing plaintext values.
    </Card.Description>
  </Card.Header>
  <Card.Content>
    {#if loading}
      <div class="text-sm text-slate-500">Loading organization secrets…</div>
    {:else if secrets.length === 0}
      <div class="text-sm text-slate-500">No organization secrets have been created yet.</div>
    {:else}
      <div class="space-y-4">
        {#each secrets as secret (secret.id)}
          <div class="rounded-2xl border border-slate-200 bg-white p-4">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <div class="text-sm font-semibold text-slate-950">{secret.name}</div>
                  <Badge variant="secondary">Organization</Badge>
                  {#if secret.disabled}
                    <Badge variant="destructive">Disabled</Badge>
                  {/if}
                </div>
                <div class="text-sm text-slate-600">{secret.description || 'No description yet.'}</div>
                <div class="text-xs text-slate-500">
                  Preview {secret.encryption.value_preview} · rotated {formatSecretTimestamp(
                    secret.encryption.rotated_at,
                  )} · updated {formatSecretTimestamp(secret.updated_at)}
                </div>
                <div class="flex flex-wrap gap-1">
                  <Badge variant="outline">{usageIndicator(secret)}</Badge>
                  {#each normalizeUsageScopes(secret) as scope (scope)}
                    <Badge variant="outline">{scope}</Badge>
                  {/each}
                </div>
              </div>

              <div class="flex flex-wrap gap-2">
                <Button variant="outline" onclick={() => onToggleRotate(secret.id)}>
                  {openRotateId === secret.id ? 'Close rotate' : 'Rotate'}
                </Button>
                <Button
                  variant="outline"
                  onclick={() => onDisable(secret)}
                  disabled={secret.disabled || actionKey === `disable:${secret.id}`}
                >
                  {actionKey === `disable:${secret.id}` ? 'Disabling…' : 'Disable'}
                </Button>
                <Button
                  variant="destructive"
                  onclick={() => onDelete(secret)}
                  disabled={actionKey === `delete:${secret.id}`}
                >
                  {actionKey === `delete:${secret.id}` ? 'Deleting…' : 'Delete'}
                </Button>
              </div>
            </div>

            {#if openRotateId === secret.id}
              <Separator class="my-4" />
              <div class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
                <div class="space-y-2">
                  <Label for={`org-rotate-${secret.id}`}>New value for {secret.name}</Label>
                  <Input
                    id={`org-rotate-${secret.id}`}
                    type="password"
                    value={rotateDrafts[secret.id] ?? ''}
                    oninput={(event) => onRotateInput(secret.id, event.currentTarget.value)}
                    placeholder="Paste the rotated secret value"
                  />
                </div>
                <Button
                  onclick={() => onRotate(secret)}
                  disabled={actionKey === `rotate:${secret.id}`}
                >
                  {actionKey === `rotate:${secret.id}` ? 'Rotating…' : 'Confirm rotate'}
                </Button>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </Card.Content>
</Card.Root>
