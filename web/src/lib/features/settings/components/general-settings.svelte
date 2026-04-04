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
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Separator } from '$ui/separator'
  import { Textarea } from '$ui/textarea'

  let projectName = $state('')
  let description = $state('')
  let maxConcurrentAgents = $state('')
  let agentRunSummaryPrompt = $state('')
  let agentRunSummaryPromptEffectiveBaseline = $state('')
  let agentRunSummaryPromptSource = $state<'builtin' | 'project_override'>('builtin')
  let saving = $state(false)
  let archiving = $state(false)

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

<div class="max-w-lg space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">General</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Project name, summary prompt, description, and archive controls.
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
        bind:value={maxConcurrentAgents}
        class="w-24"
        placeholder="Unlimited"
      />
      <p class="text-muted-foreground text-xs">
        Leave blank for unlimited. If set, use a positive integer.
      </p>
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
