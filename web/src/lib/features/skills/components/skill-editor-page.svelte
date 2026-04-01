<script lang="ts">
  import { goto } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import type { SkillFile, Skill, Workflow } from '$lib/api/contracts'
  import {
    bindSkill,
    deleteSkill,
    disableSkill,
    enableSkill,
    unbindSkill,
    updateSkill,
  } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { projectPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import SkillEditorHeader from './skill-editor-header.svelte'
  import {
    formatBytes,
    loadSkillEditorData,
    resolveEntrypointContent,
    type SkillEditorHistoryEntry,
    selectInitialSkillFiles,
  } from './skill-editor-page.helpers'
  import SkillEditorStatusBar from './skill-editor-status-bar.svelte'
  import SkillEditorWorkspace from './skill-editor-workspace.svelte'
  let { skillId }: { skillId: string } = $props()

  let skill = $state<Skill | null>(null)
  let files = $state<SkillFile[]>([])
  let content = $state('')
  let history = $state<SkillEditorHistoryEntry[]>([])
  let workflows = $state<Workflow[]>([])

  let loading = $state(true)
  let busy = $state(false)
  let editing = $state(false)
  let editDescription = $state('')
  let metadataOpen = $state(true)

  let selectedFilePath = $state<string | null>(null)
  let openFilePaths = $state<string[]>([])
  let editedContents = $state<Map<string, string>>(new Map())

  const dirtyPaths = $derived(new Set(editedContents.keys()))

  const selectedFile = $derived(
    selectedFilePath ? (files.find((f) => f.path === selectedFilePath) ?? null) : null,
  )

  const openFiles = $derived(
    openFilePaths
      .map((p) => files.find((f) => f.path === p))
      .filter((f): f is SkillFile => f !== undefined),
  )

  const activeContent = $derived.by(() => {
    if (!selectedFilePath) return ''
    if (editing && editedContents.has(selectedFilePath)) {
      return editedContents.get(selectedFilePath)!
    }
    const file = files.find((f) => f.path === selectedFilePath)
    return file?.content ?? ''
  })

  const fileCount = $derived(files.length)
  const totalSize = $derived(files.reduce((sum, f) => sum + f.size_bytes, 0))

  // Load skill data
  $effect(() => {
    if (!skillId) return

    let cancelled = false
    loading = true

    const projectId = appStore.currentProject?.id

    void (async () => {
      try {
        const loaded = await loadSkillEditorData(skillId, projectId)

        if (cancelled) return

        skill = loaded.skill
        content = loaded.content
        files = loaded.files
        history = loaded.history
        workflows = loaded.workflows

        const selection = selectInitialSkillFiles(loaded.files)
        selectedFilePath = selection.selectedFilePath
        openFilePaths = selection.openFilePaths
      } catch (err) {
        if (!cancelled) {
          toastStore.error(err instanceof ApiError ? err.detail : 'Failed to load skill.')
        }
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    })()

    return () => {
      cancelled = true
    }
  })

  function navigateBack() {
    const orgId = appStore.currentOrg?.id
    const projectId = appStore.currentProject?.id
    if (orgId && projectId) {
      void goto(projectPath(orgId, projectId, 'skills'))
    }
  }

  function selectFile(path: string) {
    selectedFilePath = path
    if (!openFilePaths.includes(path)) {
      openFilePaths = [...openFilePaths, path]
    }
  }

  function closeTab(path: string) {
    openFilePaths = openFilePaths.filter((p) => p !== path)
    if (selectedFilePath === path) {
      selectedFilePath = openFilePaths.at(-1) ?? null
    }
  }

  function startEditing() {
    if (!skill) return
    editDescription = skill.description
    // Initialize edited contents with current file contents
    editedContents = new Map()
    editing = true
  }

  function cancelEditing() {
    editing = false
    editedContents = new Map()
  }

  function handleContentChange(path: string, value: string) {
    const file = files.find((f) => f.path === path)
    if (file && value === (file.content ?? '')) {
      editedContents.delete(path)
      editedContents = new Map(editedContents)
    } else {
      editedContents = new Map(editedContents).set(path, value)
    }
  }

  async function handleSave() {
    if (!skill) return

    const entrypointContent = resolveEntrypointContent(files, editedContents, content)

    if (!entrypointContent.trim()) {
      toastStore.error('Skill content is required.')
      return
    }

    busy = true
    try {
      await updateSkill(skill.id, {
        description: editDescription.trim(),
        content: entrypointContent,
      })

      const loaded = await loadSkillEditorData(skill.id, appStore.currentProject?.id)

      skill = loaded.skill
      content = loaded.content
      files = loaded.files
      history = loaded.history
      workflows = loaded.workflows
      editing = false
      editedContents = new Map()
      toastStore.success(`Updated ${skill.name}.`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill.')
    } finally {
      busy = false
    }
  }

  async function handleToggleEnabled() {
    if (!skill) return
    busy = true
    try {
      if (skill.is_enabled) {
        const res = await disableSkill(skill.id)
        skill = res.skill
      } else {
        const res = await enableSkill(skill.id)
        skill = res.skill
      }
      toastStore.success(`${skill.is_enabled ? 'Enabled' : 'Disabled'} ${skill.name}.`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill state.')
    } finally {
      busy = false
    }
  }

  async function handleDelete() {
    if (!skill) return
    const confirmed = window.confirm(`Delete "${skill.name}" and remove it from all workflows?`)
    if (!confirmed) return

    busy = true
    try {
      await deleteSkill(skill.id)
      toastStore.success(`Deleted ${skill.name}.`)
      navigateBack()
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to delete skill.')
    } finally {
      busy = false
    }
  }

  async function handleWorkflowBinding(workflowId: string, shouldBind: boolean) {
    if (!skill) return
    busy = true
    try {
      if (shouldBind) {
        const res = await bindSkill(skill.id, [workflowId])
        skill = res.skill
      } else {
        const res = await unbindSkill(skill.id, [workflowId])
        skill = res.skill
      }
      const workflowName = workflows.find((w) => w.id === workflowId)?.name ?? 'workflow'
      toastStore.success(
        `${shouldBind ? 'Bound' : 'Unbound'} ${skill.name} ${shouldBind ? 'to' : 'from'} ${workflowName}.`,
      )
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill binding.')
    } finally {
      busy = false
    }
  }
</script>

{#if loading}
  <div class="text-muted-foreground flex h-full items-center justify-center text-sm">
    Loading skill…
  </div>
{:else if !skill}
  <div class="text-muted-foreground flex h-full items-center justify-center text-sm">
    Skill not found.
  </div>
{:else}
  <div class="flex h-full flex-col" data-testid="skill-editor-page">
    <SkillEditorHeader
      {skill}
      {editing}
      {busy}
      {metadataOpen}
      onNavigateBack={navigateBack}
      onCancelEditing={cancelEditing}
      onSave={() => void handleSave()}
      onStartEditing={startEditing}
      onToggleEnabled={() => void handleToggleEnabled()}
      onDelete={() => void handleDelete()}
      onToggleMetadata={() => (metadataOpen = !metadataOpen)}
    />

    <SkillEditorWorkspace
      {files}
      selectedPath={selectedFilePath}
      {openFiles}
      {dirtyPaths}
      {selectedFile}
      {activeContent}
      {skill}
      {workflows}
      {history}
      {editing}
      {busy}
      {metadataOpen}
      bind:editDescription
      onSelectFile={selectFile}
      onSelectTab={(path) => (selectedFilePath = path)}
      onCloseTab={closeTab}
      onContentChange={handleContentChange}
      onToggleBinding={handleWorkflowBinding}
    />

    <SkillEditorStatusBar
      {fileCount}
      totalSizeLabel={formatBytes(totalSize)}
      {selectedFile}
      dirtyCount={dirtyPaths.size}
      {editing}
    />
  </div>
{/if}
