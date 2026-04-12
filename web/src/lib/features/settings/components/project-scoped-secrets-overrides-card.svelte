<script lang="ts">
  import type { ScopedSecretRecord } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Skeleton } from '$ui/skeleton'
  import { formatSecretTimestamp, isProjectOverride, usageIndicator } from '../scoped-secrets'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    loading,
    projectOverrides,
    organizationSecrets,
    onRotate,
    onDisable,
    onDelete,
  }: {
    loading: boolean
    projectOverrides: ScopedSecretRecord[]
    organizationSecrets: ScopedSecretRecord[]
    onRotate: (secret: ScopedSecretRecord, value: string) => Promise<void>
    onDisable: (secret: ScopedSecretRecord) => Promise<void>
    onDelete: (secret: ScopedSecretRecord) => Promise<void>
  } = $props()

  let rotateOpen = $state(false)
  let rotateTarget = $state<ScopedSecretRecord | null>(null)
  let rotateDraft = $state('')
  let rotateSubmitting = $state(false)

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

<div class="space-y-3">
  <h4 class="text-sm font-medium">
    {i18nStore.t('settings.projectSecrets.overrides.heading')}
  </h4>

  {#if loading}
    <div class="space-y-2">
      {#each Array(2) as _, i (i)}
        <div class="border-border rounded-lg border p-4">
          <div class="flex items-start justify-between gap-3">
            <div class="flex-1 space-y-2">
              <Skeleton class="h-4 w-28" />
              <Skeleton class="h-3 w-44" />
              <Skeleton class="h-3 w-56" />
            </div>
            <div class="flex gap-2">
              <Skeleton class="h-7 w-14 rounded-md" />
              <Skeleton class="h-7 w-14 rounded-md" />
              <Skeleton class="h-7 w-14 rounded-md" />
            </div>
          </div>
        </div>
      {/each}
    </div>
  {:else if projectOverrides.length === 0}
    <div class="border-border rounded-lg border border-dashed p-6 text-center">
      <p class="text-muted-foreground text-sm">
        {i18nStore.t('settings.projectSecrets.overrides.messages.noOverrides')}
      </p>
      <p class="text-muted-foreground mt-0.5 text-xs">
        {i18nStore.t('settings.projectSecrets.overrides.messages.overrideHint')}
      </p>
    </div>
  {:else}
    <div class="space-y-2">
      {#each projectOverrides as secret (secret.id)}
        <div class="border-border bg-card rounded-lg border p-4">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0 space-y-1.5">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-sm font-medium">{secret.name}</span>
              <Badge variant="default">
                {i18nStore.t('settings.projectSecrets.overrides.badge.project')}
              </Badge>
              <Badge variant="outline">
                {isProjectOverride(secret, organizationSecrets)
                  ? i18nStore.t(
                      'settings.projectSecrets.overrides.badge.overridesOrgDefault',
                    )
                  : i18nStore.t('settings.projectSecrets.overrides.badge.projectOnly')}
              </Badge>
              {#if secret.disabled}
                <Badge variant="destructive">
                  {i18nStore.t('settings.projectSecrets.overrides.status.disabled')}
                </Badge>
              {/if}
            </div>
            <p class="text-muted-foreground text-sm">
              {secret.description ||
                i18nStore.t('settings.projectSecrets.overrides.messages.noDescription')}
            </p>
            <div
              class="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs"
            >
              <code class="bg-muted rounded px-1 py-0.5 font-mono">
                {secret.encryption.value_preview}
              </code>
              <span>{usageIndicator(secret)}</span>
              <span>
                {i18nStore.t('settings.projectSecrets.overrides.labels.updated', {
                  updatedAt: formatSecretTimestamp(secret.updated_at),
                })}
              </span>
            </div>
            </div>

            <div class="flex shrink-0 flex-wrap gap-2">
              <Button variant="outline" size="sm" onclick={() => openRotate(secret)}>
                {i18nStore.t('settings.projectSecrets.overrides.buttons.rotate')}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onclick={() => openConfirm('disable', secret)}
                disabled={secret.disabled}
              >
                {i18nStore.t('settings.projectSecrets.overrides.buttons.disable')}
              </Button>
              <Button variant="destructive" size="sm" onclick={() => openConfirm('delete', secret)}>
                {i18nStore.t('settings.projectSecrets.overrides.buttons.delete')}
              </Button>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Rotate dialog -->
<Dialog.Root bind:open={rotateOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>
        {i18nStore.t('settings.projectSecrets.overrides.dialogs.rotateTitle', {
          secret: rotateTarget?.name ?? '',
        })}
      </Dialog.Title>
      <Dialog.Description>
        {i18nStore.t('settings.projectSecrets.overrides.dialogs.rotateDescription')}
      </Dialog.Description>
    </Dialog.Header>
      <div class="mt-4 space-y-2">
        <Label for="project-rotate-value">
          {i18nStore.t('settings.projectSecrets.overrides.labels.newValue')}
        </Label>
        <Input
          id="project-rotate-value"
          type="password"
          bind:value={rotateDraft}
          placeholder={i18nStore.t(
            'settings.projectSecrets.overrides.placeholders.newValue',
          )}
          onkeydown={(e) => {
            if (e.key === 'Enter') submitRotate()
          }}
        />
      </div>
    <Dialog.Footer class="mt-6">
          <Dialog.Close>
            {#snippet child({ props })}
              <Button variant="outline" {...props} disabled={rotateSubmitting}>
                {i18nStore.t('settings.projectSecrets.overrides.buttons.cancel')}
              </Button>
            {/snippet}
          </Dialog.Close>
          <Button onclick={submitRotate} disabled={rotateSubmitting || !rotateDraft.trim()}>
            {rotateSubmitting
              ? i18nStore.t('settings.projectSecrets.overrides.buttons.rotating')
              : i18nStore.t('settings.projectSecrets.overrides.buttons.confirmRotate')}
          </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<!-- Disable / Delete confirmation dialog -->
<Dialog.Root bind:open={confirmOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      {@const confirmActionLabel =
        confirmTarget?.kind === 'delete'
          ? i18nStore.t(
              'settings.projectSecrets.overrides.confirmDialogs.actions.delete',
            )
          : i18nStore.t(
              'settings.projectSecrets.overrides.confirmDialogs.actions.disable',
            )}
      <Dialog.Title>
        {i18nStore.t('settings.projectSecrets.overrides.confirmDialogs.title', {
          action: confirmActionLabel,
          secret: confirmTarget?.secret.name ?? '',
        })}
      </Dialog.Title>
      <Dialog.Description>
        {#if confirmTarget?.kind === 'delete'}
          {i18nStore.t(
            'settings.projectSecrets.overrides.confirmDialogs.deleteDescription',
          )}
        {:else}
          {i18nStore.t(
            'settings.projectSecrets.overrides.confirmDialogs.disableDescription',
          )}
        {/if}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="mt-6">
        <Dialog.Close>
          {#snippet child({ props })}
            <Button
              variant="outline"
              {...props}
              disabled={confirmSubmitting}
            >
              {i18nStore.t(
                'settings.projectSecrets.overrides.buttons.cancel',
              )}
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
            ? i18nStore.t(
                'settings.projectSecrets.overrides.buttons.deleting',
              )
            : i18nStore.t(
                'settings.projectSecrets.overrides.buttons.disabling',
              )}
        {:else}
          {confirmTarget?.kind === 'delete'
            ? i18nStore.t('settings.projectSecrets.overrides.buttons.delete')
            : i18nStore.t('settings.projectSecrets.overrides.buttons.disable')}
        {/if}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
