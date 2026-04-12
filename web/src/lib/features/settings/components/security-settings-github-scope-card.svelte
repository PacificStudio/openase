<script lang="ts">
  import type { SecuritySettingsResponse } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import {
    ChevronDown,
    ChevronUp,
    KeyRound,
    LoaderCircle,
    RefreshCw,
    Trash2,
    Upload,
  } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  type Security = SecuritySettingsResponse['security']
  type GitHubSlot = Security['github']['project_override']

  let {
    slot,
    tokenValue,
    actionKey,
    organizationConfigured,
    onAction,
    onTokenChange,
  }: {
    slot: GitHubSlot
    tokenValue: string
    actionKey: string
    organizationConfigured: boolean
    onAction: (action: 'save' | 'import' | 'retest' | 'delete') => void
    onTokenChange: (value: string) => void
  } = $props()

  let tokenExpanded = $state(false)

  function statusDot(): string {
    if (!slot.configured) return 'bg-slate-400'
    if (slot.probe.valid) return 'bg-emerald-500'
    if (slot.probe.state === 'error' || slot.probe.state === 'revoked') return 'bg-rose-500'
    return 'bg-amber-500'
  }

  function statusLabel(): string {
    if (!slot.configured) {
      return i18nStore.t('settings.security.github.status.notConfigured')
    }
    return slot.probe.state.replaceAll('_', ' ')
  }

  function formatCheckedAt(value: string | null | undefined): string {
    if (!value) return 'Never'
    const parsed = new Date(value)
    if (Number.isNaN(parsed.getTime())) return value
    return parsed.toLocaleString()
  }

  function displayLogin(): string {
    const login = (slot.probe as typeof slot.probe & { login?: string }).login?.trim()
    if (!login) return ''
    return login.startsWith('@') ? login : `@${login}`
  }

  function isBusy(action: 'save' | 'import' | 'retest' | 'delete') {
    return actionKey === action
  }

  const anyBusy = $derived(actionKey !== '')
</script>

<div class="border-border rounded-lg border">
  <!-- Header -->
  <div class="flex items-center justify-between px-4 py-3">
    <div class="flex items-center gap-2">
      <span class={`inline-block size-2 rounded-full ${statusDot()}`}></span>
      <span class="text-sm font-medium">
        {i18nStore.t('settings.security.github.title.projectOverride')}
      </span>
      <span class="text-muted-foreground text-xs capitalize">{statusLabel()}</span>
    </div>
    {#if slot.configured}
      <div class="flex items-center gap-1">
        <Button
          variant="ghost"
          size="icon"
          class="size-7"
          onclick={() => onAction('retest')}
          disabled={anyBusy}
          title={i18nStore.t('settings.security.github.buttons.retest')}
        >
          {#if isBusy('retest')}
            <LoaderCircle class="size-3.5 animate-spin" />
          {:else}
            <RefreshCw class="size-3.5" />
          {/if}
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="text-destructive hover:text-destructive size-7"
          onclick={() => onAction('delete')}
          disabled={anyBusy}
          title={i18nStore.t('settings.security.github.buttons.delete')}
        >
          {#if isBusy('delete')}
            <LoaderCircle class="size-3.5 animate-spin" />
          {:else}
            <Trash2 class="size-3.5" />
          {/if}
        </Button>
      </div>
    {/if}
  </div>

  <!-- Credential details (when configured) -->
  {#if slot.configured}
    <div class="border-border border-t px-4 py-3">
      <div class="grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
        {#if displayLogin()}
          <div>
            <span class="text-muted-foreground"
              >{i18nStore.t('settings.security.github.labels.user')}</span
            >
            <div>{displayLogin()}</div>
          </div>
        {/if}
        <div>
          <span class="text-muted-foreground">
            {i18nStore.t('settings.security.github.labels.token')}
          </span>
          <div class="font-mono">{slot.token_preview}</div>
        </div>
        <div>
          <span class="text-muted-foreground">
            {i18nStore.t('settings.security.github.labels.source')}
          </span>
          <div>{slot.source ? slot.source.replaceAll('_', ' ') : '—'}</div>
        </div>
        <div>
          <span class="text-muted-foreground">
            {i18nStore.t('settings.security.github.labels.repoAccess')}
          </span>
          <div class="capitalize">{slot.probe.repo_access.replaceAll('_', ' ')}</div>
        </div>
        <div>
          <span class="text-muted-foreground">
            {i18nStore.t('settings.security.github.labels.checked')}
          </span>
          <div>{formatCheckedAt(slot.probe.checked_at)}</div>
        </div>
        {#if slot.probe.permissions.length}
          <div class="col-span-2">
            <span class="text-muted-foreground">
              {i18nStore.t('settings.security.github.labels.permissions')}
            </span>
            <div>{slot.probe.permissions.join(', ')}</div>
          </div>
        {/if}
      </div>
      {#if slot.probe.last_error}
        <p class="text-destructive mt-2 text-xs">{slot.probe.last_error}</p>
      {/if}
    </div>
  {:else if organizationConfigured}
    <div class="border-border border-t px-4 py-3">
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('settings.security.github.messages.fallbackToOrg')}
      </p>
    </div>
  {/if}

  <!-- Token input -->
  <div class="border-border border-t px-4 py-3">
    {#if slot.configured && !tokenExpanded}
      <div class="flex items-center gap-2">
        <button
          class="text-muted-foreground hover:text-foreground flex items-center gap-1 text-xs transition-colors"
          onclick={() => (tokenExpanded = true)}
        >
          <ChevronDown class="size-3" />
          {i18nStore.t('settings.security.github.buttons.rotateToken')}
        </button>
        <span class="text-muted-foreground text-xs">·</span>
        <Button
          variant="ghost"
          size="sm"
          class="text-muted-foreground h-auto px-1 py-0 text-xs"
          onclick={() => onAction('import')}
          disabled={anyBusy}
        >
          {#if isBusy('import')}
            <LoaderCircle class="mr-1 size-3 animate-spin" />
          {:else}
            <Upload class="mr-1 size-3" />
          {/if}
          {i18nStore.t('settings.security.github.buttons.importFromGh')}
        </Button>
      </div>
    {:else}
      {#if slot.configured}
        <button
          class="text-muted-foreground hover:text-foreground mb-2 flex items-center gap-1 text-xs transition-colors"
          onclick={() => (tokenExpanded = false)}
        >
          <ChevronUp class="size-3" />
          {i18nStore.t('settings.security.github.buttons.cancel')}
        </button>
      {/if}
      <div class="flex gap-2">
        <Input
          value={tokenValue}
          placeholder={i18nStore.t('settings.security.github.placeholders.token')}
          disabled={anyBusy}
          class="h-8 text-xs"
          oninput={(event) => onTokenChange(event.currentTarget.value)}
        />
        <Button size="sm" class="h-8 shrink-0" onclick={() => onAction('save')} disabled={anyBusy}>
          {#if isBusy('save')}
            <LoaderCircle class="mr-1.5 size-3 animate-spin" />
          {:else}
            <KeyRound class="mr-1.5 size-3" />
          {/if}
          {i18nStore.t('settings.security.github.buttons.save')}
        </Button>
        <Button
          variant="outline"
          size="sm"
          class="h-8 shrink-0"
          onclick={() => onAction('import')}
          disabled={anyBusy}
        >
          {#if isBusy('import')}
            <LoaderCircle class="mr-1.5 size-3 animate-spin" />
          {:else}
            <Upload class="mr-1.5 size-3" />
          {/if}
          {i18nStore.t('settings.security.github.buttons.importFromGh')}
        </Button>
      </div>
    {/if}
  </div>
</div>
