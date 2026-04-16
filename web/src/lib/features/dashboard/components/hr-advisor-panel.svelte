<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { MessageSquarePlus, WandSparkles } from '@lucide/svelte'
  import type { HRAdvisorSnapshot } from '../types'
  import HRAdvisorSummaryGrid from './hr-advisor-summary-grid.svelte'

  let {
    projectId,
    advisor,
    class: className = '',
  }: {
    projectId: string
    advisor: HRAdvisorSnapshot
    class?: string
  } = $props()

  const focusOwner = 'hr-advisor'

  function requestAdvice() {
    appStore.clearProjectAssistantFocus(focusOwner)
    appStore.requestProjectAssistant(
      i18nStore.t('dashboard.hrAdvisor.prompts.suggest', {
        projectId,
      }),
    )
  }

  function requestCreation() {
    appStore.clearProjectAssistantFocus(focusOwner)
    appStore.requestProjectAssistant(
      i18nStore.t('dashboard.hrAdvisor.prompts.create', {
        projectId,
      }),
    )
  }
</script>

<div class={cn('border-border bg-card rounded-xl border', className)}>
  <div class="flex items-center justify-between px-4 py-3">
    <div class="flex items-center gap-2">
      <h3 class="text-foreground text-sm font-medium">
        {i18nStore.t('dashboard.hrAdvisor.panel.heading')}
      </h3>
      <Badge variant="outline" class="text-[10px]">
        {i18nStore.t('dashboard.hrAdvisor.summary.labels.workflows', {
          count: advisor.summary.workflow_count,
        })}
      </Badge>
    </div>
    <span class="text-muted-foreground text-xs">
      {i18nStore.t('dashboard.hrAdvisor.messages.aiMode')}
    </span>
  </div>

  <div class="space-y-4 px-4 pb-4">
    <p class="text-muted-foreground text-xs leading-5">
      {i18nStore.t('dashboard.hrAdvisor.messages.description')}
    </p>

    <HRAdvisorSummaryGrid summary={advisor.summary} />

    <div class="grid gap-2 sm:grid-cols-2">
      <Button variant="outline" class="justify-start" onclick={requestAdvice}>
        <WandSparkles class="mr-2 size-4" />
        {i18nStore.t('dashboard.hrAdvisor.actions.askProjectAI')}
      </Button>
      <Button class="justify-start" onclick={requestCreation}>
        <MessageSquarePlus class="mr-2 size-4" />
        {i18nStore.t('dashboard.hrAdvisor.actions.askProjectAICreate')}
      </Button>
    </div>

    <div class="bg-muted/40 text-muted-foreground rounded-lg border px-3 py-3 text-xs leading-5">
      {i18nStore.t('dashboard.hrAdvisor.messages.contextHint')}
    </div>
  </div>
</div>
