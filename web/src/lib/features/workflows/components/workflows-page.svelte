<script lang="ts">
  import { onMount } from 'svelte'
  import Button from '$ui/button/button.svelte'
  import { LoaderCircle, PanelRightClose, PanelRight, Save } from '@lucide/svelte'
  import type { WorkflowSummary, HarnessContent, HarnessVariableGroup } from '../types'
  import {
    loadWorkflowPageData,
    saveWorkflowHarness,
    splitHarnessContent,
    toHarnessContent,
  } from '../api'
  import WorkflowList from './workflow-list.svelte'
  import HarnessEditor from './harness-editor.svelte'
  import WorkflowDetailPanel from './workflow-detail-panel.svelte'

  let showDetail = $state(true)
  let selectedId = $state<string | null>(null)
  let workflows = $state<WorkflowSummary[]>([])
  let harnessMap = $state<Record<string, HarnessContent>>({})
  let baselineHarnessMap = $state<Record<string, HarnessContent>>({})
  let variableGroups = $state<HarnessVariableGroup[]>([])
  let orgName = $state<string | null>(null)
  let projectName = $state<string | null>(null)
  let loading = $state(true)
  let saving = $state(false)
  let errorMessage = $state<string | null>(null)
  let saveMessage = $state<string | null>(null)

  const selectedWorkflow = $derived(
    selectedId ? (workflows.find((workflow) => workflow.id === selectedId) ?? null) : null,
  )
  const selectedHarness = $derived(selectedId ? (harnessMap[selectedId] ?? null) : null)
  const selectedBaseline = $derived(selectedId ? (baselineHarnessMap[selectedId] ?? null) : null)
  const isDirty = $derived(
    Boolean(
      selectedHarness &&
      selectedBaseline &&
      selectedHarness.rawContent !== selectedBaseline.rawContent,
    ),
  )
  const dictionarySize = $derived(
    variableGroups.reduce((count, group) => count + group.variables.length, 0),
  )

  onMount(() => {
    const controller = new AbortController()
    void hydrate(controller.signal)
    return () => controller.abort()
  })

  async function hydrate(signal?: AbortSignal) {
    loading = true
    errorMessage = null

    try {
      const data = await loadWorkflowPageData(signal)
      workflows = data.workflows
      harnessMap = { ...data.harnessDocuments }
      baselineHarnessMap = { ...data.harnessDocuments }
      variableGroups = data.variableGroups
      orgName = data.orgName
      projectName = data.projectName
      selectedId = data.workflows[0]?.id ?? null
      saveMessage = null
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        return
      }
      errorMessage = error instanceof Error ? error.message : 'Failed to load workflows.'
    } finally {
      loading = false
    }
  }

  function handleSelectWorkflow(workflowID: string) {
    selectedId = workflowID
    errorMessage = null
    saveMessage = null
  }

  function handleHarnessChange(rawContent: string) {
    if (!selectedId) {
      return
    }

    harnessMap = {
      ...harnessMap,
      [selectedId]: splitHarnessContent(rawContent),
    }
    saveMessage = null
  }

  async function handleSave() {
    if (!selectedId || !selectedHarness || saving || !isDirty) {
      return
    }

    saving = true
    errorMessage = null
    saveMessage = null

    try {
      const document = await saveWorkflowHarness(selectedId, selectedHarness.rawContent)
      const nextContent = toHarnessContent(document)

      harnessMap = {
        ...harnessMap,
        [selectedId]: nextContent,
      }
      baselineHarnessMap = {
        ...baselineHarnessMap,
        [selectedId]: nextContent,
      }
      workflows = workflows.map((workflow) =>
        workflow.id === selectedId
          ? {
              ...workflow,
              harnessPath: document.path,
              version: document.version,
            }
          : workflow,
      )
      saveMessage = `Saved ${document.path} as v${document.version}.`
    } catch (error) {
      errorMessage = error instanceof Error ? error.message : 'Failed to save harness.'
    } finally {
      saving = false
    }
  }
</script>

<div class="flex h-full flex-col">
  <div class="border-border flex items-center justify-between border-b px-4 py-2.5">
    <div>
      <h1 class="text-foreground text-sm font-semibold">Workflows</h1>
      {#if orgName || projectName}
        <p class="text-muted-foreground text-xs">
          {orgName ?? 'No org'}
          {#if projectName}
            / {projectName}
          {/if}
        </p>
      {/if}
    </div>
    <div class="flex items-center gap-2">
      {#if selectedHarness}
        <span class="text-muted-foreground hidden text-xs md:inline">
          {dictionarySize} dictionary entries
        </span>
      {/if}
      {#if saveMessage}
        <span class="hidden text-xs text-emerald-400 lg:inline">{saveMessage}</span>
      {/if}
      <Button
        variant="outline"
        size="sm"
        disabled={!selectedId || !isDirty || saving}
        onclick={handleSave}
      >
        {#if saving}
          <LoaderCircle class="size-4 animate-spin" />
        {:else}
          <Save class="size-4" />
        {/if}
        Save
      </Button>
      <Button variant="ghost" size="sm" onclick={() => (showDetail = !showDetail)}>
        {#if showDetail}
          <PanelRightClose class="size-4" />
        {:else}
          <PanelRight class="size-4" />
        {/if}
      </Button>
    </div>
  </div>

  {#if errorMessage}
    <div
      class="border-destructive/30 bg-destructive/10 text-destructive border-b px-4 py-2 text-xs"
    >
      {errorMessage}
    </div>
  {/if}

  <div class="flex flex-1 overflow-hidden">
    <div class="w-60 shrink-0">
      <WorkflowList {workflows} selectedId={selectedId ?? ''} onselect={handleSelectWorkflow} />
    </div>

    <div class="flex-1 overflow-hidden">
      {#if loading}
        <div
          class="bg-muted/10 text-muted-foreground flex h-full items-center justify-center text-sm"
        >
          <div class="flex items-center gap-2">
            <LoaderCircle class="size-4 animate-spin" />
            Loading workflow editor…
          </div>
        </div>
      {:else if selectedHarness && selectedWorkflow}
        <HarnessEditor
          content={selectedHarness}
          filePath={selectedWorkflow.harnessPath}
          version={selectedWorkflow.version}
          {variableGroups}
          onchange={handleHarnessChange}
        />
      {:else}
        <div
          class="bg-muted/10 text-muted-foreground flex h-full items-center justify-center px-6 text-center text-sm"
        >
          No workflows found in the first available project.
        </div>
      {/if}
    </div>

    {#if showDetail && selectedWorkflow}
      <div class="w-70 shrink-0">
        <WorkflowDetailPanel workflow={selectedWorkflow} />
      </div>
    {/if}
  </div>
</div>
