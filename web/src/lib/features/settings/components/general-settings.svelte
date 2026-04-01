<script lang="ts">
  import { goto } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import { archiveProject, updateProject } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Separator } from '$ui/separator'

  let projectName = $state('')
  let description = $state('')
  let maxConcurrentAgents = $state('1')
  let saving = $state(false)
  let archiving = $state(false)

  $effect(() => {
    const project = appStore.currentProject
    if (!project) {
      projectName = ''
      description = ''
      maxConcurrentAgents = '1'
      return
    }

    projectName = project.name
    description = project.description
    maxConcurrentAgents = String(project.max_concurrent_agents)
  })

  async function handleSave() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    saving = true

    try {
      const payload = await updateProject(projectId, {
        name: projectName,
        description,
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
    <h2 class="text-foreground text-base font-semibold">General</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Project name, description, and archive controls.
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
