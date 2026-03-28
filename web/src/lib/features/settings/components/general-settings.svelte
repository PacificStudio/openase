<script lang="ts">
  import { goto } from '$app/navigation'
  import {
    getSettingsSectionCapability,
    capabilityStateClasses,
    capabilityStateLabel,
  } from '$lib/features/capabilities'
  import { appStore } from '$lib/stores/app.svelte'
  import { archiveProject, listWorkflows, updateProject } from '$lib/api/openase'
  import type { Workflow } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { organizationPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Button } from '$ui/button'
  import * as Select from '$ui/select'
  import { Separator } from '$ui/separator'

  const generalCapability = getSettingsSectionCapability('general')

  let projectName = $state('')
  let description = $state('')
  let defaultWorkflow = $state('')
  let maxConcurrentAgents = $state('1')
  let workflows = $state<Array<{ value: string; label: string }>>([])
  let saving = $state(false)
  let archiving = $state(false)
  let loadingWorkflows = $state(false)

  $effect(() => {
    const project = appStore.currentProject
    if (!project) {
      projectName = ''
      description = ''
      defaultWorkflow = ''
      maxConcurrentAgents = '1'
      return
    }

    projectName = project.name
    description = project.description
    defaultWorkflow = project.default_workflow_id ?? ''
    maxConcurrentAgents = String(project.max_concurrent_agents)
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      workflows = []
      loadingWorkflows = false
      return
    }

    let cancelled = false
    const load = async () => {
      loadingWorkflows = true
      try {
        const payload = await listWorkflows(projectId)
        if (cancelled) return
        workflows = payload.workflows.map((workflow: Workflow) => ({
          value: workflow.id,
          label: workflow.name,
        }))
      } finally {
        if (!cancelled) {
          loadingWorkflows = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  async function handleSave() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    saving = true

    try {
      const payload = await updateProject(projectId, {
        name: projectName,
        description,
        default_workflow_id: defaultWorkflow || null,
        max_concurrent_agents: Number(maxConcurrentAgents),
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
    <div class="flex items-center gap-2">
      <h2 class="text-foreground text-base font-semibold">General</h2>
      <span
        class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(generalCapability.state)}`}
      >
        {capabilityStateLabel(generalCapability.state)}
      </span>
    </div>
    <p class="text-muted-foreground mt-1 text-sm">{generalCapability.summary}</p>
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
      <Label>Default workflow</Label>
      <Select.Root
        type="single"
        onValueChange={(v) => {
          defaultWorkflow = v || ''
        }}
      >
        <Select.Trigger class="w-full">
          {#if loadingWorkflows}
            Loading workflows…
          {:else}
            {workflows.find((w) => w.value === defaultWorkflow)?.label ?? 'No default workflow'}
          {/if}
        </Select.Trigger>
        <Select.Content>
          {#each workflows as w (w.value)}
            <Select.Item value={w.value}>{w.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    <div class="space-y-2">
      <Label for="max-agents">Max concurrent agents</Label>
      <Input id="max-agents" type="number" bind:value={maxConcurrentAgents} class="w-24" />
      <p class="text-muted-foreground text-xs">
        Limit the number of agents running simultaneously.
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
