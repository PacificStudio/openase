<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import { updateProject } from '$lib/api/openase'
  import type { AgentProvider } from '$lib/api/contracts'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { CheckCircle2, CircleDashed, AlertTriangle, Loader2, Zap } from '@lucide/svelte'
  import type { ProviderState } from '../types'

  let {
    projectId,
    orgId,
    initialState,
    onComplete,
  }: {
    projectId: string
    orgId: string
    initialState: ProviderState
    onComplete: (providerId: string) => void
  } = $props()

  let selecting = $state(false)
  let selectedId = $state(untrack(() => initialState.selectedProviderId))

  const providers = $derived(initialState.providers)

  const adapterIcons: Record<string, string> = {
    claude_code: 'Claude Code',
    codex: 'OpenAI Codex',
    gemini_cli: 'Gemini CLI',
    custom: 'Custom',
  }

  const availabilityLabel: Record<string, { text: string; className: string }> = {
    available: { text: '可用', className: 'text-emerald-600 dark:text-emerald-400' },
    ready: { text: '就绪', className: 'text-emerald-600 dark:text-emerald-400' },
    unavailable: { text: '不可用', className: 'text-destructive' },
    stale: { text: '状态过期', className: 'text-amber-600 dark:text-amber-400' },
    unknown: { text: '未知', className: 'text-muted-foreground' },
  }

  function isProviderAvailable(provider: AgentProvider): boolean {
    return provider.availability_state === 'available' || provider.availability_state === 'ready'
  }

  async function handleSelectProvider(providerId: string) {
    if (selecting) return
    selectedId = providerId
    selecting = true
    try {
      const payload = await updateProject(projectId, {
        default_agent_provider_id: providerId,
      })
      appStore.currentProject = payload.project
      toastStore.success('已设为默认 Provider。')
      onComplete(providerId)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : '设置默认 Provider 失败。',
      )
      selectedId = ''
    } finally {
      selecting = false
    }
  }
</script>

<div class="space-y-4">
  {#if providers.length === 0}
    <div class="border-border rounded-lg border border-dashed p-6 text-center">
      <CircleDashed class="text-muted-foreground mx-auto mb-2 size-8" />
      <p class="text-muted-foreground text-sm">尚未配置任何 Provider。</p>
      <p class="text-muted-foreground mt-1 text-xs">
        请先在组织设置中添加 AI Provider（Claude Code / Codex / Gemini CLI）。
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
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {#each providers as provider (provider.id)}
        {@const available = isProviderAvailable(provider)}
        {@const isSelected = selectedId === provider.id}
        {@const availability =
          availabilityLabel[provider.availability_state] ?? availabilityLabel.unknown}
        <div
          class={cn(
            'relative rounded-lg border p-4 transition-all',
            isSelected
              ? 'border-primary bg-primary/5 ring-primary/20 ring-1'
              : available
                ? 'border-border hover:border-primary/40 cursor-pointer'
                : 'border-border opacity-60',
          )}
        >
          {#if isSelected}
            <div
              class="bg-primary text-primary-foreground absolute -top-2 -right-2 flex size-5 items-center justify-center rounded-full"
            >
              <CheckCircle2 class="size-3.5" />
            </div>
          {/if}

          <div class="mb-3 flex items-center gap-2">
            <div class="bg-muted flex size-8 items-center justify-center rounded-lg">
              <Zap class="text-foreground size-4" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-foreground truncate text-sm font-medium">{provider.name}</p>
              <p class="text-muted-foreground text-[10px]">
                {adapterIcons[provider.adapter_type] ?? provider.adapter_type}
              </p>
            </div>
          </div>

          <div class="mb-3 space-y-1 text-xs">
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">状态</span>
              <span class={availability.className}>
                {#if available}
                  <CheckCircle2 class="mr-0.5 inline size-3" />
                {:else}
                  <AlertTriangle class="mr-0.5 inline size-3" />
                {/if}
                {availability.text}
              </span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">模型</span>
              <span class="text-foreground">{provider.model_name || '—'}</span>
            </div>
            {#if provider.machine_name}
              <div class="flex items-center justify-between">
                <span class="text-muted-foreground">机器</span>
                <span class="text-foreground">{provider.machine_name}</span>
              </div>
            {/if}
          </div>

          {#if available}
            <Button
              size="sm"
              class="w-full"
              variant={isSelected ? 'default' : 'outline'}
              disabled={selecting}
              onclick={() => void handleSelectProvider(provider.id)}
            >
              {#if selecting && selectedId === provider.id}
                <Loader2 class="mr-1.5 size-3.5 animate-spin" />
                设置中...
              {:else if isSelected}
                已选中
              {:else}
                使用这个 Provider
              {/if}
            </Button>
          {:else}
            <Button variant="ghost" size="sm" class="w-full" disabled>不可用</Button>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
