<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import {
    providerAvailabilityCheckedAtText,
    providerAvailabilityDescription,
    providerAvailabilityHeadline,
  } from '$lib/features/providers'
  import { Button } from '$ui/button'
  import * as Sheet from '$ui/sheet'
  import {
    AlertTriangle,
    Copy,
    ExternalLink,
    Loader2,
    LogIn,
    RefreshCcw,
    SearchCheck,
    TerminalSquare,
    Wrench,
  } from '@lucide/svelte'
  import {
    type ProviderGuide,
    isProviderAvailable,
    providerStatus,
    reasonSpecificHints,
    uniqueMachineIds,
  } from '../provider-guides'

  let {
    open = $bindable(false),
    orgId,
    activeGuide,
    matchingProviders,
    selectedId,
    selecting,
    isRefreshing,
    onClose,
    onCopyCommand,
    onRefresh,
    onSelectProvider,
  }: {
    open?: boolean
    orgId: string
    activeGuide: ProviderGuide | null
    matchingProviders: AgentProvider[]
    selectedId: string
    selecting: boolean
    isRefreshing: (machineIds: string[]) => boolean
    onClose: () => void
    onCopyCommand: (command: string) => Promise<void>
    onRefresh: (machineIds: string[]) => Promise<void>
    onSelectProvider: (providerId: string) => Promise<void>
  } = $props()

  const primaryProvider = $derived(
    matchingProviders.find((provider) => provider.id === selectedId) ??
      matchingProviders.find((provider) => isProviderAvailable(provider)) ??
      matchingProviders[0] ??
      null,
  )
  const reasonHints = $derived(reasonSpecificHints(primaryProvider))
  const refreshMachineIds = $derived(uniqueMachineIds(matchingProviders))
</script>

<Sheet.Root bind:open>
  <Sheet.Content side="right" class="w-full max-w-xl sm:max-w-xl">
    {#if activeGuide}
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
                    onclick={() => void onSelectProvider(provider.id)}
                  >
                    {selectedId === provider.id ? '已设为默认' : '使用这个 Provider'}
                  </Button>
                {/if}
              </div>
            {/each}
          </div>
        {/if}

        {#each [{ title: '1. 安装 CLI', icon: TerminalSquare, items: activeGuide.installCommands }, { title: '2. 登录 / 认证', icon: LogIn, items: activeGuide.authCommands }, { title: '3. 验证是否可用', icon: SearchCheck, items: activeGuide.verifyCommands }] as section (section.title)}
          <div class="space-y-3">
            <p class="text-foreground text-sm font-semibold">{section.title}</p>
            {#each section.items as item (item.command)}
              {@const Icon = section.icon}
              <div class="border-border bg-card rounded-xl border p-3">
                <div class="mb-2 flex items-center justify-between gap-3">
                  <div class="flex items-center gap-2">
                    <Icon class="text-muted-foreground size-4" />
                    <p class="text-foreground text-sm font-medium">{item.label}</p>
                  </div>
                  <Button
                    size="sm"
                    variant="ghost"
                    onclick={() => void onCopyCommand(item.command)}
                  >
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
        {/each}

        <div class="space-y-3">
          <p class="text-foreground text-sm font-semibold">常见修复提示</p>
          <div class="border-border bg-card rounded-xl border p-4">
            <div class="space-y-2 text-sm">
              {#each reasonHints as hint (hint)}
                <div class="flex items-start gap-2">
                  <AlertTriangle class="mt-0.5 size-4 shrink-0 text-amber-600" />
                  <p class="text-foreground">{hint}</p>
                </div>
              {/each}

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
            disabled={isRefreshing(refreshMachineIds)}
            onclick={() => void onRefresh(refreshMachineIds)}
          >
            {#if isRefreshing(refreshMachineIds)}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              检测中...
            {:else}
              <RefreshCcw class="mr-1.5 size-3.5" />
              完成后重新检测
            {/if}
          </Button>
          <Button onclick={onClose}>返回 Provider 列表</Button>
        </div>
      </div>
    {/if}
  </Sheet.Content>
</Sheet.Root>
