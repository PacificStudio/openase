<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import {
    createAgent,
    createWorkflow,
    listBuiltinRoles,
    listStatuses,
    listAgents,
    listWorkflows,
  } from '$lib/api/openase'
  import type { Agent, Workflow } from '$lib/api/contracts'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import {
    Bot,
    GitBranch,
    Loader2,
    CheckCircle2,
    Workflow as WorkflowIcon,
    AlertTriangle,
  } from '@lucide/svelte'
  import type { AgentWorkflowState } from '../types'
  import { getBootstrapPreset, isTerminalProjectStatus } from '../model'

  let {
    projectId,
    providerId,
    projectStatus,
    initialState,
    onComplete,
  }: {
    projectId: string
    providerId: string
    projectStatus: string
    initialState: AgentWorkflowState
    onComplete: (agents: Agent[], workflows: Workflow[]) => void
  } = $props()

  let bootstrapping = $state(false)
  let agents = $state([...untrack(() => initialState.agents)])
  let workflows = $state([...untrack(() => initialState.workflows)])

  const hasAgentAndWorkflow = $derived(agents.length > 0 && workflows.length > 0)
  const preset = $derived(getBootstrapPreset(projectStatus))
  const isTerminal = $derived(isTerminalProjectStatus(projectStatus))

  function findStatusByName(name: string, statuses: AgentWorkflowState['statuses']) {
    const target = name.trim().toLowerCase()
    return statuses.find((status) => status.name.trim().toLowerCase() === target)
  }

  async function handleBootstrap() {
    if (!providerId) {
      toastStore.error('请先选择一个 Provider。')
      return
    }
    bootstrapping = true
    try {
      // Find the configured canonical statuses for the onboarding bootstrap.
      const statusPayload = await listStatuses(projectId)
      const statuses = statusPayload.statuses

      const pickupStatus = findStatusByName(preset.pickupStatusName, statuses)
      const finishStatus = findStatusByName(preset.finishStatusName, statuses)

      if (!pickupStatus || !finishStatus) {
        toastStore.error(
          `未找到状态「${preset.pickupStatusName}」或「${preset.finishStatusName}」，请先在设置中配置状态。`,
        )
        return
      }

      // Get builtin role harness content
      let harnessContent = ''
      let roleSkillNames: string[] = []
      let rolePlatformAccessAllowed: string[] = []
      try {
        const rolesPayload = await listBuiltinRoles()
        const role = rolesPayload.roles.find((r) => r.slug === preset.roleSlug)
        if (role) {
          harnessContent = role.content
          const roleMeta = role as typeof role & {
            skill_names?: string[]
            platform_access_allowed?: string[]
          }
          roleSkillNames = roleMeta.skill_names ?? []
          rolePlatformAccessAllowed = roleMeta.platform_access_allowed ?? []
        }
      } catch {
        // Use empty harness if roles unavailable
      }

      // 1. Create Agent
      const agentPayload = await createAgent(projectId, {
        provider_id: providerId,
        name: preset.agentNameSuggestion,
      })

      // 2. Create Workflow bound to the agent
      await createWorkflow(projectId, {
        agent_id: agentPayload.agent.id,
        name: `${preset.roleName} Workflow`,
        type: preset.workflowType,
        role_slug: preset.roleSlug,
        role_name: preset.roleName,
        role_description: preset.roleName,
        skill_names: roleSkillNames,
        platform_access_allowed: rolePlatformAccessAllowed,
        pickup_status_ids: [pickupStatus.id],
        finish_status_ids: [finishStatus.id],
        harness_content: harnessContent || undefined,
        is_active: true,
        max_concurrent: 0,
        max_retry_attempts: 1,
        timeout_minutes: 30,
      })

      // Refresh data
      const [refreshedAgents, refreshedWorkflows] = await Promise.all([
        listAgents(projectId),
        listWorkflows(projectId),
      ])
      agents = refreshedAgents.agents
      workflows = refreshedWorkflows.workflows

      toastStore.success(`已创建 Agent「${preset.agentNameSuggestion}」和 Workflow。`)
      onComplete(refreshedAgents.agents, refreshedWorkflows.workflows)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : '创建 Agent 与 Workflow 失败。',
      )
    } finally {
      bootstrapping = false
    }
  }
</script>

<div class="space-y-4">
  {#if isTerminal}
    <div class="border-border flex items-start gap-3 rounded-lg border border-dashed p-4">
      <AlertTriangle class="mt-0.5 size-5 shrink-0 text-amber-500" />
      <div>
        <p class="text-foreground text-sm font-medium">当前项目状态为「{projectStatus}」</p>
        <p class="text-muted-foreground mt-1 text-xs">
          处于 Completed / Canceled / Archived 状态的项目不会自动创建执行角色。请先变更项目状态。
        </p>
      </div>
    </div>
  {:else if hasAgentAndWorkflow}
    <!-- Already set up -->
    <div class="space-y-2">
      {#each agents as agent (agent.id)}
        <div
          class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 dark:border-emerald-900/50 dark:bg-emerald-950/30"
        >
          <CheckCircle2 class="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
          <Bot class="text-muted-foreground size-4 shrink-0" />
          <div class="min-w-0 flex-1">
            <p class="text-foreground text-sm font-medium">{agent.name}</p>
            <p class="text-muted-foreground text-xs">Agent</p>
          </div>
        </div>
      {/each}
      {#each workflows as workflow (workflow.id)}
        <div
          class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 dark:border-emerald-900/50 dark:bg-emerald-950/30"
        >
          <CheckCircle2 class="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
          <WorkflowIcon class="text-muted-foreground size-4 shrink-0" />
          <div class="min-w-0 flex-1">
            <p class="text-foreground text-sm font-medium">{workflow.name}</p>
            <p class="text-muted-foreground text-xs">Workflow · {workflow.type}</p>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <!-- Bootstrap preview -->
    <div class="border-border rounded-lg border p-4">
      <p class="text-foreground mb-3 text-sm font-medium">即将创建以下配置：</p>
      <div class="space-y-2">
        <div class="bg-muted/50 flex items-center gap-3 rounded-md px-3 py-2">
          <Bot class="text-primary size-4 shrink-0" />
          <div>
            <p class="text-foreground text-sm">Agent：{preset.agentNameSuggestion}</p>
            <p class="text-muted-foreground text-xs">绑定到已选择的默认 Provider</p>
          </div>
        </div>
        <div class="bg-muted/50 flex items-center gap-3 rounded-md px-3 py-2">
          <WorkflowIcon class="text-primary size-4 shrink-0" />
          <div>
            <p class="text-foreground text-sm">Workflow：{preset.roleName} Workflow</p>
            <p class="text-muted-foreground text-xs">
              角色：{preset.roleName} · 类型：{preset.workflowType}
            </p>
          </div>
        </div>
        <div class="bg-muted/50 flex items-center gap-3 rounded-md px-3 py-2">
          <GitBranch class="text-primary size-4 shrink-0" />
          <div>
            <p class="text-foreground text-sm">状态流转</p>
            <p class="text-muted-foreground text-xs">
              Pickup：{preset.pickupStatusName} → Finish：{preset.finishStatusName}
            </p>
          </div>
        </div>
      </div>

      <Button class="mt-4 w-full" onclick={handleBootstrap} disabled={bootstrapping || !providerId}>
        {#if bootstrapping}
          <Loader2 class="mr-1.5 size-3.5 animate-spin" />
          正在创建...
        {:else}
          一键创建 Agent 与 Workflow
        {/if}
      </Button>
    </div>
  {/if}
</div>
