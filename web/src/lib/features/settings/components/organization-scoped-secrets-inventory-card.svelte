<script lang="ts">
  import type { ScopedSecretRecord } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Skeleton } from '$ui/skeleton'
  import { formatSecretTimestamp, normalizeUsageScopes, usageIndicator } from '../scoped-secrets'

  let {
    loading,
    secrets,
    onRotate,
    onDisable,
    onDelete,
  }: {
    loading: boolean
    secrets: ScopedSecretRecord[]
    onRotate: (secret: ScopedSecretRecord, value: string) => Promise<void>
    onDisable: (secret: ScopedSecretRecord) => Promise<void>
    onDelete: (secret: ScopedSecretRecord) => Promise<void>
  } = $props()

  // Rotate dialog
  let rotateOpen = $state(false)
  let rotateTarget = $state<ScopedSecretRecord | null>(null)
  let rotateDraft = $state('')
  let rotateSubmitting = $state(false)

  // Disable / Delete confirmation dialog
  let confirmOpen = $state(false)
  let confirmTarget = $state<{ kind: 'disable' | 'delete'; secret: ScopedSecretRecord } | null>(
    null,
  )
  let confirmSubmitting = $state(false)

  function openRotate(secret: ScopedSecretRecord) {
    rotateTarget = secret
    rotateDraft = ''
    rotateOpen = true
  }

  async function submitRotate() {
    if (!rotateTarget) return
    const value = rotateDraft.trim()
    if (!value) return
    rotateSubmitting = true
    try {
      await onRotate(rotateTarget, value)
      rotateOpen = false
    } catch {
      // error toast shown by parent; dialog stays open for retry
    } finally {
      rotateSubmitting = false
    }
  }

  function openConfirm(kind: 'disable' | 'delete', secret: ScopedSecretRecord) {
    confirmTarget = { kind, secret }
    confirmOpen = true
  }

  async function submitConfirm() {
    if (!confirmTarget) return
    confirmSubmitting = true
    try {
      if (confirmTarget.kind === 'disable') {
        await onDisable(confirmTarget.secret)
      } else {
        await onDelete(confirmTarget.secret)
      }
      confirmOpen = false
      confirmTarget = null
    } catch {
      // error toast shown by parent; dialog stays open for retry
    } finally {
      confirmSubmitting = false
    }
  }
</script>

{#if loading}
  <div class="space-y-2">
    {#each Array(3) as _, i (i)}
      <div class="border-border rounded-md border p-4">
        <div class="flex items-start justify-between gap-3">
          <div class="flex-1 space-y-2">
            <Skeleton class="h-4 w-32" />
            <Skeleton class="h-3 w-48" />
            <Skeleton class="h-3 w-64" />
          </div>
          <div class="flex gap-2">
            <Skeleton class="h-8 w-16 rounded-md" />
            <Skeleton class="h-8 w-16 rounded-md" />
            <Skeleton class="h-8 w-16 rounded-md" />
          </div>
        </div>
      </div>
    {/each}
  </div>
{:else if secrets.length === 0}
  <div class="border-border rounded-md border border-dashed p-8 text-center">
    <p class="text-foreground text-sm font-medium">No organization secrets yet</p>
    <p class="text-muted-foreground mt-1 text-sm">
      Add a secret above — it will be available in every project in this org.
    </p>
  </div>
{:else}
  <div class="border-border divide-border divide-y rounded-md border">
    {#each secrets as secret (secret.id)}
      <div class="bg-card px-4 py-3 first:rounded-t-md last:rounded-b-md">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0 space-y-1.5">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-foreground text-sm font-semibold">{secret.name}</span>
              <Badge variant="secondary">Organization</Badge>
              {#if secret.disabled}
                <Badge variant="destructive">Disabled</Badge>
              {/if}
            </div>
            <p class="text-muted-foreground text-sm">
              {secret.description || 'No description.'}
            </p>
            <div class="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
              <code class="bg-muted rounded px-1 py-0.5 font-mono">
                {secret.encryption.value_preview}
              </code>
              <span>Rotated {formatSecretTimestamp(secret.encryption.rotated_at)}</span>
              <span>Updated {formatSecretTimestamp(secret.updated_at)}</span>
            </div>
            <div class="flex flex-wrap gap-1">
              <Badge variant="outline">{usageIndicator(secret)}</Badge>
              {#each normalizeUsageScopes(secret) as scope (scope)}
                <Badge variant="outline">{scope}</Badge>
              {/each}
            </div>
          </div>

          <div class="flex shrink-0 flex-wrap gap-2">
            <Button variant="outline" size="sm" onclick={() => openRotate(secret)}>Rotate</Button>
            <Button
              variant="outline"
              size="sm"
              onclick={() => openConfirm('disable', secret)}
              disabled={secret.disabled}
            >
              Disable
            </Button>
            <Button variant="destructive" size="sm" onclick={() => openConfirm('delete', secret)}>
              Delete
            </Button>
          </div>
        </div>
      </div>
    {/each}
  </div>
{/if}

<!-- Rotate dialog -->
<Dialog.Root bind:open={rotateOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Rotate {rotateTarget?.name}</Dialog.Title>
      <Dialog.Description>
        Paste the new secret value. The previous value is immediately replaced and cannot be
        recovered.
      </Dialog.Description>
    </Dialog.Header>
    <div class="space-y-1.5">
      <Label for="org-rotate-value">New value</Label>
      <Input
        id="org-rotate-value"
        type="password"
        bind:value={rotateDraft}
        placeholder="Paste the new secret value"
        onkeydown={(e) => {
          if (e.key === 'Enter') submitRotate()
        }}
      />
    </div>
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={rotateSubmitting}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={submitRotate} disabled={rotateSubmitting || !rotateDraft.trim()}>
        {rotateSubmitting ? 'Rotating…' : 'Confirm rotate'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<!-- Disable / Delete confirmation dialog -->
<Dialog.Root bind:open={confirmOpen}>
  <Dialog.Content class="sm:max-w-sm">
    <Dialog.Header>
      <Dialog.Title>
        {confirmTarget?.kind === 'delete' ? 'Delete' : 'Disable'}
        {confirmTarget?.secret.name}?
      </Dialog.Title>
      <Dialog.Description>
        {#if confirmTarget?.kind === 'delete'}
          This permanently removes the secret and its binding from every project that inherits it.
          This cannot be undone.
        {:else}
          Projects will fall back to a lower-precedence value when available. You can re-enable this
          secret later.
        {/if}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={confirmSubmitting}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button
        variant={confirmTarget?.kind === 'delete' ? 'destructive' : 'outline'}
        onclick={submitConfirm}
        disabled={confirmSubmitting}
      >
        {#if confirmSubmitting}
          {confirmTarget?.kind === 'delete' ? 'Deleting…' : 'Disabling…'}
        {:else}
          {confirmTarget?.kind === 'delete' ? 'Delete' : 'Disable'}
        {/if}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
