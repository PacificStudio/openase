<script lang="ts">
  import type {
    CreateProjectUserAPIKeyResponse,
    DisableProjectUserAPIKeyResponse,
    ProjectUserAPIKey,
    SecuritySettingsResponse,
  } from '$lib/api/contracts'
  import {
    createProjectUserAPIKey,
    deleteProjectUserAPIKey,
    disableProjectUserAPIKey,
    listProjectUserAPIKeys,
    rotateProjectUserAPIKey,
  } from '$lib/api/openase'
  import { ScopeGroupPicker } from '$lib/features/workflows'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { ApiError } from '$lib/api/client'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { formatUserAPIKeyTime, toRFC3339Local } from './security-settings-user-api-keys.model'

  type Security = SecuritySettingsResponse['security']
  type PlainTextResult =
    | CreateProjectUserAPIKeyResponse
    | { plain_text_token: string; api_key: ProjectUserAPIKey }

  let { security }: { security: Security } = $props()

  let items = $state<ProjectUserAPIKey[]>([])
  let loading = $state(false)
  let error = $state('')
  let createOpen = $state(false)
  let revealOpen = $state(false)
  let currentToken = $state('')
  let currentTokenName = $state('')
  let name = $state('')
  let expiresAtLocal = $state('')
  let selectedScopes = $state<string[]>([])
  let mutationKey = $state('')

  const scopeGroups = $derived(security.user_api_keys.allowed_scope_groups ?? [])

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      items = []
      return
    }
    let cancelled = false
    const load = async () => {
      loading = true
      error = ''
      try {
        const payload = await listProjectUserAPIKeys(projectId)
        if (!cancelled) items = payload.api_keys
      } catch (caughtError) {
        if (!cancelled) error = formatError(caughtError)
      } finally {
        if (!cancelled) loading = false
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  })

  function formatError(caughtError: unknown): string {
    return caughtError instanceof ApiError
      ? caughtError.detail
      : i18nStore.t('settings.security.userApiKeys.errors.requestFailed')
  }

  function resetDraft() {
    name = ''
    expiresAtLocal = ''
    selectedScopes = []
  }

  async function handleCreate() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return
    mutationKey = 'create'
    try {
      const payload = await createProjectUserAPIKey(projectId, {
        name: name.trim(),
        scopes: [...selectedScopes],
        expires_at: toRFC3339Local(expiresAtLocal) ?? null,
      })
      items = [payload.api_key, ...items]
      showPlainTextToken(payload)
      toastStore.success(i18nStore.t('settings.security.userApiKeys.notifications.created'))
      createOpen = false
      resetDraft()
    } catch (caughtError) {
      toastStore.error(formatError(caughtError))
    } finally {
      mutationKey = ''
    }
  }

  function showPlainTextToken(payload: PlainTextResult) {
    currentToken = payload.plain_text_token
    currentTokenName = payload.api_key.name
    revealOpen = true
  }

  async function handleRotate(item: ProjectUserAPIKey) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return
    mutationKey = `rotate:${item.id}`
    try {
      const payload = await rotateProjectUserAPIKey(projectId, item.id)
      items = items.map((candidate) => (candidate.id === item.id ? payload.api_key : candidate))
      showPlainTextToken(payload)
      toastStore.success(i18nStore.t('settings.security.userApiKeys.notifications.rotated'))
    } catch (caughtError) {
      toastStore.error(formatError(caughtError))
    } finally {
      mutationKey = ''
    }
  }

  async function handleDisable(item: ProjectUserAPIKey) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return
    mutationKey = `disable:${item.id}`
    try {
      const payload: DisableProjectUserAPIKeyResponse = await disableProjectUserAPIKey(
        projectId,
        item.id,
      )
      items = items.map((candidate) => (candidate.id === item.id ? payload.api_key : candidate))
      toastStore.success(i18nStore.t('settings.security.userApiKeys.notifications.disabled'))
    } catch (caughtError) {
      toastStore.error(formatError(caughtError))
    } finally {
      mutationKey = ''
    }
  }

  async function handleDelete(item: ProjectUserAPIKey) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return
    mutationKey = `delete:${item.id}`
    try {
      await deleteProjectUserAPIKey(projectId, item.id)
      items = items.filter((candidate) => candidate.id !== item.id)
      toastStore.success(i18nStore.t('settings.security.userApiKeys.notifications.deleted'))
    } catch (caughtError) {
      toastStore.error(formatError(caughtError))
    } finally {
      mutationKey = ''
    }
  }

  async function copyToken() {
    try {
      await navigator.clipboard.writeText(currentToken)
      toastStore.success(i18nStore.t('settings.security.userApiKeys.notifications.copied'))
    } catch {
      toastStore.error(i18nStore.t('settings.security.userApiKeys.errors.copyFailed'))
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-start justify-between gap-3">
    <div>
      <h3 class="text-sm font-semibold">{i18nStore.t('settings.security.userApiKeys.heading')}</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        {i18nStore.t('settings.security.userApiKeys.description')}
      </p>
    </div>
    <Dialog.Root bind:open={createOpen}>
      <Dialog.Trigger>
        {#snippet child({ props })}
          <Button size="sm" {...props}
            >{i18nStore.t('settings.security.userApiKeys.buttons.create')}</Button
          >
        {/snippet}
      </Dialog.Trigger>
      <Dialog.Content class="sm:max-w-2xl">
        <Dialog.Header>
          <Dialog.Title>{i18nStore.t('settings.security.userApiKeys.dialog.title')}</Dialog.Title>
          <Dialog.Description>
            {i18nStore.t('settings.security.userApiKeys.dialog.description')}
          </Dialog.Description>
        </Dialog.Header>
        <Dialog.Body class="space-y-4">
          <div class="space-y-2">
            <Label for="api-key-name"
              >{i18nStore.t('settings.security.userApiKeys.labels.name')}</Label
            >
            <Input
              id="api-key-name"
              value={name}
              oninput={(event) => (name = (event.currentTarget as HTMLInputElement).value)}
            />
          </div>
          <div class="space-y-2">
            <Label for="api-key-expires"
              >{i18nStore.t('settings.security.userApiKeys.labels.expiresAt')}</Label
            >
            <Input
              id="api-key-expires"
              type="datetime-local"
              value={expiresAtLocal}
              oninput={(event) =>
                (expiresAtLocal = (event.currentTarget as HTMLInputElement).value)}
            />
          </div>
          <div class="space-y-2">
            <Label>{i18nStore.t('settings.security.userApiKeys.labels.allowedScopes')}</Label>
            <ScopeGroupPicker
              groups={scopeGroups}
              selected={selectedScopes}
              onchange={(scopes) => (selectedScopes = scopes)}
              disabled={mutationKey === 'create'}
            />
            {#if scopeGroups.length === 0}
              <p class="text-muted-foreground text-xs">
                {i18nStore.t('settings.security.userApiKeys.messages.noAllowedScopes')}
              </p>
            {/if}
          </div>
        </Dialog.Body>
        <Dialog.Footer>
          <Dialog.Close>
            {#snippet child({ props })}
              <Button variant="outline" {...props}
                >{i18nStore.t('settings.security.userApiKeys.buttons.cancel')}</Button
              >
            {/snippet}
          </Dialog.Close>
          <Button
            onclick={handleCreate}
            disabled={mutationKey === 'create' || !name.trim() || selectedScopes.length === 0}
            >{i18nStore.t('settings.security.userApiKeys.buttons.create')}</Button
          >
        </Dialog.Footer>
      </Dialog.Content>
    </Dialog.Root>
  </div>

  {#if loading}
    <div class="text-muted-foreground text-sm">
      {i18nStore.t('settings.security.userApiKeys.messages.loading')}
    </div>
  {:else if error}
    <div class="text-destructive text-sm">{error}</div>
  {:else if items.length === 0}
    <div class="bg-muted/30 text-muted-foreground rounded-lg px-4 py-3 text-sm">
      {i18nStore.t('settings.security.userApiKeys.messages.empty')}
    </div>
  {:else}
    <div class="space-y-3">
      {#each items as item (item.id)}
        <div class="rounded-lg border px-4 py-3">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <div class="text-sm font-medium">{item.name}</div>
                <span
                  class="text-muted-foreground rounded-full border px-2 py-0.5 text-[10px] uppercase"
                  >{item.status}</span
                >
              </div>
              <div class="text-muted-foreground font-mono text-xs">{item.token_hint}</div>
              <div class="text-muted-foreground text-xs">
                {i18nStore.t('settings.security.userApiKeys.messages.metadata', {
                  created: formatUserAPIKeyTime(item.created_at),
                  lastUsed: formatUserAPIKeyTime(item.last_used_at),
                  expires: formatUserAPIKeyTime(item.expires_at),
                })}
              </div>
            </div>
            <div class="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                onclick={() => handleRotate(item)}
                disabled={mutationKey !== ''}
                >{i18nStore.t('settings.security.userApiKeys.buttons.rotate')}</Button
              >
              <Button
                size="sm"
                variant="outline"
                onclick={() => handleDisable(item)}
                disabled={mutationKey !== '' || item.status !== 'active'}
                >{i18nStore.t('settings.security.userApiKeys.buttons.disable')}</Button
              >
              <Button
                size="sm"
                variant="destructive"
                onclick={() => handleDelete(item)}
                disabled={mutationKey !== ''}
                >{i18nStore.t('settings.security.userApiKeys.buttons.delete')}</Button
              >
            </div>
          </div>
          <div class="mt-3 flex flex-wrap gap-1">
            {#each item.scopes as scope (scope)}
              <code class="bg-muted rounded px-1.5 py-0.5 text-[10px]">{scope}</code>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<Dialog.Root bind:open={revealOpen}>
  <Dialog.Content class="sm:max-w-xl">
    <Dialog.Header>
      <Dialog.Title>{i18nStore.t('settings.security.userApiKeys.reveal.title')}</Dialog.Title>
      <Dialog.Description>
        {i18nStore.t('settings.security.userApiKeys.reveal.description', {
          name: currentTokenName,
        })}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Body class="space-y-3">
      <div class="bg-muted rounded-lg p-3 font-mono text-xs break-all">{currentToken}</div>
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('settings.security.userApiKeys.reveal.warning')}
      </p>
    </Dialog.Body>
    <Dialog.Footer>
      <Button variant="outline" onclick={copyToken}
        >{i18nStore.t('settings.security.userApiKeys.buttons.copyToken')}</Button
      >
      <Dialog.Close>
        {#snippet child({ props })}
          <Button {...props}>{i18nStore.t('settings.security.userApiKeys.buttons.done')}</Button>
        {/snippet}
      </Dialog.Close>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
