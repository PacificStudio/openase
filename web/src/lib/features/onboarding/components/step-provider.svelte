<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import { listProviders, refreshMachineHealth, updateProject } from '$lib/api/openase'
  import type { AgentProvider } from '$lib/api/contracts'
  import {
    adapterDisplayName,
    adapterIconPath,
    providerAvailabilityCheckedAtText,
    providerAvailabilityDescription,
    providerAvailabilityHeadline,
  } from '$lib/features/providers'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import * as Sheet from '$ui/sheet'
  import {
    AlertTriangle,
    CheckCircle2,
    CircleDashed,
    Copy,
    ExternalLink,
    Loader2,
    LogIn,
    RefreshCcw,
    SearchCheck,
    TerminalSquare,
    Wrench,
    Zap,
  } from '@lucide/svelte'
  import type { ProviderState } from '../types'

  type GuideKey = 'claude-code' | 'codex' | 'gemini'
  type GuideCommand = {
    label: string
    command: string
  }
  type ProviderGuide = {
    key: GuideKey
    title: string
    adapterTypes: string[]
    docsUrl: string
    docsLabel: string
    recommendedModel: string
    installCommands: GuideCommand[]
    authCommands: GuideCommand[]
    verifyCommands: GuideCommand[]
    commonFixHints: string[]
  }
  type DetectionStatus = 'yes' | 'no' | 'pending'

  const providerGuides: ProviderGuide[] = [
    {
      key: 'claude-code',
      title: 'Claude Code',
      adapterTypes: ['claude-code-cli', 'claude_code'],
      docsUrl: 'https://code.claude.com/docs/en/getting-started',
      docsLabel: 'Claude Code 官方文档',
      recommendedModel: 'claude-opus-4-6',
      installCommands: [
        {
          label: '官方推荐安装',
          command: 'curl -fsSL https://claude.ai/install.sh | bash',
        },
      ],
      authCommands: [
        {
          label: '首次登录',
          command: 'claude',
        },
      ],
      verifyCommands: [
        {
          label: '确认 CLI 可执行',
          command: 'claude --version',
        },
      ],
      commonFixHints: [
        'Windows 环境需要先安装 Git for Windows，Claude Code 会依赖 Git Bash。',
        '如果账号没有 Claude Code 权限，登录后仍会保持不可用状态。',
      ],
    },
    {
      key: 'codex',
      title: 'OpenAI Codex',
      adapterTypes: ['codex-app-server', 'codex'],
      docsUrl: 'https://developers.openai.com/codex/cli',
      docsLabel: 'Codex CLI 官方文档',
      recommendedModel: 'gpt-5.4',
      installCommands: [
        {
          label: '全局安装',
          command: 'npm i -g @openai/codex',
        },
      ],
      authCommands: [
        {
          label: '登录 ChatGPT / API',
          command: 'codex --login',
        },
      ],
      verifyCommands: [
        {
          label: '确认 CLI 可执行',
          command: 'codex --version',
        },
      ],
      commonFixHints: [
        '如果命令存在但无法执行，优先确认 npm 全局 bin 目录已经加入 PATH。',
        '如果登录状态丢失，重新运行 `codex --login`，或检查 API key / ChatGPT 计划权限。',
      ],
    },
    {
      key: 'gemini',
      title: 'Gemini CLI',
      adapterTypes: ['gemini-cli', 'gemini_cli'],
      docsUrl: 'https://github.com/google-gemini/gemini-cli',
      docsLabel: 'Gemini CLI 官方仓库',
      recommendedModel: 'gemini-2.5-pro',
      installCommands: [
        {
          label: '全局安装',
          command: 'npm install -g @google/gemini-cli',
        },
      ],
      authCommands: [
        {
          label: '首次登录 / 选择鉴权方式',
          command: 'gemini',
        },
      ],
      verifyCommands: [
        {
          label: '确认 CLI 可执行',
          command: 'gemini --version',
        },
      ],
      commonFixHints: [
        '首次启动时可以选择 Google 账号登录，也可以改用 `GEMINI_API_KEY` / Vertex AI。',
        '如果命令不存在，先确认 Node 18+ 与 npm 全局 bin 目录都在 PATH 中。',
      ],
    },
  ]

  const availabilityLabel: Record<string, { text: string; className: string }> = {
    available: { text: '可用', className: 'text-emerald-600 dark:text-emerald-400' },
    ready: { text: '可用', className: 'text-emerald-600 dark:text-emerald-400' },
    unavailable: { text: '不可用', className: 'text-destructive' },
    stale: { text: '状态过期', className: 'text-amber-600 dark:text-amber-400' },
    unknown: { text: '待检测', className: 'text-muted-foreground' },
  }

  const cliDetectionLabel: Record<DetectionStatus, string> = {
    yes: '已检测到',
    no: '未检测到',
    pending: '待配置',
  }

  const authDetectionLabel: Record<DetectionStatus, string> = {
    yes: '已登录',
    no: '未登录',
    pending: '待确认',
  }

  let {
    projectId,
    orgId,
    initialState,
    onComplete,
    onStateChange = () => {},
  }: {
    projectId: string
    orgId: string
    initialState: ProviderState
    onComplete: (providerId: string) => void
    onStateChange?: (nextState: ProviderState) => void
  } = $props()

  let selecting = $state(false)
  let selectedId = $state(untrack(() => initialState.selectedProviderId))
  let providers = $state<AgentProvider[]>(untrack(() => [...initialState.providers]))
  let guideOpen = $state(false)
  let activeGuideKey = $state<GuideKey | null>(null)
  let refreshingTokens = $state<string[]>([])
  let autoRefreshStarted = $state(false)

  const activeGuide = $derived(providerGuides.find((guide) => guide.key === activeGuideKey) ?? null)

  $effect(() => {
    providers = [...initialState.providers]
    selectedId = initialState.selectedProviderId
  })

  $effect(() => {
    if (autoRefreshStarted) {
      return
    }

    autoRefreshStarted = true
    void refreshProviders(uniqueMachineIds(providers), true)
  })

  function isProviderAvailable(provider: AgentProvider): boolean {
    return provider.availability_state === 'available' || provider.availability_state === 'ready'
  }

  function matchesGuide(provider: AgentProvider, guide: ProviderGuide): boolean {
    return guide.adapterTypes.includes(provider.adapter_type)
  }

  function guideProviders(guide: ProviderGuide): AgentProvider[] {
    return providers.filter((provider) => matchesGuide(provider, guide))
  }

  function guideForProvider(provider: AgentProvider): ProviderGuide | null {
    return providerGuides.find((guide) => matchesGuide(provider, guide)) ?? null
  }

  function primaryGuideProvider(items: AgentProvider[]): AgentProvider | null {
    return (
      items.find((provider) => provider.id === selectedId) ??
      items.find((provider) => isProviderAvailable(provider)) ??
      items[0] ??
      null
    )
  }

  function uniqueMachineIds(items: AgentProvider[]): string[] {
    return Array.from(
      new Set(
        items
          .map((provider) => provider.machine_id)
          .filter((value): value is string => Boolean(value && value.trim().length > 0)),
      ),
    )
  }

  function refreshToken(machineIds: string[]): string {
    return machineIds.length > 0 ? machineIds.slice().sort().join('|') : 'provider-list'
  }

  function isRefreshing(machineIds: string[]): boolean {
    return refreshingTokens.includes(refreshToken(machineIds))
  }

  function cliDetectionState(items: AgentProvider[]): DetectionStatus {
    if (items.some((provider) => isProviderAvailable(provider))) {
      return 'yes'
    }
    if (items.some((provider) => provider.availability_reason === 'cli_missing')) {
      return 'no'
    }
    if (items.length === 0) {
      return 'pending'
    }
    return 'yes'
  }

  function authDetectionState(items: AgentProvider[]): DetectionStatus {
    if (items.some((provider) => isProviderAvailable(provider))) {
      return 'yes'
    }
    if (items.some((provider) => provider.availability_reason === 'not_logged_in')) {
      return 'no'
    }
    return 'pending'
  }

  function syncProviderState(nextProviders: AgentProvider[], nextSelectedId = selectedId) {
    providers = nextProviders
    onStateChange({
      providers: nextProviders,
      selectedProviderId: nextSelectedId,
    })
  }

  async function refreshProviders(machineIds: string[], silent = false) {
    const token = refreshToken(machineIds)
    if (refreshingTokens.includes(token)) {
      return
    }

    refreshingTokens = [...refreshingTokens, token]

    try {
      await Promise.allSettled(machineIds.map((machineId) => refreshMachineHealth(machineId)))
      const payload = await listProviders(orgId)
      syncProviderState(payload.providers)
      if (!silent) {
        toastStore.success('已重新检测 Provider 可用性。')
      }
    } catch (caughtError) {
      if (!silent) {
        toastStore.error(
          caughtError instanceof ApiError ? caughtError.detail : '重新检测 Provider 失败。',
        )
      }
    } finally {
      refreshingTokens = refreshingTokens.filter((value) => value !== token)
    }
  }

  async function handleSelectProvider(providerId: string) {
    if (selecting) return

    const previousSelectedId = selectedId
    selectedId = providerId
    selecting = true

    try {
      const payload = await updateProject(projectId, {
        default_agent_provider_id: providerId,
      })
      appStore.currentProject = payload.project
      syncProviderState(providers, providerId)
      toastStore.success('已设为默认 Provider。')
      onComplete(providerId)
    } catch (caughtError) {
      selectedId = previousSelectedId
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : '设置默认 Provider 失败。',
      )
    } finally {
      selecting = false
    }
  }

  function openGuide(key: GuideKey) {
    activeGuideKey = key
    guideOpen = true
  }

  async function copyCommand(command: string) {
    try {
      await navigator.clipboard.writeText(command)
      toastStore.success('命令已复制。')
    } catch {
      toastStore.error('复制命令失败。')
    }
  }

  function reasonSpecificHints(provider: AgentProvider | null): string[] {
    switch (provider?.availability_reason) {
      case 'machine_offline':
        return ['绑定机器当前离线，先恢复机器在线状态，再回到这里执行重新检测。']
      case 'machine_degraded':
        return ['机器已连接但处于 degraded 状态，先修复主机健康问题，再重新检测。']
      case 'machine_maintenance':
        return ['机器处于维护模式，退出维护后再重新检测。']
      case 'cli_missing':
        return ['OpenASE 已找到这个 Provider，但在目标机器 PATH 中没有检测到对应 CLI。']
      case 'not_logged_in':
        return ['CLI 已安装，但当前认证缺失或已过期。重新执行登录命令后再检测。']
      case 'not_ready':
        return ['CLI 存在但 readiness probe 仍未通过，优先先跑一次验证命令确认具体报错。']
      case 'config_incomplete':
        return ['当前 Provider 注册信息不完整，检查 CLI command、模型和机器绑定是否都已填写。']
      case 'l4_snapshot_missing':
      case 'stale_l4_snapshot':
        return ['OpenASE 还没有拿到可用的最新机器快照，执行重新检测会重新拉取快照。']
      default:
        return []
    }
  }

  function providerStatus(provider: AgentProvider | null) {
    if (!provider) {
      return availabilityLabel.unknown
    }
    return availabilityLabel[provider.availability_state] ?? availabilityLabel.unknown
  }
