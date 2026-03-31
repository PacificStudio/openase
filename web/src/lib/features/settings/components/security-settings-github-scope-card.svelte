<script lang="ts">
  import type { SecuritySettingsResponse } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { KeyRound, LoaderCircle, RefreshCw, Trash2, Upload } from '@lucide/svelte'

  type Security = SecuritySettingsResponse['security']
  type GitHubScope = 'organization' | 'project'
  type GitHubSlot = Security['github']['organization']

  let {
    scope,
    title,
    description,
    slot,
    tokenValue,
    actionKey,
    organizationConfigured,
    onAction,
    onTokenChange,
  }: {
    scope: GitHubScope
    title: string
    description: string
    slot: GitHubSlot
    tokenValue: string
    actionKey: string
    organizationConfigured: boolean
    onAction: (scope: GitHubScope, action: 'save' | 'import' | 'retest' | 'delete') => void
    onTokenChange: (scope: GitHubScope, value: string) => void
  } = $props()

  function probeTone() {
    if (!slot.configured) return 'secondary'
    if (slot.probe.valid) return 'outline'
    if (slot.probe.state === 'error' || slot.probe.state === 'revoked') return 'destructive'
    return 'secondary'
  }

  function probeLabel() {
    if (!slot.configured) return 'Missing'
    return slot.probe.state.replaceAll('_', ' ')
  }

  function slotHint() {
    if (scope === 'project' && !slot.configured && organizationConfigured) {
      return 'No project override is configured. This project currently falls back to the organization default.'
    }
    if (!slot.configured) {
      return 'No platform-managed credential is stored at this scope yet.'
    }
    return 'This scope is stored in platform secret storage and immediately probed after save or import.'
  }

  function formatCheckedAt(value: string | null | undefined) {
    if (!value) return 'Not checked yet'
    const parsed = new Date(value)
    if (Number.isNaN(parsed.getTime())) return value
    return parsed.toLocaleString()
  }

  function isBusy(action: 'save' | 'import' | 'retest' | 'delete') {
    return actionKey === `${scope}:${action}`
  }
</script>

<div class="border-border bg-card rounded-2xl border p-4">
  <div class="space-y-1">
    <div class="flex flex-wrap items-center gap-2">
      <div class="font-medium">{title}</div>
      <Badge variant={probeTone()}>{probeLabel()}</Badge>
      {#if slot.source}
        <Badge variant="outline">{slot.source}</Badge>
      {/if}
    </div>
    <p class="text-muted-foreground text-sm">{description}</p>
  </div>

  <div class="text-muted-foreground mt-4 space-y-2 text-sm">
    <p>{slotHint()}</p>
    <p>Token preview: {slot.token_preview || 'Not configured'}</p>
    <p>Repo access: {slot.probe.repo_access.replaceAll('_', ' ')}</p>
    <p>Checked at: {formatCheckedAt(slot.probe.checked_at)}</p>
    {#if slot.probe.permissions.length}
      <p>Permissions: {slot.probe.permissions.join(', ')}</p>
    {/if}
    {#if slot.probe.last_error}
      <p class="text-destructive">Last error: {slot.probe.last_error}</p>
    {/if}
  </div>

  <div class="mt-4 space-y-2">
    <Label for={`github-token-${scope}`}>{slot.configured ? 'Rotate token' : 'Paste token'}</Label>
    <Textarea
      id={`github-token-${scope}`}
      value={tokenValue}
      rows={3}
      placeholder="ghu_xxx or github_pat_xxx"
      disabled={actionKey !== ''}
      oninput={(event) => onTokenChange(scope, event.currentTarget.value)}
    />
  </div>

  <div class="mt-4 flex flex-wrap gap-2">
    <Button onclick={() => onAction(scope, 'save')} disabled={actionKey !== ''}>
      {#if isBusy('save')}
        <LoaderCircle class="mr-2 size-4 animate-spin" />
      {:else}
        <KeyRound class="mr-2 size-4" />
      {/if}
      {slot.configured ? 'Save rotation' : 'Save token'}
    </Button>

    <Button variant="outline" onclick={() => onAction(scope, 'import')} disabled={actionKey !== ''}>
      {#if isBusy('import')}
        <LoaderCircle class="mr-2 size-4 animate-spin" />
      {:else}
        <Upload class="mr-2 size-4" />
      {/if}
      Import from gh
    </Button>

    <Button
      variant="outline"
      onclick={() => onAction(scope, 'retest')}
      disabled={!slot.configured || actionKey !== ''}
    >
      {#if isBusy('retest')}
        <LoaderCircle class="mr-2 size-4 animate-spin" />
      {:else}
        <RefreshCw class="mr-2 size-4" />
      {/if}
      Retest
    </Button>

    <Button
      variant="destructive"
      onclick={() => onAction(scope, 'delete')}
      disabled={!slot.configured || actionKey !== ''}
    >
      {#if isBusy('delete')}
        <LoaderCircle class="mr-2 size-4 animate-spin" />
      {:else}
        <Trash2 class="mr-2 size-4" />
      {/if}
      Delete
    </Button>
  </div>
</div>
