<script lang="ts">
  import type { ScopedSecretRecord } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Skeleton } from '$ui/skeleton'
  import { formatSecretTimestamp, normalizeUsageScopes, usageIndicator } from '../scoped-secrets'
  import { i18nStore } from '$lib/i18n/store.svelte'

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
    <p class="text-foreground text-sm font-medium">
      {i18nStore.t('settings.organizationSecrets.inventory.messages.noSecrets')}
    </p>
    <p class="text-muted-foreground mt-1 text-sm">
      {i18nStore.t('settings.organizationSecrets.inventory.messages.addHint')}
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
              <Badge variant="secondary">
                {i18nStore.t('settings.organizationSecrets.inventory.badge.organization')}
              </Badge>
              {#if secret.disabled}
                <Badge variant="destructive">
                  {i18nStore.t('settings.organizationSecrets.inventory.status.disabled')}
                </Badge>
              {/if}
            </div>
            <p class="text-muted-foreground text-sm">
              {secret.description ||
                i18nStore.t('settings.organizationSecrets.inventory.messages.noDescription')}
            </p>
            <div class="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
              <code class="bg-muted rounded px-1 py-0.5 font-mono">
                {secret.encryption.value_preview}
              </code>
              <span>
                {i18nStore.t('settings.organizationSecrets.inventory.labels.rotated', {
                  rotatedAt: formatSecretTimestamp(secret.encryption.rotated_at),
                })}
              </span>
              <span>
                {i18nStore.t('settings.organizationSecrets.inventory.labels.updated', {
                  updatedAt: formatSecretTimestamp(secret.updated_at),
                })}
              </span>
            </div>
            <div class="flex flex-wrap gap-1">
              <Badge variant="outline">{usageIndicator(secret)}</Badge>
              {#each normalizeUsageScopes(secret) as scope (scope)}
                <Badge variant="outline">{scope}</Badge>
              {/each}
            </div>
          </div>

          <div class="flex shrink-0 flex-wrap gap-2">
            <Button variant="outline" size="sm" onclick={() => openRotate(secret)}>
              {i18nStore.t('settings.organizationSecrets.inventory.buttons.rotate')}
            </Button>
            <Button
              variant="outline"
              size="sm"
              onclick={() => openConfirm('disable', secret)}
              disabled={secret.disabled}
            >
              {i18nStore.t('settings.organizationSecrets.inventory.buttons.disable')}
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onclick={() => openConfirm('delete', secret)}
            >
              {i18nStore.t('settings.organizationSecrets.inventory.buttons.delete')}
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
    <Dialog.Title>
      {i18nStore.t('settings.organizationSecrets.inventory.dialogs.rotateTitle', {
        secret: rotateTarget?.name ?? '',
      })}
    </Dialog.Title>
    <Dialog.Description>
      {i18nStore.t('settings.organizationSecrets.inventory.dialogs.rotateDescription')}
    </Dialog.Description>
  </Dialog.Header>
    <div class="space-y-1.5">
      <Label for="org-rotate-value">
        {i18nStore.t('settings.organizationSecrets.inventory.labels.newValue')}
      </Label>
      <Input
        id="org-rotate-value"
        type="password"
        bind:value={rotateDraft}
        placeholder={i18nStore.t(
          'settings.organizationSecrets.inventory.placeholders.newValue',
        )}
        onkeydown={(e) => {
          if (e.key === 'Enter') submitRotate()
        }}
      />
    </div>
    <Dialog.Footer>
        <Dialog.Close>
          {#snippet child({ props })}
            <Button variant="outline" {...props} disabled={rotateSubmitting}>
              {i18nStore.t('settings.organizationSecrets.inventory.buttons.cancel')}
            </Button>
          {/snippet}
        </Dialog.Close>
        <Button onclick={submitRotate} disabled={rotateSubmitting || !rotateDraft.trim()}>
          {rotateSubmitting
            ? i18nStore.t('settings.organizationSecrets.inventory.buttons.rotating')
            : i18nStore.t('settings.organizationSecrets.inventory.buttons.confirmRotate')}
        </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<!-- Disable / Delete confirmation dialog -->
<Dialog.Root bind:open={confirmOpen}>
  <Dialog.Content class="sm:max-w-sm">
    <Dialog.Header>
      <Dialog.Title>
        {i18nStore.t('settings.organizationSecrets.inventory.confirmDialogs.title', {
          action:
            confirmTarget?.kind === 'delete'
              ? i18nStore.t('settings.organizationSecrets.inventory.buttons.delete')
              : i18nStore.t('settings.organizationSecrets.inventory.buttons.disable'),
          secret: confirmTarget?.secret.name ?? '',
        })}
      </Dialog.Title>
      <Dialog.Description>
        {#if confirmTarget?.kind === 'delete'}
          {i18nStore.t('settings.organizationSecrets.inventory.confirmDialogs.deleteDescription')}
        {:else}
          {i18nStore.t('settings.organizationSecrets.inventory.confirmDialogs.disableDescription')}
        {/if}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
        <Button
          variant="outline"
          {...props}
          disabled={confirmSubmitting}
        >
          {i18nStore.t('settings.organizationSecrets.inventory.buttons.cancel')}
        </Button>
        {/snippet}
      </Dialog.Close>
      <Button
        variant={confirmTarget?.kind === 'delete' ? 'destructive' : 'outline'}
        onclick={submitConfirm}
        disabled={confirmSubmitting}
      >
        {#if confirmSubmitting}
          {confirmTarget?.kind === 'delete'
            ? i18nStore.t('settings.organizationSecrets.inventory.buttons.deleting')
            : i18nStore.t('settings.organizationSecrets.inventory.buttons.disabling')}
        {:else}
          {confirmTarget?.kind === 'delete'
            ? i18nStore.t('settings.organizationSecrets.inventory.buttons.delete')
            : i18nStore.t('settings.organizationSecrets.inventory.buttons.disable')}
        {/if}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
