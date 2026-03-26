<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation'
  import type { AgentProvider, Machine, Project } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { createProject, createProvider } from '$lib/api/openase'
  import { createEmptyProviderDraft, parseProviderDraft } from '$lib/features/agents/public'
  import type { ProviderDraft } from '$lib/features/agents/public'
  import {
    createProjectDraft,
    parseProjectDraft,
    slugFromName,
    type ProjectCreationDraft,
  } from '$lib/features/catalog-creation/model'
  import { projectPath } from '$lib/stores/app-context'
  import MachineCreationPanel from './machine-creation-panel.svelte'
  import ProjectCreationPanel from './project-creation-panel.svelte'
  import ProviderCreationPanel from './provider-creation-panel.svelte'

  let {
    orgId,
    defaultProviderId = null,
    projects,
    providers,
    machines = [],
  }: {
    orgId: string | null
    defaultProviderId?: string | null
    projects: Project[]
    providers: AgentProvider[]
    machines?: Machine[]
  } = $props()

  const machineConsoleHref = $derived(
    orgId && projects.length > 0 ? projectPath(orgId, projects[0].id, 'machines') : null,
  )

  let projectDraft = $state<ProjectCreationDraft>(createProjectDraft()),
    providerDraft = $state<ProviderDraft>(createEmptyProviderDraft()),
    projectSlugDirty = $state(false),
    creatingProject = $state(false),
    projectError = $state(''),
    creatingProvider = $state(false),
    providerError = $state(''),
    providerFeedback = $state('')

  $effect(() => {
    if (
      projectDraft.name ||
      projectDraft.slug ||
      projectDraft.description ||
      projectDraft.defaultAgentProviderId
    ) {
      return
    }

    projectDraft = createProjectDraft(defaultProviderId)
  })

  function updateProjectName(value: string) {
    projectDraft = {
      ...projectDraft,
      name: value,
      slug: projectSlugDirty ? projectDraft.slug : slugFromName(value),
    }
    projectError = ''
  }

  function updateProjectSlug(value: string) {
    projectSlugDirty = true
    projectDraft = { ...projectDraft, slug: value }
    projectError = ''
  }

  function updateProjectField(field: keyof ProjectCreationDraft, value: string) {
    projectDraft = { ...projectDraft, [field]: value }
    projectError = ''
  }

  function updateProviderField(field: keyof ProviderDraft, value: string) {
    providerDraft = { ...providerDraft, [field]: value }
    providerError = ''
    providerFeedback = ''
  }

  function updateProviderAdapter(value: string) {
    providerDraft = { ...providerDraft, adapterType: value }
    providerError = ''
    providerFeedback = ''
  }

  async function handleCreateProject() {
    if (!orgId) {
      projectError = 'Organization context is unavailable.'
      return
    }

    const parsed = parseProjectDraft(projectDraft)
    if (!parsed.ok) {
      projectError = parsed.error
      return
    }

    creatingProject = true
    projectError = ''

    try {
      const payload = await createProject(orgId, parsed.value)
      await goto(projectPath(orgId, payload.project.id))
    } catch (caughtError) {
      projectError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create project.'
    } finally {
      creatingProject = false
    }
  }

  async function handleCreateProvider() {
    if (!orgId) {
      providerError = 'Organization context is unavailable.'
      return
    }

    const parsed = parseProviderDraft(providerDraft)
    if (!parsed.ok) {
      providerError = parsed.error
      return
    }

    creatingProvider = true
    providerError = ''
    providerFeedback = ''

    try {
      const payload = await createProvider(orgId, parsed.value)
      providerDraft = createEmptyProviderDraft()
      providerFeedback = `Created provider ${payload.provider.name}.`
      await invalidateAll()
    } catch (caughtError) {
      providerError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create provider.'
    } finally {
      creatingProvider = false
    }
  }
</script>

<section class="space-y-4">
  <div>
    <h2 class="text-foreground text-lg font-semibold">Creation lanes</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Start new project scope, register provider inventory, and route into machine capacity from one
      page.
    </p>
  </div>

  <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
    <ProjectCreationPanel
      draft={projectDraft}
      {providers}
      creating={creatingProject}
      error={projectError}
      onNameInput={updateProjectName}
      onSlugInput={updateProjectSlug}
      onFieldChange={updateProjectField}
      onSubmit={() => void handleCreateProject()}
    />

    <div class="space-y-4">
      <ProviderCreationPanel
        draft={providerDraft}
        {machines}
        creating={creatingProvider}
        feedback={providerFeedback}
        error={providerError}
        onFieldChange={updateProviderField}
        onAdapterChange={updateProviderAdapter}
        onSubmit={() => void handleCreateProvider()}
      />

      <MachineCreationPanel
        projectCount={projects.length}
        providerCount={providers.length}
        href={machineConsoleHref}
      />
    </div>
  </div>
</section>
