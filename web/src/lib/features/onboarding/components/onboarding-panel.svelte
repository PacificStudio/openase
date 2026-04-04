<script lang="ts">
  import { cn } from '$lib/utils'
  import { ApiError } from '$lib/api/client'
  import { appStore } from '$lib/stores/app.svelte'
  import {
    Github,
    FolderGit2,
    Zap,
    Bot,
    Ticket,
    Sparkles,
    CheckCircle2,
    Lock,
    Loader2,
    CircleDot,
  } from '@lucide/svelte'
  import type { OnboardingData, OnboardingStep, OnboardingStepId } from '../types'
  import { buildOnboardingSteps, currentActiveStep } from '../model'
  import { loadOnboardingData } from '../data'
  import OnboardingHelpLinks from './onboarding-help-links.svelte'
  import OnboardingStepDetails from './onboarding-step-details.svelte'

  let {
    projectId,
    orgId,
    projectName,
    projectStatus,
    onOnboardingComplete,
  }: {
    projectId: string
    orgId: string
    projectName: string
    projectStatus: string
    onOnboardingComplete: () => void
  } = $props()

  function openProjectAssistant(prompt: string) {
    appStore.requestProjectAssistant(prompt)
  }

  let loading = $state(true)
  let error = $state('')
  let data = $state<OnboardingData | null>(null)
  let expandedStep = $state<OnboardingStepId | null>(null)

  const steps = $derived(data ? buildOnboardingSteps(data) : [])
  const activeStepId = $derived(data ? currentActiveStep(data) : null)

  // Auto-expand active step
  $effect(() => {
    if (activeStepId && expandedStep !== activeStepId) {
      expandedStep = activeStepId
    }
  })

  const stepIcons: Record<OnboardingStepId, typeof Github> = {
    github_token: Github,
    repo: FolderGit2,
    provider: Zap,
    agent_workflow: Bot,
    first_ticket: Ticket,
    ai_discovery: Sparkles,
  }

  const selectedProviderId = $derived(data ? data.provider.selectedProviderId : '')

  $effect(() => {
    void projectId
    void orgId
    let cancelled = false

    const load = async () => {
      loading = true
      error = ''
      try {
        const payload = await loadOnboardingData(projectId, orgId)
        if (cancelled) return
        payload.projectStatus = projectStatus
        data = payload
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : '加载导览数据失败。'
      } finally {
        if (!cancelled) loading = false
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  })

  function handleStepClick(step: OnboardingStep) {
    if (step.status === 'locked') return
    expandedStep = expandedStep === step.id ? null : step.id
  }

  function refreshData() {
    const load = async () => {
      try {
        const payload = await loadOnboardingData(projectId, orgId)
        payload.projectStatus = projectStatus
        data = payload
      } catch {
        // silent refresh failure
      }
    }
    void load()
  }
</script>

<div class="space-y-6">
  <!-- Welcome bar -->
  <div class="bg-primary/5 border-primary/20 rounded-xl border p-5">
    <div class="flex items-start gap-4">
      <div class="bg-primary/10 flex size-12 shrink-0 items-center justify-center rounded-xl">
        <Sparkles class="text-primary size-6" />
      </div>
      <div>
        <h2 class="text-foreground text-lg font-semibold">
          欢迎来到 {projectName}
        </h2>
        <p class="text-muted-foreground mt-1 text-sm">
          项目已创建，接下来我们会带你把它配置到可运行状态。完成以下步骤后，Agent 就可以开始工作了。
        </p>
      </div>
    </div>
  </div>

  {#if loading}
    <div
      class="border-border bg-card flex items-center justify-center rounded-xl border px-4 py-12"
    >
      <Loader2 class="text-muted-foreground size-5 animate-spin" />
      <span class="text-muted-foreground ml-2 text-sm">加载配置状态...</span>
    </div>
  {:else if error}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive rounded-xl border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {:else if data}
    <!-- Onboarding checklist -->
    <div class="border-border bg-card rounded-xl border">
      <div class="border-border border-b px-5 py-3">
        <h3 class="text-foreground text-sm font-medium">配置进度</h3>
        <div class="bg-muted mt-2 h-1.5 overflow-hidden rounded-full">
          <div
            class="bg-primary h-full rounded-full transition-all duration-500"
            style="width: {(steps.filter((s) => s.status === 'completed').length / steps.length) *
              100}%"
          ></div>
        </div>
        <p class="text-muted-foreground mt-1.5 text-xs">
          {steps.filter((s) => s.status === 'completed').length} / {steps.length} 步已完成
        </p>
      </div>

      <div class="divide-border divide-y">
        {#each steps as step, idx (step.id)}
          {@const StepIcon = stepIcons[step.id]}
          {@const isExpanded = expandedStep === step.id}
          {@const isActive = step.status === 'active'}
          {@const isCompleted = step.status === 'completed'}
          {@const isLocked = step.status === 'locked'}

          <div class={cn('transition-colors', isExpanded && isActive ? 'bg-muted/30' : '')}>
            <!-- Step header -->
            <button
              type="button"
              class={cn(
                'flex w-full items-center gap-3 px-5 py-3.5 text-left transition-colors',
                isLocked ? 'cursor-not-allowed opacity-50' : 'hover:bg-muted/50',
              )}
              disabled={isLocked}
              onclick={() => handleStepClick(step)}
            >
              <!-- Step indicator -->
              <div class="flex size-7 shrink-0 items-center justify-center">
                {#if isCompleted}
                  <CheckCircle2 class="size-5 text-emerald-600 dark:text-emerald-400" />
                {:else if isActive}
                  <CircleDot class="text-primary size-5" />
                {:else}
                  <Lock class="text-muted-foreground size-4" />
                {/if}
              </div>

              <!-- Step icon & label -->
              <div class="bg-muted flex size-8 shrink-0 items-center justify-center rounded-lg">
                <StepIcon
                  class={cn('size-4', isActive ? 'text-primary' : 'text-muted-foreground')}
                />
              </div>

              <div class="min-w-0 flex-1">
                <p
                  class={cn(
                    'text-sm font-medium',
                    isCompleted ? 'text-muted-foreground' : 'text-foreground',
                  )}
                >
                  {step.label}
                </p>
                <p class="text-muted-foreground text-xs">{step.description}</p>
              </div>

              <!-- Step number -->
              <span class="text-muted-foreground shrink-0 text-xs font-medium">
                {idx + 1}/{steps.length}
              </span>
            </button>

            <!-- Expanded content -->
            {#if isExpanded && !isLocked}
              <div class="border-border border-t px-5 pt-4 pb-5 pl-[4.5rem]">
                <OnboardingStepDetails
                  {step}
                  {data}
                  {projectId}
                  {orgId}
                  {projectStatus}
                  {selectedProviderId}
                  onOpenProjectAI={openProjectAssistant}
                  onRefreshData={refreshData}
                  onDataChange={(nextData) => {
                    data = nextData
                  }}
                  {onOnboardingComplete}
                />
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>
    <OnboardingHelpLinks onOpenProjectAI={openProjectAssistant} />
  {/if}
</div>
