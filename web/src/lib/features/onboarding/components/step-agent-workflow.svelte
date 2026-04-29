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
  import { cn } from '$lib/utils'
  import {
    Bot,
    GitBranch,
    Loader2,
    CheckCircle2,
    Workflow as WorkflowIcon,
    AlertTriangle,
    Code2,
    ClipboardList,
    Lightbulb,
  } from '@lucide/svelte'
  import type { AgentWorkflowState, BootstrapPresetKey, ProjectBootstrapPreset } from '../types'
  import { bootstrapPresets, isTerminalProjectStatus } from '../model'
  import { onboardingT } from '../i18n'

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
    onComplete: (agents: Agent[], workflows: Workflow[], presetKey: BootstrapPresetKey) => void
  } = $props()

  const presetIcons = {
    fullstack: Code2,
    pm: ClipboardList,
    researcher: Lightbulb,
  } as const

  let bootstrapping = $state(false)
  let agents = $state([...untrack(() => initialState.agents)])
  let workflows = $state([...untrack(() => initialState.workflows)])
  function resolveInitialPreset(): ProjectBootstrapPreset | null {
    return untrack(() => {
      if (!initialState.selectedPresetKey) {
        return null
      }

      return (
        bootstrapPresets.find((preset) => preset.key === initialState.selectedPresetKey) ?? null
      )
    })
  }

  let selectedPreset = $state<ProjectBootstrapPreset | null>(resolveInitialPreset())

  const hasAgentAndWorkflow = $derived(agents.length > 0 && workflows.length > 0)
  const isTerminal = $derived(isTerminalProjectStatus(projectStatus))

  function findStatusByName(name: string, statuses: AgentWorkflowState['statuses']) {
    const target = name.trim().toLowerCase()
    return statuses.find((status) => status.name.trim().toLowerCase() === target)
  }

  async function handleBootstrap() {
    if (!selectedPreset) return
    if (!providerId) {
      toastStore.error(onboardingT('onboarding.agentWorkflow.errors.selectProvider'))
      return
    }
    bootstrapping = true
    const preset = selectedPreset
    try {
      const statusPayload = await listStatuses(projectId)
      const statuses = statusPayload.statuses

      const pickupStatus = findStatusByName(preset.pickupStatusName, statuses)
      const finishStatus = findStatusByName(preset.finishStatusName, statuses)

      if (!pickupStatus || !finishStatus) {
        toastStore.error(
          onboardingT('onboarding.agentWorkflow.errors.statusesMissing', {
            pickupStatusName: preset.pickupStatusName,
            finishStatusName: preset.finishStatusName,
          }),
        )
        return
      }

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

      const agentPayload = await createAgent(projectId, {
        provider_id: providerId,
        name: preset.agentNameSuggestion,
      })

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

      const [refreshedAgents, refreshedWorkflows] = await Promise.all([
        listAgents(projectId),
        listWorkflows(projectId),
      ])
      agents = refreshedAgents.agents
      workflows = refreshedWorkflows.workflows

      toastStore.success(
        onboardingT('onboarding.agentWorkflow.toasts.created', {
          agentName: preset.agentNameSuggestion,
        }),
      )
      onComplete(refreshedAgents.agents, refreshedWorkflows.workflows, preset.key)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : onboardingT('onboarding.agentWorkflow.toasts.creationFailed'),
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
        <p class="text-foreground text-sm font-medium">
          {onboardingT('onboarding.agentWorkflow.status.currentStatus', { projectStatus })}
        </p>
        <p class="text-muted-foreground mt-1 text-xs">
          {onboardingT('onboarding.agentWorkflow.status.terminalNotice')}
        </p>
      </div>
    </div>
  {:else if hasAgentAndWorkflow}
    <div class="space-y-2">
      {#each agents as agent (agent.id)}
        <div
          class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 dark:border-emerald-900/50 dark:bg-emerald-950/30"
        >
          <CheckCircle2 class="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
          <Bot class="text-muted-foreground size-4 shrink-0" />
          <div class="min-w-0 flex-1">
            <p class="text-foreground text-sm font-medium">{agent.name}</p>
            <p class="text-muted-foreground text-xs">
              {onboardingT('onboarding.agentWorkflow.labels.agent')}
            </p>
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
            <p class="text-muted-foreground text-xs">
              {onboardingT('onboarding.agentWorkflow.labels.workflowWithType', {
                workflowType: workflow.type,
              })}
            </p>
          </div>
        </div>
      {/each}
    </div>
  {:else if selectedPreset}
    <!-- Preview + create -->
    <div class="border-border rounded-lg border p-4">
      <div class="mb-3 flex items-center justify-between">
        <p class="text-foreground text-sm font-medium">
          {onboardingT('onboarding.agentWorkflow.preview.heading')}
        </p>
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground text-xs underline transition-colors"
          onclick={() => (selectedPreset = null)}
        >
          {onboardingT('onboarding.agentWorkflow.preview.change')}
        </button>
      </div>
      <div class="space-y-2">
        <div class="bg-muted/50 flex items-center gap-3 rounded-md px-3 py-2">
          <Bot class="text-primary size-4 shrink-0" />
          <div>
            <p class="text-foreground text-sm">
              {onboardingT('onboarding.agentWorkflow.preview.agentLabel', {
                agentName: selectedPreset.agentNameSuggestion,
              })}
            </p>
            <p class="text-muted-foreground text-xs">
              {onboardingT('onboarding.agentWorkflow.preview.boundToProvider')}
            </p>
          </div>
        </div>
        <div class="bg-muted/50 flex items-center gap-3 rounded-md px-3 py-2">
          <WorkflowIcon class="text-primary size-4 shrink-0" />
          <div>
            <p class="text-foreground text-sm">
              {onboardingT('onboarding.agentWorkflow.preview.workflowLabel', {
                roleName: selectedPreset.roleName,
              })}
            </p>
            <p class="text-muted-foreground text-xs">
              {onboardingT('onboarding.agentWorkflow.preview.roleLabel', {
                roleName: selectedPreset.roleName,
              })}
            </p>
          </div>
        </div>
        <div class="bg-muted/50 flex items-center gap-3 rounded-md px-3 py-2">
          <GitBranch class="text-primary size-4 shrink-0" />
          <div>
            <p class="text-foreground text-sm">
              {onboardingT('onboarding.agentWorkflow.preview.statusFlow')}
            </p>
            <p class="text-muted-foreground text-xs">
              {onboardingT('onboarding.agentWorkflow.preview.pickupFinish', {
                pickupStatusName: selectedPreset.pickupStatusName,
                finishStatusName: selectedPreset.finishStatusName,
              })}
            </p>
          </div>
        </div>
      </div>

      <Button class="mt-4 w-full" onclick={handleBootstrap} disabled={bootstrapping || !providerId}>
        {#if bootstrapping}
          <Loader2 class="mr-1.5 size-3.5 animate-spin" />
          {onboardingT('onboarding.agentWorkflow.actions.creating')}
        {:else}
          {onboardingT('onboarding.agentWorkflow.actions.create')}
        {/if}
      </Button>
    </div>
  {:else}
    <!-- Preset picker -->
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
      {#each bootstrapPresets as preset (preset.key)}
        {@const Icon = presetIcons[preset.key]}
        <button
          type="button"
          class={cn(
            'border-border bg-card hover:border-primary/50 hover:bg-muted/30 flex flex-col rounded-lg border p-4 text-left transition-colors',
          )}
          onclick={() => (selectedPreset = preset)}
        >
          <div class="bg-primary/10 mb-3 flex size-9 items-center justify-center rounded-lg">
            <Icon class="text-primary size-4" />
          </div>
          <p class="text-foreground text-sm font-semibold">{preset.title}</p>
          <p class="text-muted-foreground mt-1 text-xs leading-5">{preset.subtitle}</p>
          <p class="text-muted-foreground mt-3 text-xs">
            {onboardingT('onboarding.agentWorkflow.presets.roleLabel', {
              roleName: preset.roleName,
            })}
          </p>
        </button>
      {/each}
    </div>
  {/if}
</div>
