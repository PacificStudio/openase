<script lang="ts">
  import { goto } from '$app/navigation'
  import { projectPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import { Bot, Sparkles, ArrowRight } from '@lucide/svelte'

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

  let completed = $state(false)

  function finishOnboarding() {
    if (completed) return
    completed = true
    onComplete()
  }

  function handleOpenProjectAI(prompt: string) {
    onOpenProjectAI(prompt)
    finishOnboarding()
  }

  function handleOpenHarnessAI() {
    void goto(projectPath(orgId, projectId, 'workflows'))
    finishOnboarding()
  }
</script>

<div class="space-y-4">
  <div class="bg-muted/50 rounded-lg p-3">
    <p class="text-foreground text-sm font-medium">最后一步里，点击任意一个按钮都可以结束导览。</p>
    <p class="text-muted-foreground mt-1 text-xs">
      你可以直接体验 Project AI、前往 Workflow 编辑器，或者点“我知道了”先结束导览。
    </p>
  </div>

  <!-- Project AI -->
  <div class="border-border rounded-lg border p-4">
    <div class="flex items-center gap-3">
      <div class="bg-primary/10 flex size-10 items-center justify-center rounded-lg">
        <Bot class="text-primary size-5" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <p class="text-foreground text-sm font-medium">Project AI</p>
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

  <div class="flex justify-end">
    <Button variant="ghost" size="sm" class="text-xs" onclick={finishOnboarding}>我知道了</Button>
  </div>
</div>
