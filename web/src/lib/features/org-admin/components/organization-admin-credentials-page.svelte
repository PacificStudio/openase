<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { OrgGitHubCredentialResponse } from '$lib/api/contracts'
  import {
    deleteOrgGitHubCredential,
    getOrgGitHubCredential,
    importOrgGitHubCredentialFromGHCLI,
    retestOrgGitHubCredential,
    saveOrgGitHubCredential,
  } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import {
    ChevronDown,
    ChevronUp,
    KeyRound,
    LoaderCircle,
    ShieldCheck,
    Upload,
  } from '@lucide/svelte'
  import OrganizationAdminCredentialsActions from './organization-admin-credentials-actions.svelte'
  import OrganizationAdminCredentialsIntro from './organization-admin-credentials-intro.svelte'
  import OrganizationAdminCredentialsLoading from './organization-admin-credentials-loading.svelte'

  let { organizationId }: { organizationId: string } = $props()

  type Slot = OrgGitHubCredentialResponse['credential']
  type Probe = Slot['probe'] & { login?: string }

  let credential = $state<Slot | null>(null)
  let loading = $state(false)
  let error = $state('')
  let actionKey = $state('')
  let manualToken = $state('')
  let tokenExpanded = $state(false)

  $effect(() => {
    if (!organizationId) {
      credential = null
      error = ''
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''
      try {
        const payload = await getOrgGitHubCredential(organizationId)
        if (cancelled) return
        credential = payload.credential
      } catch (caughtError) {
        if (cancelled) return
        credential = null
        error =
          caughtError instanceof ApiError
            ? caughtError.detail
            : 'Failed to load org GitHub credential.'
      } finally {
        if (!cancelled) loading = false
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  })

  function statusDot(): string {
    if (!credential?.configured) return 'bg-slate-400'
    if (credential.probe.valid) return 'bg-emerald-500'
    if (credential.probe.state === 'error' || credential.probe.state === 'revoked')
      return 'bg-rose-500'
    return 'bg-amber-500'
  }

  function statusLabel(): string {
    if (!credential?.configured) return 'Not configured'
    return credential.probe.state.replaceAll('_', ' ')
  }

  function displayLogin(): string {
    const login = (credential?.probe as Probe | undefined)?.login?.trim()
    if (!login) return ''
    return login.startsWith('@') ? login : `@${login}`
  }

  function formatCheckedAt(value: string | null | undefined): string {
    if (!value) return 'Never'
    const parsed = new Date(value)
    if (Number.isNaN(parsed.getTime())) return value
    return parsed.toLocaleString()
  }

  const anyBusy = $derived(actionKey !== '')

  function isBusy(action: 'save' | 'import' | 'retest' | 'delete') {
    return actionKey === action
  }

  async function mutate(action: 'save' | 'import' | 'retest' | 'delete') {
    if (!organizationId) return

    actionKey = action
    error = ''

    try {
      let payload: OrgGitHubCredentialResponse
      if (action === 'save') {
        const token = manualToken.trim()
        if (!token) {
          toastStore.error('GitHub token is required.')
          return
        }
        payload = await saveOrgGitHubCredential(organizationId, { token })
        manualToken = ''
        tokenExpanded = false
        toastStore.success('Saved org GitHub credential.')
      } else if (action === 'import') {
        payload = await importOrgGitHubCredentialFromGHCLI(organizationId)
        tokenExpanded = false
        toastStore.success('Imported org credential from gh.')
      } else if (action === 'retest') {
        payload = await retestOrgGitHubCredential(organizationId)
        toastStore.success('Retested org GitHub credential.')
      } else {
        payload = await deleteOrgGitHubCredential(organizationId)
        manualToken = ''
        toastStore.success('Deleted org GitHub credential.')
      }
      credential = payload.credential
    } catch (caughtError) {
      const message =
        caughtError instanceof ApiError ? caughtError.detail : 'GitHub credential update failed.'
      error = message
      toastStore.error(message)
    } finally {
      actionKey = ''
    }
  }
</script>

<div class="space-y-6">
  <OrganizationAdminCredentialsIntro />

  {#if loading}
    <OrganizationAdminCredentialsLoading />
  {:else if error}
    <div class="text-destructive text-sm">{error}</div>
  {:else if credential !== null}
    <div class="border-border rounded-lg border">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3">
        <div class="flex items-center gap-2">
          <ShieldCheck class="text-muted-foreground size-4" />
          <span class="text-sm font-medium">GitHub outbound credential</span>
          <span class={`inline-block size-2 rounded-full ${statusDot()}`}></span>
          <span class="text-muted-foreground text-xs capitalize">{statusLabel()}</span>
        </div>
        <OrganizationAdminCredentialsActions
          configured={credential.configured}
          {anyBusy}
          {actionKey}
          onRetest={() => mutate('retest')}
          onDelete={() => mutate('delete')}
        />
      </div>

      <!-- Details -->
      {#if credential.configured}
        <div class="border-border border-t px-4 py-3">
          <div class="grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
            {#if displayLogin()}
              <div>
                <span class="text-muted-foreground">User</span>
                <div>{displayLogin()}</div>
              </div>
            {/if}
            <div>
              <span class="text-muted-foreground">Token</span>
              <div class="font-mono">{credential.token_preview}</div>
            </div>
            <div>
              <span class="text-muted-foreground">Source</span>
              <div>{credential.source ? credential.source.replaceAll('_', ' ') : '—'}</div>
            </div>
            <div>
              <span class="text-muted-foreground">Repo access</span>
              <div class="capitalize">{credential.probe.repo_access.replaceAll('_', ' ')}</div>
            </div>
            <div>
              <span class="text-muted-foreground">Checked</span>
              <div>{formatCheckedAt(credential.probe.checked_at)}</div>
            </div>
            {#if credential.probe.permissions.length}
              <div class="col-span-2">
                <span class="text-muted-foreground">Permissions</span>
                <div>{credential.probe.permissions.join(', ')}</div>
              </div>
            {/if}
          </div>
          {#if credential.probe.last_error}
            <p class="text-destructive mt-2 text-xs">{credential.probe.last_error}</p>
          {/if}
        </div>
      {/if}

      <!-- Token input -->
      <div class="border-border border-t px-4 py-3">
        {#if credential.configured && !tokenExpanded}
          <div class="flex items-center gap-2">
            <button
              class="text-muted-foreground hover:text-foreground flex items-center gap-1 text-xs transition-colors"
              onclick={() => (tokenExpanded = true)}
            >
              <ChevronDown class="size-3" />
              Rotate token
            </button>
            <span class="text-muted-foreground text-xs">·</span>
            <Button
              variant="ghost"
              size="sm"
              class="text-muted-foreground h-auto px-1 py-0 text-xs"
              onclick={() => mutate('import')}
              disabled={anyBusy}
            >
              {#if isBusy('import')}
                <LoaderCircle class="mr-1 size-3 animate-spin" />
              {:else}
                <Upload class="mr-1 size-3" />
              {/if}
              Import from gh
            </Button>
          </div>
        {:else}
          {#if credential.configured}
            <button
              class="text-muted-foreground hover:text-foreground mb-2 flex items-center gap-1 text-xs transition-colors"
              onclick={() => (tokenExpanded = false)}
            >
              <ChevronUp class="size-3" />
              Cancel
            </button>
          {/if}
          <div class="flex gap-2">
            <Input
              value={manualToken}
              placeholder="ghu_xxx or github_pat_xxx"
              disabled={anyBusy}
              class="h-8 text-xs"
              oninput={(event) => (manualToken = event.currentTarget.value)}
            />
            <Button
              size="sm"
              class="h-8 shrink-0"
              onclick={() => mutate('save')}
              disabled={anyBusy}
            >
              {#if isBusy('save')}
                <LoaderCircle class="mr-1.5 size-3 animate-spin" />
              {:else}
                <KeyRound class="mr-1.5 size-3" />
              {/if}
              Save
            </Button>
            <Button
              variant="outline"
              size="sm"
              class="h-8 shrink-0"
              onclick={() => mutate('import')}
              disabled={anyBusy}
            >
              {#if isBusy('import')}
                <LoaderCircle class="mr-1.5 size-3 animate-spin" />
              {:else}
                <Upload class="mr-1.5 size-3" />
              {/if}
              Import from gh
            </Button>
          </div>
        {/if}
      </div>
    </div>

    <p class="text-muted-foreground text-xs leading-5">
      This credential is shared across all projects in the org. Projects with their own override in
      Security settings will use that instead.
    </p>
  {/if}
</div>
