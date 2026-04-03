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
  let useCustomSummaryPrompt = $state(false)
  let selectedRunSummarySections = $state<RunSummarySectionKey[]>([...defaultRunSummarySectionKeys])
  let customRunSummaryInstructions = $state('')
  let agentRunSummaryPrompt = $state('')
  let saving = $state(false)
  let archiving = $state(false)

  function applySelectedSummarySections() {
    agentRunSummaryPrompt = buildRunSummaryPrompt(
      selectedRunSummarySections,
      customRunSummaryInstructions,
    )
    useCustomSummaryPrompt = true
  }

  function toggleRunSummarySection(key: RunSummarySectionKey) {
    if (selectedRunSummarySections.includes(key)) {
      if (selectedRunSummarySections.length === 1) {
        return
      }
      selectedRunSummarySections = selectedRunSummarySections.filter((value) => value !== key)
      return
    }

    selectedRunSummarySections = [...selectedRunSummarySections, key]
  }

  $effect(() => {
    const project = appStore.currentProject
    if (!project) {
      projectName = ''
      description = ''
      maxConcurrentAgents = ''
      useCustomSummaryPrompt = false
      selectedRunSummarySections = [...defaultRunSummarySectionKeys]
      customRunSummaryInstructions = ''
      agentRunSummaryPrompt = buildRunSummaryPrompt(defaultRunSummarySectionKeys, '')
      return
    }

    projectName = project.name
    description = project.description
    maxConcurrentAgents =
      typeof project.max_concurrent_agents === 'number' && project.max_concurrent_agents > 0
        ? String(project.max_concurrent_agents)
        : ''
    useCustomSummaryPrompt = (project.agent_run_summary_prompt ?? '').trim() !== ''
    selectedRunSummarySections = [...defaultRunSummarySectionKeys]
    customRunSummaryInstructions = ''
    agentRunSummaryPrompt =
      project.agent_run_summary_prompt?.trim() !== ''
        ? (project.agent_run_summary_prompt ?? '')
        : buildRunSummaryPrompt(defaultRunSummarySectionKeys, '')
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
      const payload = await updateProject(projectId, {
        name: projectName,
        description,
        max_concurrent_agents: parsedMaxConcurrentAgents,
        agent_run_summary_prompt: useCustomSummaryPrompt ? agentRunSummaryPrompt.trim() : '',
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

    <div class="space-y-2">
      <div class="space-y-3 rounded-lg border p-4">
        <div class="space-y-2">
          <Label class="text-sm font-medium">Run summary prompt</Label>
          <label class="flex items-center gap-2">
            <Checkbox
              id="custom-run-summary-prompt"
              checked={useCustomSummaryPrompt}
              onCheckedChange={(checked) => {
                useCustomSummaryPrompt = Boolean(checked)
                if (useCustomSummaryPrompt && agentRunSummaryPrompt.trim() === '') {
                  applySelectedSummarySections()
                }
              }}
            />
            <Label for="custom-run-summary-prompt" class="cursor-pointer">
              Customize run summary prompt
            </Label>
          </label>
          <p class="text-muted-foreground text-xs">
            When disabled, OpenASE uses the built-in post-run summary prompt.
          </p>
        </div>

        {#if useCustomSummaryPrompt}
          <div class="space-y-3 border-t pt-3">
            <div class="space-y-2">
              <div>
                <p class="text-sm font-medium">Summary sections</p>
                <p class="text-muted-foreground text-xs">
                  Select the summary blocks you want to inject into the editable prompt below.
                </p>
              </div>
              <div class="grid gap-2 sm:grid-cols-2">
                {#each runSummarySectionDefinitions as section (section.key)}
                  <label class="flex items-start gap-2 rounded-md border p-3">
                    <Checkbox
                      id={`run-summary-section-${section.key}`}
                      checked={selectedRunSummarySections.includes(section.key)}
                      onCheckedChange={() => toggleRunSummarySection(section.key)}
                    />
                    <div class="min-w-0">
                      <Label
                        for={`run-summary-section-${section.key}`}
                        class="cursor-pointer text-sm font-medium"
                      >
                        {section.title}
                      </Label>
                      <p class="text-muted-foreground mt-1 text-xs">
                        {section.description}
                      </p>
                    </div>
                  </label>
                {/each}
              </div>
              <p class="text-muted-foreground text-xs">Keep at least one section selected.</p>
              <p class="text-muted-foreground text-xs">
                Selections act as a prompt builder. Existing saved prompts are not reverse-parsed
                back into these checkboxes.
              </p>
            </div>

            <div class="space-y-2">
              <Label for="run-summary-custom-instructions">Additional instructions</Label>
              <Textarea
                id="run-summary-custom-instructions"
                bind:value={customRunSummaryInstructions}
                rows={4}
                class="min-h-24 text-sm"
                placeholder="Optional guidance to append after the generated sections."
              />
            </div>

            <div class="flex items-center gap-3">
              <Button type="button" variant="outline" onclick={applySelectedSummarySections}>
                Apply selected sections
              </Button>
              <p class="text-muted-foreground text-xs">
                This regenerates the prompt below and overwrites its current contents.
              </p>
            </div>

            <div class="space-y-2">
              <Label for="agent-run-summary-prompt">Final run summary prompt</Label>
              <Textarea
                id="agent-run-summary-prompt"
                bind:value={agentRunSummaryPrompt}
                rows={12}
                class="min-h-56 font-mono text-sm"
                placeholder="Leave blank to fall back to the built-in post-run summary prompt."
              />
              <p class="text-muted-foreground text-xs">
                You can edit this prompt directly after applying template sections.
              </p>
            </div>
          </div>
        {/if}
      </div>
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
