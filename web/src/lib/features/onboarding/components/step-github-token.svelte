<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import {
    importGitHubOutboundCredentialFromGHCLI,
    retestGitHubOutboundCredential,
    saveGitHubOutboundCredential,
  } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import {
    Github,
    Import,
    ClipboardPaste,
    Loader2,
    CheckCircle2,
    AlertCircle,
    User,
  } from '@lucide/svelte'
  import type { GitHubTokenState } from '../types'
  import { parseGitHubTokenState } from '../github'

  let {
    projectId,
    initialState,
    onComplete,
  }: {
    projectId: string
    initialState: GitHubTokenState
    onComplete: (updated: GitHubTokenState) => void
  } = $props()

  let mode = $state<'choose' | 'import' | 'paste'>('choose')
  let tokenInput = $state('')
  let importing = $state(false)
  let saving = $state(false)
  let probing = $state(false)
  let probeResult = $state<GitHubTokenState>({ ...untrack(() => initialState) })

  const isProbeValid = $derived(probeResult.probeStatus === 'valid')
  const isProbeInvalid = $derived(probeResult.probeStatus === 'invalid')

  async function handleImportFromCLI() {
    importing = true
    try {
      await importGitHubOutboundCredentialFromGHCLI(projectId, { scope: 'project' })
      toastStore.success('GitHub token imported from gh CLI.')
      await runProbe()
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : 'Import failed. Confirm gh CLI is installed locally and already signed in.',
      )
    } finally {
      importing = false
    }
  }

  async function handlePasteToken() {
    if (!tokenInput.trim()) {
      toastStore.error('Enter a GitHub token.')
      return
    }
    saving = true
    try {
      await saveGitHubOutboundCredential(projectId, {
        scope: 'project',
        token: tokenInput.trim(),
      })
      toastStore.success('GitHub token saved.')
      await runProbe()
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save the token.',
      )
    } finally {
      saving = false
    }
  }

  async function runProbe() {
    probing = true
    probeResult = { ...probeResult, probeStatus: 'testing' }
    try {
      const result = await retestGitHubOutboundCredential(projectId, { scope: 'project' })
      const parsed = parseGitHubTokenState(result)
      probeResult = {
        ...parsed,
        confirmed: false,
      }
      if (parsed.probeStatus !== 'valid') {
        toastStore.error('GitHub token validation failed. Check the token permissions.')
      }
    } catch (caughtError) {
      probeResult = { ...probeResult, probeStatus: 'invalid' }
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to verify GitHub identity.',
      )
    } finally {
      probing = false
    }
  }

  function handleConfirmIdentity() {
    const confirmed = { ...probeResult, confirmed: true }
    probeResult = confirmed
    onComplete(confirmed)
  }

  function handleRetry() {
    mode = 'choose'
    tokenInput = ''
    probeResult = { hasToken: false, probeStatus: 'unknown', login: '', confirmed: false }
  }
</script>

