<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { ApiError } from '$lib/api/client'
  import {
    bindWorkflowSkills,
    saveWorkflowHarness,
    unbindWorkflowSkills,
    validateHarness,
  } from '$lib/api/openase'
  import Button from '$ui/button/button.svelte'
  import { Plus, PanelRightClose, PanelRight } from '@lucide/svelte'
  import type { HarnessValidationIssue } from '$lib/api/contracts'
  import type { WorkflowSummary, HarnessContent } from '../types'
  import { type SkillState, extractBody, extractFrontmatter } from '../model'
  import { createDefaultWorkflow, loadWorkflowHarness, loadWorkflowIndex } from '../data'
  import WorkflowList from './workflow-list.svelte'
  import WorkflowDetailPanel from './workflow-detail-panel.svelte'
  import WorkflowEditorPanel from './workflow-editor-panel.svelte'

  let showDetail = $state(true)
  let loading = $state(false)
  let error = $state('')
  let saving = $state(false)
  let validating = $state(false)
  let creating = $state(false)
  let statusMessage = $state('')
  let workflows = $state<WorkflowSummary[]>([])
  let selectedId = $state('')
  let harness = $state<HarnessContent | null>(null)
  let draftHarness = $state('')
  let skillStates = $state<SkillState[]>([])
  let validationIssues = $state<HarnessValidationIssue[]>([])
  let builtinRoleContent = $state('')
  let statuses = $state<Array<{ id: string; name: string }>>([])

  let selectedWorkflow = $derived(workflows.find((workflow) => workflow.id === selectedId))
  let isDirty = $derived(harness ? draftHarness !== harness.rawContent : false)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      workflows = []
      harness = null
      draftHarness = ''
      skillStates = []
      statuses = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const payload = await loadWorkflowIndex(projectId, selectedId)
        if (cancelled) return

        const nextWorkflows = payload.workflows
        workflows = nextWorkflows
        if (!selectedId || !nextWorkflows.some((workflow) => workflow.id === selectedId)) {
          selectedId = nextWorkflows[0]?.id ?? ''
        }

        skillStates = payload.skillStates
        builtinRoleContent = payload.builtinRoleContent
        statuses = payload.statuses
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load workflows.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  $effect(() => {
    const workflowId = selectedId
    const projectId = appStore.currentProject?.id
    if (!workflowId || !projectId) {
      harness = null
      draftHarness = ''
      validationIssues = []
      return
    }

    let cancelled = false

    const loadHarness = async () => {
      try {
        const payload = await loadWorkflowHarness(projectId, workflowId)
        if (cancelled) return

        harness = payload.harness
        draftHarness = payload.harness.rawContent
        validationIssues = []
        skillStates = payload.skillStates
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load harness.'
      }
    }

    void loadHarness()

    return () => {
      cancelled = true
    }
  })

  async function handleSave() {
    if (!selectedId) return

    saving = true
    statusMessage = ''
    error = ''

    try {
      const payload = await saveWorkflowHarness(selectedId, draftHarness)
      const content = payload.harness.content
      harness = {
        frontmatter: extractFrontmatter(content),
        body: extractBody(content),
        rawContent: content,
      }
      draftHarness = content
      statusMessage = 'Harness saved.'
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to save harness.'
    } finally {
      saving = false
    }
  }

  async function handleValidate() {
    validating = true
    statusMessage = ''
    error = ''

    try {
      const payload = await validateHarness(draftHarness)
      validationIssues = payload.issues
      statusMessage = payload.valid ? 'Harness is valid.' : 'Harness has validation issues.'
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to validate harness.'
    } finally {
      validating = false
    }
  }

  async function handleCreateWorkflow() {
    const projectId = appStore.currentProject?.id
    if (!projectId || statuses.length === 0) return

    creating = true
    statusMessage = ''
    error = ''

    try {
      const payload = await createDefaultWorkflow(
        projectId,
        workflows.length,
        statuses,
        builtinRoleContent,
      )

      workflows = [...workflows, payload.workflow]
      selectedId = payload.selectedId
      statusMessage = 'Workflow created.'
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to create workflow.'
    } finally {
      creating = false
    }
  }

  async function handleToggleSkill(skill: SkillState) {
    if (!selectedId) return

    error = ''
    statusMessage = ''

    try {
      if (skill.bound) {
        await unbindWorkflowSkills(selectedId, [skill.path])
        skillStates = skillStates.map((item) =>
          item.path === skill.path ? { ...item, bound: false } : item,
        )
        statusMessage = `Unbound ${skill.name}.`
        return
      }

      await bindWorkflowSkills(selectedId, [skill.path])
      skillStates = skillStates.map((item) =>
        item.path === skill.path ? { ...item, bound: true } : item,
      )
      statusMessage = `Bound ${skill.name}.`
    } catch (caughtError) {
      error =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update workflow skills.'
    }
  }

  function handleApplyAssistantDraft(content: string) {
    draftHarness = content
    validationIssues = []
    error = ''
    statusMessage = 'Applied AI suggestion to the harness draft.'
  }
</script>

<div class="flex h-full flex-col">
  <div class="border-border flex items-center justify-between border-b px-4 py-2.5">
    <h1 class="text-foreground text-sm font-semibold">Workflows</h1>
    <div class="flex items-center gap-2">
      <Button variant="ghost" size="sm" onclick={() => (showDetail = !showDetail)}>
        {#if showDetail}
          <PanelRightClose class="size-4" />
        {:else}
          <PanelRight class="size-4" />
        {/if}
      </Button>
      <Button size="sm" onclick={handleCreateWorkflow} disabled={creating || statuses.length === 0}>
        <Plus class="size-4" />
        {creating ? 'Creating…' : 'New Workflow'}
      </Button>
    </div>
  </div>

  {#if loading}
    <div class="text-muted-foreground flex flex-1 items-center justify-center text-sm">
      Loading workflows…
    </div>
  {:else if error && workflows.length === 0}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive m-4 rounded-md border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {:else}
    <div class="flex flex-1 overflow-hidden">
      <div class="w-60 shrink-0">
        <WorkflowList {workflows} {selectedId} onselect={(id) => (selectedId = id)} />
      </div>

      <WorkflowEditorPanel
        {selectedWorkflow}
        projectId={appStore.currentProject?.id}
        harness={harness
          ? {
              frontmatter: extractFrontmatter(draftHarness),
              body: extractBody(draftHarness),
              rawContent: draftHarness,
            }
          : null}
        {skillStates}
        {validationIssues}
        {statusMessage}
        {error}
        {saving}
        {validating}
        {isDirty}
        onDraftChange={(raw) => {
          draftHarness = raw
        }}
        onApplyAssistantDraft={handleApplyAssistantDraft}
        onSave={() => {
          void handleSave()
        }}
        onValidate={() => {
          void handleValidate()
        }}
        onToggleSkill={(skill) => {
          void handleToggleSkill(skill)
        }}
      />

      {#if showDetail && selectedWorkflow}
        <div class="w-70 shrink-0">
          <WorkflowDetailPanel workflow={selectedWorkflow} />
        </div>
      {/if}
    </div>
  {/if}
</div>
