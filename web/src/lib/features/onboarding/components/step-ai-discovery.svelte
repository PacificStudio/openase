<script lang="ts">
  import { goto } from '$app/navigation'
  import { projectPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import { Bot, Sparkles, CheckCircle2, ArrowRight } from '@lucide/svelte'

  let {
    orgId,
    projectId,
    hasWorkflow,
    onOpenProjectAI,
    onComplete,
  }: {
    orgId: string
    projectId: string
    hasWorkflow: boolean
    onOpenProjectAI: (prompt: string) => void
    onComplete: () => void
  } = $props()

  let projectAIOpened = $state(false)
  let harnessAIOpened = $state(false)

  const allDone = $derived(projectAIOpened && harnessAIOpened)

  $effect(() => {
    if (allDone) {
      onComplete()
    }
  })

  function handleOpenProjectAI(prompt: string) {
    projectAIOpened = true
    onOpenProjectAI(prompt)
  }

  function handleOpenHarnessAI() {
    harnessAIOpened = true
    void goto(projectPath(orgId, projectId, 'workflows'))
  }
</script>

<div class="space-y-4">
  <!-- Project AI -->
  <div class="border-border rounded-lg border p-4">
    <div class="flex items-center gap-3">
      <div class="bg-primary/10 flex size-10 items-center justify-center rounded-lg">
        <Bot class="text-primary size-5" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <p class="text-foreground text-sm font-medium">Project AI</p>
          {#if projectAIOpened}
            <CheckCircle2 class="size-3.5 text-emerald-600 dark:text-emerald-400" />
          {/if}
        </div>
        <p class="text-muted-foreground text-xs">让 AI 帮你拆解需求、规划后续工单</p>
      </div>
    </div>
    <div class="mt-3 flex flex-wrap gap-2">
      <Button
        variant="outline"
        size="sm"
        class="text-xs"
        onclick={() => handleOpenProjectAI('基于当前项目和已有 Ticket，再帮我拆 3 个后续工单')}
      >
        <Sparkles class="mr-1 size-3" />
        帮我拆 3 个后续工单
      </Button>
      <Button
        variant="outline"
        size="sm"
        class="text-xs"
        onclick={() => handleOpenProjectAI('我下一步应该先做什么？')}
      >
        <Sparkles class="mr-1 size-3" />
        下一步做什么
      </Button>
    </div>
  </div>

  <!-- Harness AI -->
  <div class="border-border rounded-lg border p-4">
    <div class="flex items-center gap-3">
      <div class="bg-primary/10 flex size-10 items-center justify-center rounded-lg">
        <Sparkles class="text-primary size-5" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <p class="text-foreground text-sm font-medium">Harness AI</p>
          {#if harnessAIOpened}
            <CheckCircle2 class="size-3.5 text-emerald-600 dark:text-emerald-400" />
          {/if}
        </div>
        <p class="text-muted-foreground text-xs">用 AI 调整 Workflow 的工作规范和角色设定</p>
      </div>
    </div>
    {#if hasWorkflow}
      <Button variant="outline" size="sm" class="mt-3 text-xs" onclick={handleOpenHarnessAI}>
        <ArrowRight class="mr-1 size-3" />
        前往 Workflow 编辑器
      </Button>
    {:else}
      <p class="text-muted-foreground mt-2 text-xs">请先完成 Agent 与 Workflow 创建步骤。</p>
    {/if}
  </div>

  {#if allDone}
    <div
      class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 dark:border-emerald-900/50 dark:bg-emerald-950/30"
    >
      <CheckCircle2 class="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
      <p class="text-foreground text-sm font-medium">导览已完成！</p>
    </div>
  {/if}
</div>