<div class="space-y-4">
  {#if isProbeValid && !probeResult.confirmed}
    <!-- Identity confirmation -->
    <div
      class="rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-900/50 dark:bg-emerald-950/30"
    >
      <div class="flex items-center gap-3">
        <div
          class="flex size-10 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-900/50"
        >
          <User class="size-5 text-emerald-600 dark:text-emerald-400" />
        </div>
        <div class="flex-1">
          <p class="text-foreground text-sm font-medium">GitHub identity verified</p>
          <p class="text-muted-foreground text-sm">
            OpenASE will use the GitHub account <span class="text-foreground font-medium"
              >{probeResult.login}</span
            >
          </p>
        </div>
      </div>
      <div class="mt-3 flex items-center gap-2">
        <Button size="sm" onclick={handleConfirmIdentity}>
          <CheckCircle2 class="mr-1.5 size-3.5" />
          Use this GitHub identity for the project
        </Button>
        <Button variant="ghost" size="sm" onclick={handleRetry}>Change token</Button>
      </div>
    </div>
  {:else if probeResult.confirmed}
    <!-- Completed state -->
    <div
      class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-900/50 dark:bg-emerald-950/30"
    >
      <CheckCircle2 class="size-5 shrink-0 text-emerald-600 dark:text-emerald-400" />
      <div>
        <p class="text-foreground text-sm font-medium">GitHub connected</p>
        <p class="text-muted-foreground text-xs">Account: {probeResult.login}</p>
      </div>
      <Button variant="ghost" size="sm" class="ml-auto" onclick={handleRetry}>Reconfigure</Button>
    </div>
  {:else if probing}
    <!-- Probing state -->
    <div class="border-border flex items-center gap-3 rounded-lg border p-4">
      <Loader2 class="text-primary size-5 animate-spin" />
      <p class="text-muted-foreground text-sm">Verifying GitHub identity and permissions...</p>
    </div>
  {:else if isProbeInvalid}
    <!-- Probe failed -->
    <div class="bg-destructive/5 border-destructive/30 rounded-lg border p-4">
      <div class="flex items-center gap-2">
        <AlertCircle class="text-destructive size-4" />
        <p class="text-destructive text-sm font-medium">Token validation failed</p>
      </div>
      <p class="text-muted-foreground mt-1 text-xs">
        Check the token permissions. Repository read/write access is required.
      </p>
      <Button variant="outline" size="sm" class="mt-2" onclick={handleRetry}>Retry</Button>
    </div>
  {:else}
    <!-- Mode selection -->
    {#if mode === 'choose'}
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <button
          type="button"
          class="border-border hover:border-primary/50 hover:bg-primary/5 flex items-start gap-3 rounded-lg border p-4 text-left transition-colors"
          onclick={() => (mode = 'import')}
        >
          <div class="bg-primary/10 flex size-9 shrink-0 items-center justify-center rounded-lg">
            <Import class="text-primary size-4" />
          </div>
          <div>
            <p class="text-foreground text-sm font-medium">Import automatically from local gh</p>
            <p class="text-muted-foreground mt-0.5 text-xs">
              Detect and import the local <code class="bg-muted rounded px-1 text-[10px]"
                >gh auth token</code
              > automatically
            </p>
          </div>
        </button>

        <button
          type="button"
          class="border-border hover:border-primary/50 hover:bg-primary/5 flex items-start gap-3 rounded-lg border p-4 text-left transition-colors"
          onclick={() => (mode = 'paste')}
        >
          <div class="bg-primary/10 flex size-9 shrink-0 items-center justify-center rounded-lg">
            <ClipboardPaste class="text-primary size-4" />
          </div>
          <div>
            <p class="text-foreground text-sm font-medium">Paste a token manually</p>
            <p class="text-muted-foreground mt-0.5 text-xs">
              Run <code class="bg-muted rounded px-1 text-[10px]">gh auth token</code> and paste the result
            </p>
          </div>
        </button>
      </div>
    {:else if mode === 'import'}
      <div class="space-y-3">
        <p class="text-muted-foreground text-sm">
          Make sure GitHub CLI is installed and already signed in. OpenASE will read the current
          token automatically.
        </p>
        <div class="bg-muted/50 rounded-md p-3">
          <p class="text-muted-foreground mb-1 text-xs">If you are not signed in yet:</p>
          <code class="text-foreground text-xs">gh auth login</code>
        </div>
        <div class="flex items-center gap-2">
          <Button onclick={handleImportFromCLI} disabled={importing}>
            {#if importing}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              Importing...
            {:else}
              <Github class="mr-1.5 size-3.5" />
              Import now
            {/if}
          </Button>
          <Button variant="ghost" size="sm" onclick={() => (mode = 'choose')}>Back</Button>
        </div>
      </div>
    {:else}
      <div class="space-y-3">
        <p class="text-muted-foreground text-sm">Paste the token below after you obtain it:</p>
        <div class="bg-muted/50 space-y-1 rounded-md p-3">
          <p class="text-muted-foreground text-xs">
            Run this command in the terminal to get the token:
          </p>
          <code class="text-foreground text-xs">gh auth token</code>
        </div>
        <div class="flex items-center gap-2">
          <Input
            type="password"
            placeholder="ghp_xxxxxxxxxxxx"
            bind:value={tokenInput}
            class="flex-1"
          />
          <Button onclick={handlePasteToken} disabled={saving || !tokenInput.trim()}>
            {#if saving}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              Saving...
            {:else}
              Save token
            {/if}
          </Button>
        </div>
        <Button variant="ghost" size="sm" onclick={() => (mode = 'choose')}>Back</Button>
      </div>
    {/if}
  {/if}
</div>
