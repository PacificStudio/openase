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
  import { Button } from '$ui/button'
  import { Checkbox } from '$ui/checkbox'
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
        error: `${label} must be a non-negative integer or left blank.`,
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
        toastStore.error('Max concurrent agents must be a positive integer or left blank.')
        return
      }
      if (parsedMaxConcurrentAgents === 0 && normalizedMaxConcurrentAgents !== '') {
        toastStore.error('Max concurrent agents must be a positive integer or left blank.')
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
      if (
        retentionEnabled &&
        parsedKeepLatestN.value === 0 &&
        parsedKeepRecentDays.value === 0
      ) {
        toastStore.error(
          'Enable Project AI retention with at least one keep rule: latest conversations or recent days.',
        )
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
      toastStore.success('Project settings saved.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save project settings.',
      )
    } finally {
      saving = false
    }
  }

  async function handleArchive() {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId) return

    const confirmed = window.confirm(
      'Archive this project? The project will remain in the system but leave the active workspace surface.',
    )
    if (!confirmed) return

    archiving = true

    try {
      await archiveProject(projectId)
      appStore.currentProject = null
      await goto(orgId ? organizationPath(orgId) : '/')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to archive project.',
      )
    } finally {
      archiving = false
    }
  }
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">General</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Project name, summary prompt, retention rules, description, and archive controls.
    </p>
  </div>

  <Separator />

  <div class="space-y-4">
    <div class="space-y-2">
      <Label for="project-name">Project name</Label>
      <Input id="project-name" bind:value={projectName} />
    </div>

    <div class="space-y-2">
      <Label for="description">Description</Label>
      <Input id="description" bind:value={description} />
    </div>

    <div class="space-y-3">
      <div class="space-y-1">
        <Label for="agent-run-summary-prompt" class="text-sm font-medium">Run summary prompt</Label>
        <p class="text-muted-foreground text-xs">
          The editor starts with the prompt currently in effect. Click a section pill to append it.
        </p>
        {#if agentRunSummaryPromptSource === 'builtin'}
          <p class="text-muted-foreground text-xs">
            Using the built-in default. Saving without edits keeps the project on the built-in
            prompt.
          </p>
        {:else}
          <p class="text-muted-foreground text-xs">
            Using a project override. Clear the editor and save to revert to the built-in prompt.
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
        placeholder="Leave blank to use the built-in post-run summary prompt."
      />
    </div>

    <div class="space-y-2">
      <Label for="max-agents">Max concurrent agents</Label>
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
        placeholder="Unlimited"
      />
      <p class="text-muted-foreground text-xs">
        Leave blank for unlimited. If set, use a positive integer.
      </p>
    </div>

    <div class="space-y-3 rounded-lg border p-4">
      <div class="space-y-1">
        <h3 class="text-sm font-medium">Project AI retention</h3>
        <p class="text-muted-foreground text-xs">
          Retain a conversation if it is within the latest N conversations or active within the
          last M days.
        </p>
        <p class="text-muted-foreground text-xs">
          Auto-prune skips dirty workspaces by default and preserves live runtimes plus pending
          user interrupts.
        </p>
      </div>

      <div class="flex items-center gap-2">
        <Checkbox id="project-ai-retention-enabled" bind:checked={retentionEnabled} />
        <Label for="project-ai-retention-enabled" class="text-sm font-medium">
          Enable Project AI retention
        </Label>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-2">
          <Label for="keep-latest-conversations">Keep latest conversations</Label>
          <Input
            id="keep-latest-conversations"
            type="number"
            min="0"
            step="1"
            value={keepLatestN}
            oninput={(event) => {
              keepLatestN = (event.currentTarget as HTMLInputElement).value
            }}
            class="w-32"
            placeholder="0"
          />
          <p class="text-muted-foreground text-xs">
            Keep the latest N conversations per user in this project.
          </p>
        </div>

        <div class="space-y-2">
          <Label for="keep-recent-days">Keep recent days</Label>
          <Input
            id="keep-recent-days"
            type="number"
            min="0"
            step="1"
            value={keepRecentDays}
            oninput={(event) => {
              keepRecentDays = (event.currentTarget as HTMLInputElement).value
            }}
            class="w-32"
            placeholder="0"
          />
          <p class="text-muted-foreground text-xs">
            Keep conversations with activity in the last M days.
          </p>
        </div>
      </div>
    </div>
  </div>

  <div class="flex justify-start pt-2">
    <Button onclick={handleSave} disabled={saving}>
      {saving ? 'Saving…' : 'Save changes'}
    </Button>
  </div>

  <Separator />

  <div class="border-destructive/30 bg-destructive/5 rounded-lg border p-4">
    <div class="space-y-2">
      <h3 class="text-foreground text-sm font-medium">Archive project</h3>
      <p class="text-muted-foreground text-sm">
        Move this project out of the active workspace surface after a confirmation step.
      </p>
    </div>

    <div class="mt-4 flex justify-start">
      <Button variant="destructive" onclick={handleArchive} disabled={archiving}>
        {archiving ? 'Archiving…' : 'Archive project'}
      </Button>
    </div>
  </div>
</div>
