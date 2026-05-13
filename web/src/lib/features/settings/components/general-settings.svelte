<script lang="ts">
  import { goto } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import { archiveProject, updateProject } from '$lib/api/openase'
  import {
    buildRunSummaryPrompt,
    defaultRunSummarySectionKeys,
    runSummarySectionDefinitions,
    type RunSummarySectionKey,
  } from '$lib/features/settings/run-summary-prompt-template'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import ProjectArchivePanel from '$lib/features/settings/components/project-archive-panel.svelte'
  import ProjectAIRetentionSettings from '$lib/features/settings/components/project-ai-retention-settings.svelte'
  import ProjectPipelinePresetPanel from '$lib/features/settings/components/project-pipeline-preset-panel.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Separator } from '$ui/separator'
  import { Textarea } from '$ui/textarea'

  let projectName = $state('')
  let description = $state('')
  let maxConcurrentAgents = $state('')
  let retentionEnabled = $state(false)
  let keepLatestN = $state('')
  let keepRecentDays = $state('')
  let agentRunSummaryPrompt = $state('')
  let agentRunSummaryPromptEffectiveBaseline = $state('')
  let agentRunSummaryPromptSource = $state<'builtin' | 'project_override'>('builtin')
  let saving = $state(false)
  let archiving = $state(false)

  function parseNonNegativeIntegerOrBlank(raw: string, label: string) {
    const normalized = raw.trim()
    if (!normalized) {
      return { value: 0 }
    }
    const parsed = Number(normalized)
    if (!Number.isInteger(parsed) || parsed < 0) {
      return {
        value: 0,
        error: i18nStore.t('settings.general.errors.nonNegativeInteger', { label }),
      }
    }
    return { value: parsed }
  }

  function appendSummarySection(key: RunSummarySectionKey) {
    const section = runSummarySectionDefinitions.find((d) => d.key === key)
    if (!section) return
    const snippet = `${section.heading}\n${section.instruction}`
    const current = agentRunSummaryPrompt.trimEnd()
    agentRunSummaryPrompt = current ? `${current}\n\n${snippet}` : snippet
  }

  $effect(() => {
    const project = appStore.currentProject
    if (!project) {
      projectName = ''
      description = ''
      maxConcurrentAgents = ''
      retentionEnabled = false
      keepLatestN = ''
      keepRecentDays = ''
      agentRunSummaryPromptSource = 'builtin'
      agentRunSummaryPromptEffectiveBaseline = buildRunSummaryPrompt(
        defaultRunSummarySectionKeys,
        '',
      )
      agentRunSummaryPrompt = agentRunSummaryPromptEffectiveBaseline
      return
    }

    projectName = project.name
    description = project.description
    maxConcurrentAgents =
      typeof project.max_concurrent_agents === 'number' && project.max_concurrent_agents > 0
        ? String(project.max_concurrent_agents)
        : ''
    retentionEnabled = project.project_ai_retention?.enabled ?? false
    keepLatestN =
      typeof project.project_ai_retention?.keep_latest_n === 'number' &&
      project.project_ai_retention.keep_latest_n > 0
        ? String(project.project_ai_retention.keep_latest_n)
        : ''
    keepRecentDays =
      typeof project.project_ai_retention?.keep_recent_days === 'number' &&
      project.project_ai_retention.keep_recent_days > 0
        ? String(project.project_ai_retention.keep_recent_days)
        : ''
    agentRunSummaryPromptSource =
      project.agent_run_summary_prompt_source ??
      (project.agent_run_summary_prompt?.trim() !== '' ? 'project_override' : 'builtin')
    agentRunSummaryPromptEffectiveBaseline =
      project.effective_agent_run_summary_prompt?.trim() !== ''
        ? (project.effective_agent_run_summary_prompt ?? '')
        : project.agent_run_summary_prompt?.trim() !== ''
          ? (project.agent_run_summary_prompt ?? '')
          : buildRunSummaryPrompt(defaultRunSummarySectionKeys, '')
    agentRunSummaryPrompt = agentRunSummaryPromptEffectiveBaseline
  })

  async function handleSave() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    saving = true

    try {
      const normalizedMaxConcurrentAgents = maxConcurrentAgents.trim()
      const parsedMaxConcurrentAgents = normalizedMaxConcurrentAgents
        ? Number(normalizedMaxConcurrentAgents)
        : 0
      if (!Number.isInteger(parsedMaxConcurrentAgents) || parsedMaxConcurrentAgents < 0) {
        toastStore.error(i18nStore.t('settings.general.errors.invalidMaxConcurrentAgents'))
        return
      }
      if (parsedMaxConcurrentAgents === 0 && normalizedMaxConcurrentAgents !== '') {
        toastStore.error(i18nStore.t('settings.general.errors.invalidMaxConcurrentAgents'))
        return
      }
      const parsedKeepLatestN = parseNonNegativeIntegerOrBlank(
        keepLatestN,
        'Keep latest conversations',
      )
      if (parsedKeepLatestN.error) {
        toastStore.error(parsedKeepLatestN.error)
        return
      }
      const parsedKeepRecentDays = parseNonNegativeIntegerOrBlank(
        keepRecentDays,
        'Keep recent days',
      )
      if (parsedKeepRecentDays.error) {
        toastStore.error(parsedKeepRecentDays.error)
        return
      }
      if (retentionEnabled && parsedKeepLatestN.value === 0 && parsedKeepRecentDays.value === 0) {
        toastStore.error(i18nStore.t('settings.general.errors.retentionRule'))
        return
      }
      const normalizedPrompt = agentRunSummaryPrompt.trim()
      const normalizedEffectiveBaseline = agentRunSummaryPromptEffectiveBaseline.trim()
      const agentRunSummaryPromptPayload =
        agentRunSummaryPromptSource === 'builtin' &&
        normalizedPrompt === normalizedEffectiveBaseline
          ? ''
          : normalizedPrompt
      const payload = await updateProject(projectId, {
        name: projectName,
        description,
        max_concurrent_agents: parsedMaxConcurrentAgents,
        agent_run_summary_prompt: agentRunSummaryPromptPayload,
        project_ai_retention: {
          enabled: retentionEnabled,
          keep_latest_n: parsedKeepLatestN.value,
          keep_recent_days: parsedKeepRecentDays.value,
        },
      })
      appStore.currentProject = payload.project
      toastStore.success(i18nStore.t('settings.general.messages.saveSuccess'))
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('settings.general.errors.saveFailure'),
      )
    } finally {
      saving = false
    }
  }

  async function handleArchive() {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId) return

    const confirmed = window.confirm(i18nStore.t('settings.general.archive.confirmation'))
    if (!confirmed) return

    archiving = true

    try {
      await archiveProject(projectId)
      appStore.currentProject = null
      await goto(orgId ? organizationPath(orgId) : '/')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('settings.archive.errors.failure'),
      )
    } finally {
      archiving = false
    }
  }
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">
      {i18nStore.t('settings.general.heading')}
    </h2>
    <p class="text-muted-foreground mt-1 text-sm">
      {i18nStore.t('settings.general.description')}
    </p>
  </div>

  <Separator />

  <div class="space-y-4">
    <div class="space-y-2">
      <Label for="project-name">{i18nStore.t('settings.general.labels.projectName')}</Label>
      <Input id="project-name" bind:value={projectName} />
    </div>

    <div class="space-y-2">
      <Label for="description">{i18nStore.t('settings.general.labels.description')}</Label>
      <Input id="description" bind:value={description} />
    </div>

    <div class="space-y-3">
      <div class="space-y-1">
        <Label for="agent-run-summary-prompt" class="text-sm font-medium">
          {i18nStore.t('settings.general.labels.runSummaryPrompt')}
        </Label>
        <p class="text-muted-foreground text-xs">
          {i18nStore.t('settings.general.hints.runSummaryGuide')}
        </p>
        {#if agentRunSummaryPromptSource === 'builtin'}
          <p class="text-muted-foreground text-xs">
            {i18nStore.t('settings.general.hints.runSummaryBuiltin')}
          </p>
        {:else}
          <p class="text-muted-foreground text-xs">
            {i18nStore.t('settings.general.hints.runSummaryOverride')}
          </p>
        {/if}
      </div>

      <div class="flex flex-wrap gap-1.5">
        {#each runSummarySectionDefinitions as section (section.key)}
          <button
            type="button"
            class="border-border hover:bg-muted text-foreground inline-flex items-center rounded-full border px-2.5 py-1 text-[11px] font-medium transition-colors"
            title={section.description}
            onclick={() => appendSummarySection(section.key)}
          >
            {section.title}
          </button>
        {/each}
      </div>

      <Textarea
        id="agent-run-summary-prompt"
        bind:value={agentRunSummaryPrompt}
        rows={14}
        class="min-h-56 font-mono text-xs leading-relaxed"
        placeholder={i18nStore.t('settings.general.hints.runSummaryPlaceholder')}
      />
    </div>

    <div class="space-y-2">
      <Label for="max-agents">{i18nStore.t('settings.general.labels.maxConcurrentAgents')}</Label>
      <Input
        id="max-agents"
        type="number"
        min="1"
        step="1"
        value={maxConcurrentAgents}
        oninput={(event) => {
          maxConcurrentAgents = (event.currentTarget as HTMLInputElement).value
        }}
        class="w-24"
        placeholder={i18nStore.t('settings.general.placeholders.unlimited')}
      />
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('settings.general.hints.maxAgentsReminder')}
      </p>
    </div>

    <ProjectAIRetentionSettings
      bind:enabled={retentionEnabled}
      bind:keepLatestN
      bind:keepRecentDays
    />
  </div>

  <div class="flex justify-start pt-2">
    <Button onclick={handleSave} disabled={saving}>
      {saving
        ? i18nStore.t('settings.general.button.saving')
        : i18nStore.t('settings.general.button.saveChanges')}
    </Button>
  </div>

  <Separator />
  <ProjectPipelinePresetPanel />

  <Separator />
  <ProjectArchivePanel {archiving} onArchive={handleArchive} />
</div>