</script>

<div class="space-y-6">
  <div class="border-border bg-card space-y-4 rounded-xl border p-4">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
      <div class="space-y-1">
        <h4 class="text-foreground text-sm font-semibold">内置 CLI 配置向导</h4>
        <p class="text-muted-foreground text-xs">
          进入此步骤时会基于已注册 Provider 的绑定机器重新拉取可用性。即使你还没注册
          Provider，也可以先按下面的官方指引完成安装、登录与验证。
        </p>
      </div>
      <Button
        variant="outline"
        size="sm"
        disabled={isRefreshing(uniqueMachineIds(providers))}
        onclick={() => void refreshProviders(uniqueMachineIds(providers))}
      >
        {#if isRefreshing(uniqueMachineIds(providers))}
          <Loader2 class="mr-1.5 size-3.5 animate-spin" />
          检测中...
        {:else}
          <RefreshCcw class="mr-1.5 size-3.5" />
          重新检测全部 Provider
        {/if}
      </Button>
    </div>

    <div class="grid grid-cols-1 gap-3 xl:grid-cols-3">
      {#each providerGuides as guide (guide.key)}
        {@const matchingProviders = guideProviders(guide)}
        {@const availableProviders = matchingProviders.filter((provider) =>
          isProviderAvailable(provider),
        )}
        {@const primaryProvider = primaryGuideProvider(matchingProviders)}
        {@const status = providerStatus(primaryProvider)}
        {@const cliState = cliDetectionState(matchingProviders)}
        {@const authState = authDetectionState(matchingProviders)}
        {@const iconPath = adapterIconPath(guide.adapterTypes[0] ?? '')}

        <div class="border-border bg-background flex h-full flex-col rounded-xl border p-4">
          <div class="mb-3 flex items-start gap-3">
            <div class="bg-muted flex size-11 shrink-0 items-center justify-center rounded-xl">
              {#if iconPath}
                <img src={iconPath} alt="" class="size-6" />
              {:else}
                <Zap class="text-foreground size-5" />
              {/if}
            </div>
            <div class="min-w-0 flex-1 space-y-1">
              <div class="flex items-center gap-2">
                <p class="text-foreground text-sm font-semibold">{guide.title}</p>
                <span class={cn('text-xs', status.className)}>{status.text}</span>
              </div>
              <p class="text-muted-foreground text-xs">
                {matchingProviders.length > 0
                  ? `已注册 ${matchingProviders.length} 个 Provider`
                  : '尚未注册 Provider'}
              </p>
            </div>
          </div>

          <div class="mb-3 grid grid-cols-2 gap-2 text-xs">
            <div class="bg-muted/50 rounded-lg px-3 py-2">
              <p class="text-muted-foreground">CLI</p>
              <p class="text-foreground mt-1 font-medium">{cliDetectionLabel[cliState]}</p>
            </div>
            <div class="bg-muted/50 rounded-lg px-3 py-2">
              <p class="text-muted-foreground">登录</p>
              <p class="text-foreground mt-1 font-medium">{authDetectionLabel[authState]}</p>
            </div>
            <div class="bg-muted/50 rounded-lg px-3 py-2">
              <p class="text-muted-foreground">推荐模型</p>
              <p class="text-foreground mt-1 font-medium">{guide.recommendedModel}</p>
            </div>
            <div class="bg-muted/50 rounded-lg px-3 py-2">
              <p class="text-muted-foreground">默认入口</p>
              <p class="text-foreground mt-1 font-medium">{guide.title}</p>
            </div>
          </div>

          {#if primaryProvider}
            <div class="border-border bg-muted/30 mb-3 rounded-lg border px-3 py-2 text-xs">
              <p class="text-foreground font-medium">
                {providerAvailabilityHeadline(
                  primaryProvider.availability_state,
                  primaryProvider.availability_reason,
                )}
              </p>
              <p class="text-muted-foreground mt-1">
                {providerAvailabilityDescription(
                  primaryProvider.availability_state,
                  primaryProvider.availability_reason,
                )}
              </p>
            </div>
          {/if}

          <div class="mt-auto flex gap-2">
            {#if availableProviders.length === 1}
              <Button
                size="sm"
                class="flex-1"
                variant={selectedId === availableProviders[0]?.id ? 'default' : 'outline'}
                disabled={selecting}
                onclick={() => void handleSelectProvider(availableProviders[0]!.id)}
              >
                {#if selecting && selectedId === availableProviders[0]?.id}
                  <Loader2 class="mr-1.5 size-3.5 animate-spin" />
                  设置中...
                {:else if selectedId === availableProviders[0]?.id}
                  已设为默认
                {:else}
                  使用这个 Provider
                {/if}
              </Button>
            {:else}
              <Button
                size="sm"
                class="flex-1"
                variant="outline"
                onclick={() => openGuide(guide.key)}
              >
                {availableProviders.length > 1
                  ? `查看 ${availableProviders.length} 个实例`
                  : '继续配置'}
              </Button>
            {/if}

            <Button size="sm" variant="ghost" onclick={() => openGuide(guide.key)}>指南</Button>
          </div>
        </div>
      {/each}
    </div>
  </div>

  <div class="space-y-3">
    <div class="space-y-1">
      <h4 class="text-foreground text-sm font-semibold">已注册 Provider</h4>
      <p class="text-muted-foreground text-xs">
        已注册的 Provider 会继续保留精确的机器、模型与可用性信息。不可用实例可以直接回到对应 CLI
        指南，并在完成配置后重新检测。
      </p>
    </div>

    {#if providers.length === 0}
      <div class="border-border rounded-xl border border-dashed p-6 text-center">
        <CircleDashed class="text-muted-foreground mx-auto mb-2 size-8" />
        <p class="text-foreground text-sm font-medium">还没有注册任何 Provider。</p>
        <p class="text-muted-foreground mt-1 text-xs">
          先参考上面的 CLI 指南完成安装与登录，然后在组织设置里注册对应
          Provider，回来后点“重新检测全部 Provider”即可。
        </p>
        <Button
          variant="outline"
          size="sm"
          class="mt-3"
          onclick={() => window.open(`/orgs/${orgId}/settings`, '_blank')}
        >
          前往组织设置
        </Button>
      </div>
    {:else}
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
        {#each providers as provider (provider.id)}
          {@const available = isProviderAvailable(provider)}
          {@const isSelected = selectedId === provider.id}
          {@const status = providerStatus(provider)}
          {@const checkedAt = providerAvailabilityCheckedAtText(provider.availability_checked_at)}
          {@const guide = guideForProvider(provider)}
          {@const machineIds = provider.machine_id ? [provider.machine_id] : []}
          {@const iconPath = adapterIconPath(provider.adapter_type)}

          <div
            class={cn(
              'border-border bg-card rounded-xl border p-4 transition-all',
              isSelected ? 'border-primary bg-primary/5 ring-primary/20 ring-1' : '',
            )}
          >
            <div class="mb-3 flex items-start gap-3">
              <div class="bg-muted flex size-10 shrink-0 items-center justify-center rounded-xl">
                {#if iconPath}
                  <img src={iconPath} alt="" class="size-6" />
                {:else}
                  <Zap class="text-foreground size-5" />
                {/if}
              </div>

              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <p class="text-foreground truncate text-sm font-semibold">{provider.name}</p>
                  <span class={cn('text-xs', status.className)}>{status.text}</span>
                </div>
                <p class="text-muted-foreground mt-1 text-xs">
                  {adapterDisplayName(provider.adapter_type)} · {provider.model_name ||
                    '未设置模型'}
                </p>
              </div>

              {#if isSelected}
                <div
                  class="bg-primary text-primary-foreground flex size-5 items-center justify-center rounded-full"
                >
                  <CheckCircle2 class="size-3.5" />
                </div>
              {/if}
            </div>

            <div class="mb-3 space-y-1 text-xs">
              <div class="flex items-center justify-between gap-3">
                <span class="text-muted-foreground">机器</span>
                <span class="text-foreground text-right">{provider.machine_name || '—'}</span>
              </div>
              <div class="flex items-center justify-between gap-3">
                <span class="text-muted-foreground">CLI</span>
                <span class="text-foreground text-right">{provider.cli_command || '—'}</span>
              </div>
              {#if checkedAt}
                <div class="flex items-center justify-between gap-3">
                  <span class="text-muted-foreground">最近检测</span>
                  <span class="text-foreground text-right">{checkedAt}</span>
                </div>
              {/if}
            </div>

            <div class="border-border bg-muted/30 mb-3 rounded-lg border px-3 py-2 text-xs">
              <p class="text-foreground font-medium">
                {providerAvailabilityHeadline(
                  provider.availability_state,
                  provider.availability_reason,
                )}
              </p>
              <p class="text-muted-foreground mt-1">
                {providerAvailabilityDescription(
                  provider.availability_state,
                  provider.availability_reason,
                )}
              </p>
            </div>

            <div class="flex flex-wrap gap-2">
              {#if available}
                <Button
                  size="sm"
                  variant={isSelected ? 'default' : 'outline'}
                  disabled={selecting}
                  onclick={() => void handleSelectProvider(provider.id)}
                >
                  {#if selecting && isSelected}
                    <Loader2 class="mr-1.5 size-3.5 animate-spin" />
                    设置中...
                  {:else if isSelected}
                    已设为默认
                  {:else}
                    使用这个 Provider
                  {/if}
                </Button>
              {:else}
                {#if guide}
                  <Button size="sm" variant="outline" onclick={() => openGuide(guide.key)}>
                    查看 {guide.title} 指南
                  </Button>
                {/if}
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={isRefreshing(machineIds)}
                  onclick={() => void refreshProviders(machineIds)}
                >
                  {#if isRefreshing(machineIds)}
                    <Loader2 class="mr-1.5 size-3.5 animate-spin" />
                    检测中...
                  {:else}
                    <RefreshCcw class="mr-1.5 size-3.5" />
                    重新检测
                  {/if}
                </Button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<Sheet.Root bind:open={guideOpen}>
  <Sheet.Content side="right" class="w-full max-w-xl sm:max-w-xl">
    {#if activeGuide}
      {@const matchingProviders = guideProviders(activeGuide)}
      {@const primaryProvider = primaryGuideProvider(matchingProviders)}
      {@const reasonHints = reasonSpecificHints(primaryProvider)}
      <div class="space-y-5">
        <Sheet.Header>
          <Sheet.Title>{activeGuide.title} 配置指南</Sheet.Title>
          <Sheet.Description>
            先完成安装、登录和验证，再回到这里重新检测；只要对应 Provider
            变成可用，就可以直接在当前步骤设为默认。
          </Sheet.Description>
        </Sheet.Header>

        <div class="border-border bg-muted/30 rounded-xl border p-4">
          <div class="flex items-start justify-between gap-3">
            <div class="space-y-1">
              <p class="text-foreground text-sm font-semibold">官方指引</p>
              <p class="text-muted-foreground text-xs">
                建议优先按官方文档完成安装与认证，避免和本机 PATH / 凭据状态不一致。
              </p>
            </div>
            <a
              href={activeGuide.docsUrl}
              target="_blank"
              rel="noopener noreferrer"
              class="text-primary inline-flex items-center gap-1 text-xs font-medium"
            >
              打开文档
              <ExternalLink class="size-3.5" />
            </a>
          </div>

          <div class="mt-3 grid grid-cols-2 gap-2 text-xs">
            <div class="bg-background rounded-lg px-3 py-2">
              <p class="text-muted-foreground">推荐模型</p>
              <p class="text-foreground mt-1 font-medium">{activeGuide.recommendedModel}</p>
            </div>
            <div class="bg-background rounded-lg px-3 py-2">
              <p class="text-muted-foreground">已注册实例</p>
              <p class="text-foreground mt-1 font-medium">{matchingProviders.length}</p>
            </div>
          </div>
        </div>

        {#if primaryProvider}
          <div class="border-border bg-card rounded-xl border p-4">
            <p class="text-foreground text-sm font-semibold">当前 OpenASE 检测结果</p>
            <p class="text-foreground mt-3 text-sm">
              {providerAvailabilityHeadline(
                primaryProvider.availability_state,
                primaryProvider.availability_reason,
              )}
            </p>
            <p class="text-muted-foreground mt-1 text-xs">
              {providerAvailabilityDescription(
                primaryProvider.availability_state,
                primaryProvider.availability_reason,
              )}
            </p>
            {#if primaryProvider.availability_checked_at}
              <p class="text-muted-foreground mt-2 text-xs">
                最近检测：{providerAvailabilityCheckedAtText(
                  primaryProvider.availability_checked_at,
                ) || primaryProvider.availability_checked_at}
              </p>
            {/if}
          </div>
        {/if}

        {#if matchingProviders.length > 0}
          <div class="space-y-2">
            <p class="text-foreground text-sm font-semibold">当前已注册实例</p>
            {#each matchingProviders as provider (provider.id)}
              <div
                class="border-border bg-card flex items-center justify-between gap-3 rounded-xl border p-3"
              >
                <div class="min-w-0">
                  <p class="text-foreground truncate text-sm font-medium">{provider.name}</p>
                  <p class="text-muted-foreground text-xs">
                    {provider.machine_name || '—'} · {provider.model_name || '未设置模型'} · {providerStatus(
                      provider,
                    ).text}
                  </p>
                </div>

                {#if isProviderAvailable(provider)}
                  <Button
                    size="sm"
                    variant={selectedId === provider.id ? 'default' : 'outline'}
                    disabled={selecting}
                    onclick={() => void handleSelectProvider(provider.id)}
                  >
                    {selectedId === provider.id ? '已设为默认' : '使用这个 Provider'}
                  </Button>
                {/if}
              </div>
            {/each}
          </div>
        {/if}

        <div class="space-y-3">
          <p class="text-foreground text-sm font-semibold">1. 安装 CLI</p>
          {#each activeGuide.installCommands as item (item.command)}
            <div class="border-border bg-card rounded-xl border p-3">
              <div class="mb-2 flex items-center justify-between gap-3">
                <div class="flex items-center gap-2">
                  <TerminalSquare class="text-muted-foreground size-4" />
                  <p class="text-foreground text-sm font-medium">{item.label}</p>
                </div>
                <Button size="sm" variant="ghost" onclick={() => void copyCommand(item.command)}>
                  <Copy class="mr-1.5 size-3.5" />
                  复制
                </Button>
              </div>
              <pre class="bg-muted overflow-x-auto rounded-lg px-3 py-2 text-xs"><code
                  >{item.command}</code
                ></pre>
            </div>
          {/each}
        </div>

        <div class="space-y-3">
          <p class="text-foreground text-sm font-semibold">2. 登录 / 认证</p>
          {#each activeGuide.authCommands as item (item.command)}
            <div class="border-border bg-card rounded-xl border p-3">
              <div class="mb-2 flex items-center justify-between gap-3">
                <div class="flex items-center gap-2">
                  <LogIn class="text-muted-foreground size-4" />
                  <p class="text-foreground text-sm font-medium">{item.label}</p>
                </div>
                <Button size="sm" variant="ghost" onclick={() => void copyCommand(item.command)}>
                  <Copy class="mr-1.5 size-3.5" />
                  复制
                </Button>
              </div>
              <pre class="bg-muted overflow-x-auto rounded-lg px-3 py-2 text-xs"><code
                  >{item.command}</code
                ></pre>
            </div>
          {/each}
        </div>

        <div class="space-y-3">
          <p class="text-foreground text-sm font-semibold">3. 验证是否可用</p>
          {#each activeGuide.verifyCommands as item (item.command)}
            <div class="border-border bg-card rounded-xl border p-3">
              <div class="mb-2 flex items-center justify-between gap-3">
                <div class="flex items-center gap-2">
                  <SearchCheck class="text-muted-foreground size-4" />
                  <p class="text-foreground text-sm font-medium">{item.label}</p>
                </div>
                <Button size="sm" variant="ghost" onclick={() => void copyCommand(item.command)}>
                  <Copy class="mr-1.5 size-3.5" />
                  复制
                </Button>
              </div>
              <pre class="bg-muted overflow-x-auto rounded-lg px-3 py-2 text-xs"><code
                  >{item.command}</code
                ></pre>
            </div>
          {/each}
        </div>

        <div class="space-y-3">
          <p class="text-foreground text-sm font-semibold">常见修复提示</p>
          <div class="border-border bg-card rounded-xl border p-4">
            <div class="space-y-2 text-sm">
              {#if reasonHints.length > 0}
                {#each reasonHints as hint (hint)}
                  <div class="flex items-start gap-2">
                    <AlertTriangle class="mt-0.5 size-4 shrink-0 text-amber-600" />
                    <p class="text-foreground">{hint}</p>
                  </div>
                {/each}
              {/if}

              {#each activeGuide.commonFixHints as hint (hint)}
                <div class="flex items-start gap-2">
                  <Wrench class="text-muted-foreground mt-0.5 size-4 shrink-0" />
                  <p class="text-foreground">{hint}</p>
                </div>
              {/each}
            </div>
          </div>
        </div>

        <div class="flex flex-wrap gap-2 pt-2">
          <Button
            variant="outline"
            onclick={() => window.open(`/orgs/${orgId}/settings`, '_blank')}
          >
            打开组织设置
          </Button>
          <Button
            variant="outline"
            disabled={isRefreshing(uniqueMachineIds(matchingProviders))}
            onclick={() => void refreshProviders(uniqueMachineIds(matchingProviders))}
          >
            {#if isRefreshing(uniqueMachineIds(matchingProviders))}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              检测中...
            {:else}
              <RefreshCcw class="mr-1.5 size-3.5" />
              完成后重新检测
            {/if}
          </Button>
          <Button onclick={() => (guideOpen = false)}>返回 Provider 列表</Button>
        </div>
      </div>
    {/if}
  </Sheet.Content>
</Sheet.Root>
