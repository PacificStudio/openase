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
  import { KeyRound, LoaderCircle, RefreshCw, Trash2, Upload } from '@lucide/svelte'
  import OrganizationAdminCredentialsDialogs from './organization-admin-credentials-dialogs.svelte'
  import OrganizationAdminCredentialsLoading from './organization-admin-credentials-loading.svelte'

  let { organizationId }: { organizationId: string } = $props()

  type Slot = OrgGitHubCredentialResponse['credential']
  type Probe = Slot['probe'] & { login?: string }

  let credential = $state<Slot | null>(null)
  let loading = $state(false)
  let error = $state('')
  let actionKey = $state('')

  let saveDialogOpen = $state(false)
  let deleteDialogOpen = $state(false)
  let tokenDraft = $state('')

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
    if (!credential?.configured) return 'bg-muted-foreground/40'
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

  async function mutate(action: 'save' | 'import' | 'retest' | 'delete') {
    if (!organizationId) return

    actionKey = action
    error = ''

    try {
      let payload: OrgGitHubCredentialResponse
      if (action === 'save') {
        const token = tokenDraft.trim()
        if (!token) {
          toastStore.error('GitHub token is required.')
          return
        }
        payload = await saveOrgGitHubCredential(organizationId, { token })
        tokenDraft = ''
        saveDialogOpen = false
        toastStore.success('Saved org GitHub credential.')
      } else if (action === 'import') {
        payload = await importOrgGitHubCredentialFromGHCLI(organizationId)
        toastStore.success('Imported org credential from gh.')
      } else if (action === 'retest') {
        payload = await retestOrgGitHubCredential(organizationId)
        toastStore.success('Retested org GitHub credential.')
      } else {
        payload = await deleteOrgGitHubCredential(organizationId)
        tokenDraft = ''
        deleteDialogOpen = false
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

<div class="space-y-4">
  <div>
    <h3 class="text-foreground text-sm font-semibold">GitHub credential</h3>
    <p class="text-muted-foreground mt-0.5 text-xs">
      Org-level GitHub credential used as the default for all projects. Individual projects can
      override it in their own Security settings.
    </p>
  </div>

  {#if loading}
    <OrganizationAdminCredentialsLoading />
  {:else if error}
    <div
      class="border-destructive/30 bg-destructive/5 text-destructive rounded-md border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {:else if credential !== null}
    <div class="border-border rounded-md border">
      <!-- Status + actions row -->
      <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
        <div class="flex items-center gap-2">
          <span class="inline-block size-2 rounded-full {statusDot()}"></span>
          <span class="text-sm font-medium">GitHub outbound credential</span>
          <span class="text-muted-foreground text-xs capitalize">{statusLabel()}</span>
        </div>

        <div class="flex items-center gap-1.5">
          {#if credential.configured}
            <Button
              variant="ghost"
              size="sm"
              class="h-7 px-2 text-xs"
              onclick={() => mutate('retest')}
              disabled={anyBusy}
            >
              {#if actionKey === 'retest'}
                <LoaderCircle class="mr-1 size-3 animate-spin" />
              {:else}
                <RefreshCw class="mr-1 size-3" />
              {/if}
              Retest
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="h-7 px-2 text-xs"
              onclick={() => {
                tokenDraft = ''
                saveDialogOpen = true
              }}
              disabled={anyBusy}
            >
              <KeyRound class="mr-1 size-3" />
              Rotate token
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="text-destructive hover:text-destructive h-7 px-2 text-xs"
              onclick={() => (deleteDialogOpen = true)}
              disabled={anyBusy}
            >
              <Trash2 class="mr-1 size-3" />
              Delete
            </Button>
          {:else}
            <Button
              variant="outline"
              size="sm"
              class="h-7 px-2 text-xs"
              onclick={() => mutate('import')}
              disabled={anyBusy}
            >
              {#if actionKey === 'import'}
                <LoaderCircle class="mr-1 size-3 animate-spin" />
              {:else}
                <Upload class="mr-1 size-3" />
              {/if}
              Import from gh
            </Button>
            <Button
              variant="outline"
              size="sm"
              class="h-7 px-2 text-xs"
              onclick={() => {
                tokenDraft = ''
                saveDialogOpen = true
              }}
              disabled={anyBusy}
            >
              <KeyRound class="mr-1 size-3" />
              Save token
            </Button>
          {/if}
        </div>
      </div>

      <!-- Details (when configured) -->
      {#if credential.configured}
        <div class="border-border border-t px-4 py-3">
          <dl class="grid grid-cols-2 gap-x-4 gap-y-2.5 text-xs sm:grid-cols-3">
            {#if displayLogin()}
              <div>
                <dt class="text-muted-foreground">User</dt>
                <dd class="mt-0.5 font-medium">{displayLogin()}</dd>
              </div>
            {/if}
            <div>
              <dt class="text-muted-foreground">Token</dt>
              <dd class="mt-0.5 font-mono">{credential.token_preview}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Source</dt>
              <dd class="mt-0.5 capitalize">
                {credential.source ? credential.source.replaceAll('_', ' ') : '—'}
              </dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Repo access</dt>
              <dd class="mt-0.5 capitalize">{credential.probe.repo_access.replaceAll('_', ' ')}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Checked</dt>
              <dd class="mt-0.5">{formatCheckedAt(credential.probe.checked_at)}</dd>
            </div>
            {#if credential.probe.permissions.length}
              <div class="col-span-2 sm:col-span-3">
                <dt class="text-muted-foreground">Permissions</dt>
                <dd class="mt-0.5">{credential.probe.permissions.join(', ')}</dd>
              </div>
            {/if}
          </dl>
          {#if credential.probe.last_error}
            <p class="text-destructive mt-3 text-xs">{credential.probe.last_error}</p>
          {/if}
        </div>
      {/if}
    </div>

    <p class="text-muted-foreground text-xs leading-5">
      This credential is shared across all projects in the org. Projects with their own override in
      Security settings will use that instead.
    </p>
  {/if}
</div>

<OrganizationAdminCredentialsDialogs
  {credential}
  {saveDialogOpen}
  {deleteDialogOpen}
  {tokenDraft}
  {anyBusy}
  {actionKey}
  onSaveOpenChange={(open) => (saveDialogOpen = open)}
  onDeleteOpenChange={(open) => (deleteDialogOpen = open)}
  onTokenDraftChange={(value) => (tokenDraft = value)}
  onSave={() => mutate('save')}
  onDelete={() => mutate('delete')}
/>
