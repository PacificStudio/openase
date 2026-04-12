<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import {
    importGitHubOutboundCredentialFromGHCLI,
    retestGitHubOutboundCredential,
    saveGitHubOutboundCredential,
  } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
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

  const STEP_KEY = 'onboarding.step.githubToken'
  const KEYS = {
    welcomeTitle: `${STEP_KEY}.welcomeTitle`,
    welcomeDescription: `${STEP_KEY}.welcomeDescription`,
    loadingStatus: `${STEP_KEY}.loadingStatus`,
    identityVerifiedTitle: `${STEP_KEY}.identityVerifiedTitle`,
    identityVerifiedBody: `${STEP_KEY}.identityVerifiedBody`,
    importSuccess: `${STEP_KEY}.importSuccess`,
    importFailed: `${STEP_KEY}.importFailed`,
    confirmIdentity: `${STEP_KEY}.confirmIdentity`,
    changeToken: `${STEP_KEY}.changeToken`,
    connectedTitle: `${STEP_KEY}.connectedTitle`,
    connectedAccount: `${STEP_KEY}.connectedAccount`,
    reconfigure: `${STEP_KEY}.reconfigure`,
    verifyingIdentity: `${STEP_KEY}.verifyingIdentity`,
    validationFailedTitle: `${STEP_KEY}.validationFailedTitle`,
    validationFailedMessage: `${STEP_KEY}.validationFailedMessage`,
    tokenRequired: `${STEP_KEY}.tokenRequired`,
    saveSuccess: `${STEP_KEY}.saveSuccess`,
    saveFailed: `${STEP_KEY}.saveFailed`,
    validationFailedToast: `${STEP_KEY}.validationFailedToast`,
    verifyFailed: `${STEP_KEY}.verifyFailed`,
    chooseImportTitle: `${STEP_KEY}.chooseImportTitle`,
    chooseImportDescription: `${STEP_KEY}.chooseImportDescription`,
    choosePasteTitle: `${STEP_KEY}.choosePasteTitle`,
    choosePasteDescription: `${STEP_KEY}.choosePasteDescription`,
    importHint: `${STEP_KEY}.importHint`,
    importSignInHint: `${STEP_KEY}.importSignInHint`,
    importing: `${STEP_KEY}.importing`,
    importNow: `${STEP_KEY}.importNow`,
    pastePrompt: `${STEP_KEY}.pastePrompt`,
    pasteCommandHint: `${STEP_KEY}.pasteCommandHint`,
    tokenPlaceholder: `${STEP_KEY}.tokenPlaceholder`,
    saving: `${STEP_KEY}.saving`,
    saveToken: `${STEP_KEY}.saveToken`,
    back: `${STEP_KEY}.back`,
  } as const

  type StepCopyKey = (typeof KEYS)[keyof typeof KEYS]

  const copy: Record<
    StepCopyKey,
    string | ((params?: Record<string, string | number>) => string)
  > = {
    [KEYS.welcomeTitle]: ({ projectName } = { projectName: '' }) => `Welcome to ${projectName}`,
    [KEYS.welcomeDescription]:
      'The project is created. Next, we will guide you through getting it into a runnable state. After these steps complete, an agent can start working.',
    [KEYS.loadingStatus]: 'Loading setup status...',
    [KEYS.identityVerifiedTitle]: 'GitHub identity verified',
    [KEYS.identityVerifiedBody]: ({ login } = { login: '' }) =>
      `OpenASE will use the GitHub account ${login} once you confirm it.`,
    [KEYS.importSuccess]: 'GitHub token imported from gh CLI.',
    [KEYS.importFailed]:
      'Import failed. Confirm gh CLI is installed locally and already signed in.',
    [KEYS.confirmIdentity]: 'Use this GitHub identity for the project',
    [KEYS.changeToken]: 'Change token',
    [KEYS.connectedTitle]: 'GitHub connected',
    [KEYS.connectedAccount]: ({ login } = { login: '' }) => `Account: ${login}`,
    [KEYS.reconfigure]: 'Reconfigure',
    [KEYS.verifyingIdentity]: 'Verifying GitHub identity and permissions...',
    [KEYS.validationFailedTitle]: 'Token validation failed',
    [KEYS.validationFailedMessage]: 'Check the token permissions. Repository read/write access is required.',
    [KEYS.tokenRequired]: 'Enter a GitHub token.',
    [KEYS.saveSuccess]: 'GitHub token saved.',
    [KEYS.saveFailed]: 'Failed to save the token.',
    [KEYS.validationFailedToast]: 'GitHub token validation failed. Check the token permissions.',
    [KEYS.verifyFailed]: 'Failed to verify GitHub identity.',
    [KEYS.chooseImportTitle]: 'Import automatically from local gh',
    [KEYS.chooseImportDescription]:
      'Detect and import the local gh auth token automatically.',
    [KEYS.choosePasteTitle]: 'Paste a token manually',
    [KEYS.choosePasteDescription]:
      'Run gh auth token and paste the result.',
    [KEYS.importHint]:
      'Make sure GitHub CLI is installed and already signed in. OpenASE will read the current token automatically.',
    [KEYS.importSignInHint]: 'If you are not signed in yet:',
    [KEYS.importing]: 'Importing...',
    [KEYS.importNow]: 'Import now',
    [KEYS.pastePrompt]: 'Paste the token below after you obtain it:',
    [KEYS.pasteCommandHint]: 'Run this command in the terminal to get the token:',
    [KEYS.tokenPlaceholder]: 'ghp_xxxxxxxxxxxx',
    [KEYS.saving]: 'Saving...',
    [KEYS.saveToken]: 'Save token',
    [KEYS.back]: 'Back',
  }

  function t(key: StepCopyKey, params?: Record<string, string | number>) {
    const value = copy[key]
    if (typeof value === 'function') {
      return value(params)
    }
    return value ?? ''
  }

  async function handleImportFromCLI() {
    importing = true
    try {
      await importGitHubOutboundCredentialFromGHCLI(projectId)
      toastStore.success(t(KEYS.importSuccess))
      await runProbe()
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : t(KEYS.importFailed),
      )
    } finally {
      importing = false
    }
  }

  async function handlePasteToken() {
    if (!tokenInput.trim()) {
      toastStore.error(t(KEYS.tokenRequired))
      return
    }
    saving = true
    try {
      await saveGitHubOutboundCredential(projectId, {
        token: tokenInput.trim(),
      })
      toastStore.success(t(KEYS.saveSuccess))
      await runProbe()
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : t(KEYS.saveFailed),
      )
    } finally {
      saving = false
    }
  }

  async function runProbe() {
    probing = true
    probeResult = { ...probeResult, probeStatus: 'testing' }
    try {
      const result = await retestGitHubOutboundCredential(projectId)
      const parsed = parseGitHubTokenState(result)
      probeResult = {
        ...parsed,
        confirmed: false,
      }
      if (parsed.probeStatus !== 'valid') {
        toastStore.error(t(KEYS.validationFailedToast))
      }
    } catch (caughtError) {
      probeResult = { ...probeResult, probeStatus: 'invalid' }
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : t(KEYS.verifyFailed),
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
          <p class="text-foreground text-sm font-medium">{t(KEYS.identityVerifiedTitle)}</p>
          <p class="text-muted-foreground text-sm">
            {t(KEYS.identityVerifiedBody, { login: probeResult.login })}
          </p>
        </div>
      </div>
      <div class="mt-3 flex items-center gap-2">
        <Button size="sm" onclick={handleConfirmIdentity}>
          <CheckCircle2 class="mr-1.5 size-3.5" />
          {t(KEYS.confirmIdentity)}
        </Button>
        <Button variant="ghost" size="sm" onclick={handleRetry}>{t(KEYS.changeToken)}</Button>
      </div>
    </div>
  {:else if probeResult.confirmed}
    <!-- Completed state -->
    <div
      class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-900/50 dark:bg-emerald-950/30"
    >
      <CheckCircle2 class="size-5 shrink-0 text-emerald-600 dark:text-emerald-400" />
      <div>
            <p class="text-foreground text-sm font-medium">{t(KEYS.connectedTitle)}</p>
            <p class="text-muted-foreground text-xs">
              {t(KEYS.connectedAccount, { login: probeResult.login })}
            </p>
      </div>
          <Button variant="ghost" size="sm" class="ml-auto" onclick={handleRetry}>
            {t(KEYS.reconfigure)}
      </Button>
    </div>
  {:else if probing}
    <!-- Probing state -->
    <div class="border-border flex items-center gap-3 rounded-lg border p-4">
      <Loader2 class="text-primary size-5 animate-spin" />
      <p class="text-muted-foreground text-sm">{t(KEYS.verifyingIdentity)}</p>
    </div>
  {:else if isProbeInvalid}
    <!-- Probe failed -->
    <div class="bg-destructive/5 border-destructive/30 rounded-lg border p-4">
      <div class="flex items-center gap-2">
        <AlertCircle class="text-destructive size-4" />
        <p class="text-destructive text-sm font-medium">{t(KEYS.validationFailedTitle)}</p>
      </div>
      <p class="text-muted-foreground mt-1 text-xs">
        {t(KEYS.validationFailedMessage)}
      </p>
      <Button
        variant="outline"
        size="sm"
        class="mt-2"
        onclick={handleRetry}
      >
        {i18nStore.t('common.retry')}
      </Button>
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
            <p class="text-foreground text-sm font-medium">{t(KEYS.chooseImportTitle)}</p>
            <p class="text-muted-foreground mt-0.5 text-xs">
              {t(KEYS.chooseImportDescription)}
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
            <p class="text-foreground text-sm font-medium">{t(KEYS.choosePasteTitle)}</p>
            <p class="text-muted-foreground mt-0.5 text-xs">
              {t(KEYS.choosePasteDescription)}
            </p>
          </div>
        </button>
      </div>
    {:else if mode === 'import'}
      <div class="space-y-3">
        <p class="text-muted-foreground text-sm">{t(KEYS.importHint)}</p>
        <div class="bg-muted/50 rounded-md p-3">
          <p class="text-muted-foreground mb-1 text-xs">{t(KEYS.importSignInHint)}</p>
          <code class="text-foreground text-xs">gh auth login</code>
        </div>
        <div class="flex items-center gap-2">
          <Button onclick={handleImportFromCLI} disabled={importing}>
            {#if importing}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              {t(KEYS.importing)}
            {:else}
              <Github class="mr-1.5 size-3.5" />
              {t(KEYS.importNow)}
            {/if}
          </Button>
          <Button variant="ghost" size="sm" onclick={() => (mode = 'choose')}>
            {t(KEYS.back)}
          </Button>
        </div>
      </div>
    {:else}
      <div class="space-y-3">
        <p class="text-muted-foreground text-sm">{t(KEYS.pastePrompt)}</p>
        <div class="bg-muted/50 space-y-1 rounded-md p-3">
          <p class="text-muted-foreground text-xs">
            {t(KEYS.pasteCommandHint)}
          </p>
          <code class="text-foreground text-xs">gh auth token</code>
        </div>
        <div class="flex items-center gap-2">
          <Input
            type="password"
            placeholder={t(KEYS.tokenPlaceholder)}
            bind:value={tokenInput}
            class="flex-1"
          />
          <Button onclick={handlePasteToken} disabled={saving || !tokenInput.trim()}>
            {#if saving}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              {t(KEYS.saving)}
            {:else}
              {t(KEYS.saveToken)}
            {/if}
          </Button>
        </div>
        <Button variant="ghost" size="sm" onclick={() => (mode = 'choose')}>
          {t(KEYS.back)}
        </Button>
      </div>
    {/if}
  {/if}
</div>
