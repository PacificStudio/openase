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
      toastStore.success('GitHub Token 已从 gh CLI 导入。')
      await runProbe()
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : '导入失败，请确认本机已安装 gh CLI 并已登录。',
      )
    } finally {
      importing = false
    }
  }

  async function handlePasteToken() {
    if (!tokenInput.trim()) {
      toastStore.error('请输入 GitHub Token。')
      return
    }
    saving = true
    try {
      await saveGitHubOutboundCredential(projectId, {
        scope: 'project',
        token: tokenInput.trim(),
      })
      toastStore.success('GitHub Token 已保存。')
      await runProbe()
    } catch (caughtError) {
      toastStore.error(caughtError instanceof ApiError ? caughtError.detail : '保存 Token 失败。')
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
        toastStore.error('GitHub Token 验证失败，请检查 Token 权限。')
      }
    } catch (caughtError) {
      probeResult = { ...probeResult, probeStatus: 'invalid' }
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : '验证 GitHub 身份失败。',
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
          <p class="text-foreground text-sm font-medium">GitHub 身份验证成功</p>
          <p class="text-muted-foreground text-sm">
            将使用 GitHub 账号 <span class="text-foreground font-medium">{probeResult.login}</span>
          </p>
        </div>
      </div>
      <div class="mt-3 flex items-center gap-2">
        <Button size="sm" onclick={handleConfirmIdentity}>
          <CheckCircle2 class="mr-1.5 size-3.5" />
          这是我要用于该项目的 GitHub 身份
        </Button>
        <Button variant="ghost" size="sm" onclick={handleRetry}>更换 Token</Button>
      </div>
    </div>
  {:else if probeResult.confirmed}
    <!-- Completed state -->
    <div
      class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-900/50 dark:bg-emerald-950/30"
    >
      <CheckCircle2 class="size-5 shrink-0 text-emerald-600 dark:text-emerald-400" />
      <div>
        <p class="text-foreground text-sm font-medium">已连接 GitHub</p>
        <p class="text-muted-foreground text-xs">账号：{probeResult.login}</p>
      </div>
      <Button variant="ghost" size="sm" class="ml-auto" onclick={handleRetry}>重新配置</Button>
    </div>
  {:else if probing}
    <!-- Probing state -->
    <div class="border-border flex items-center gap-3 rounded-lg border p-4">
      <Loader2 class="text-primary size-5 animate-spin" />
      <p class="text-muted-foreground text-sm">正在验证 GitHub 身份与权限...</p>
    </div>
  {:else if isProbeInvalid}
    <!-- Probe failed -->
    <div class="bg-destructive/5 border-destructive/30 rounded-lg border p-4">
      <div class="flex items-center gap-2">
        <AlertCircle class="text-destructive size-4" />
        <p class="text-destructive text-sm font-medium">Token 验证失败</p>
      </div>
      <p class="text-muted-foreground mt-1 text-xs">请检查 Token 权限，需要 repo 读写权限。</p>
      <Button variant="outline" size="sm" class="mt-2" onclick={handleRetry}>重试</Button>
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
            <p class="text-foreground text-sm font-medium">从本机 gh 自动导入</p>
            <p class="text-muted-foreground mt-0.5 text-xs">
              自动检测本机 <code class="bg-muted rounded px-1 text-[10px]">gh auth token</code> 并导入
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
            <p class="text-foreground text-sm font-medium">手动粘贴 Token</p>
            <p class="text-muted-foreground mt-0.5 text-xs">
              通过 <code class="bg-muted rounded px-1 text-[10px]">gh auth token</code> 获取后粘贴
            </p>
          </div>
        </button>
      </div>
    {:else if mode === 'import'}
      <div class="space-y-3">
        <p class="text-muted-foreground text-sm">
          确保已安装 GitHub CLI 并已登录，平台会自动读取当前认证 Token。
        </p>
        <div class="bg-muted/50 rounded-md p-3">
          <p class="text-muted-foreground mb-1 text-xs">如果尚未登录：</p>
          <code class="text-foreground text-xs">gh auth login</code>
        </div>
        <div class="flex items-center gap-2">
          <Button onclick={handleImportFromCLI} disabled={importing}>
            {#if importing}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              正在导入...
            {:else}
              <Github class="mr-1.5 size-3.5" />
              一键导入
            {/if}
          </Button>
          <Button variant="ghost" size="sm" onclick={() => (mode = 'choose')}>返回</Button>
        </div>
      </div>
    {:else}
      <div class="space-y-3">
        <p class="text-muted-foreground text-sm">获取 Token 后粘贴到下方：</p>
        <div class="bg-muted/50 space-y-1 rounded-md p-3">
          <p class="text-muted-foreground text-xs">在终端执行以下命令获取 Token：</p>
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
              保存中...
            {:else}
              保存 Token
            {/if}
          </Button>
        </div>
        <Button variant="ghost" size="sm" onclick={() => (mode = 'choose')}>返回</Button>
      </div>
    {/if}
  {/if}
</div>
