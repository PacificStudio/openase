<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { Skeleton } from '$ui/skeleton'
  import { cn } from '$lib/utils'
  import { GripVertical } from '@lucide/svelte'
  import SkillAiSidebar from './skill-ai-sidebar.svelte'
  import { createSkillEditorPageController } from './skill-editor-page-controller.svelte'
  import SkillEditorHeader from './skill-editor-header.svelte'
  import SkillMetadataBar from './skill-metadata-bar.svelte'
  import { formatBytes } from './skill-editor-page.helpers'
  import SkillEditorStatusBar from './skill-editor-status-bar.svelte'
  import SkillEditorWorkspace from './skill-editor-workspace.svelte'

  let { skillId }: { skillId: string } = $props()
  const controller = createSkillEditorPageController({ getSkillId: () => skillId })
</script>

<svelte:window
  onkeydown={controller.handleKeydown}
  onbeforeunload={controller.handleBeforeUnload}
/>

{#if controller.loading}
  <div class="flex h-full flex-col">
    <!-- Skeleton header -->
    <header class="border-border flex shrink-0 items-center justify-between border-b px-4 py-2">
      <div class="flex items-center gap-3">
        <Skeleton class="size-7 rounded-md" />
        <div class="flex items-center gap-2">
          <Skeleton class="size-2 rounded-full" />
          <Skeleton class="h-4 w-28" />
          <Skeleton class="h-4 w-12 rounded-full" />
          <Skeleton class="h-4 w-8 rounded-full" />
        </div>
      </div>
      <div class="flex items-center gap-1">
        <Skeleton class="h-7 w-16 rounded-md" />
        <Skeleton class="size-7 rounded-md" />
        <Skeleton class="size-7 rounded-md" />
      </div>
    </header>

    <!-- Skeleton workspace -->
    <div class="flex min-h-0 flex-1">
      <!-- Skeleton file tree -->
      <aside class="border-border w-48 shrink-0 border-r p-2">
        <div class="mb-1.5 flex items-center justify-between px-1">
          <Skeleton class="h-3 w-10" />
        </div>
        <div class="space-y-1">
          {#each { length: 6 } as _, i}
            <div
              class="flex items-center gap-2 px-2 py-1"
              style:padding-left={i > 2 ? '20px' : '8px'}
            >
              <Skeleton class="size-3.5 shrink-0" />
              <Skeleton class="h-3 {i === 0 ? 'w-16' : i < 3 ? 'w-20' : 'w-14'}" />
            </div>
          {/each}
        </div>
      </aside>

      <!-- Skeleton editor -->
      <div class="flex min-w-0 flex-1 flex-col">
        <div class="border-border flex items-center gap-1 border-b px-2 py-1.5">
          <Skeleton class="h-6 w-20 rounded-md" />
          <Skeleton class="h-6 w-16 rounded-md" />
        </div>
        <div class="flex-1 space-y-2 p-4">
          <Skeleton class="h-3.5 w-[72%]" />
          <Skeleton class="h-3.5 w-[55%]" />
          <Skeleton class="h-3.5 w-[88%]" />
          <Skeleton class="h-3.5 w-[45%]" />
          <Skeleton class="h-3.5 w-[67%]" />
          <Skeleton class="h-3.5 w-[78%]" />
          <Skeleton class="h-3.5 w-[52%]" />
          <Skeleton class="h-3.5 w-[90%]" />
          <Skeleton class="h-3.5 w-[40%]" />
          <Skeleton class="h-3.5 w-[63%]" />
          <Skeleton class="h-3.5 w-[80%]" />
          <Skeleton class="h-3.5 w-[48%]" />
        </div>
      </div>

      <!-- (metadata is now an inline bar, no skeleton sidebar needed) -->
    </div>

    <!-- Skeleton status bar -->
    <footer class="border-border bg-muted/30 flex shrink-0 items-center gap-4 border-t px-4 py-1">
      <Skeleton class="h-3 w-12" />
      <Skeleton class="h-3 w-10" />
    </footer>
  </div>
{:else if !controller.skill}
  <div class="text-muted-foreground flex h-full items-center justify-center text-sm">
    Skill not found.
  </div>
{:else}
  <div class="flex h-full flex-col" data-testid="skill-editor-page">
    <SkillEditorHeader
      skill={controller.skill!}
      busy={controller.busy}
      hasDirtyChanges={controller.hasDirtyChanges}
      history={controller.history}
      assistantOpen={controller.showAssistant}
      assistantDisabled={!controller.selectedFileIsText && !controller.showAssistant}
      onNavigateBack={controller.navigateBack}
      onSave={() => void controller.handleSave()}
      onToggleEnabled={() => void controller.handleToggleEnabled()}
      onDelete={() => void controller.handleDelete()}
      onToggleAssistant={() => (controller.showAssistant = !controller.showAssistant)}
    />

    <SkillMetadataBar
      skill={controller.skill!}
      workflows={controller.workflows}
      busy={controller.busy}
      editDescription={controller.editDescription}
      onEditDescriptionChange={controller.setEditDescription}
      onToggleBinding={controller.handleWorkflowBinding}
    />

    <div class="flex min-h-0 flex-1 overflow-hidden">
      <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <SkillEditorWorkspace
          files={controller.draftFiles}
          emptyDirectoryPaths={controller.emptyDirectoryPaths}
          selectedPath={controller.selectedFilePath}
          selectedTreePath={controller.selectedTreePath}
          openFiles={controller.openFiles}
          dirtyPaths={controller.dirtyPaths}
          selectedFile={controller.selectedFile}
          activeContent={controller.activeContent}
          pendingCreate={controller.pendingCreate}
          onSelectTreeNode={controller.selectTreeNode}
          onCreateFile={controller.handleCreateFile}
          onCreateFolder={controller.handleCreateFolder}
          onCreateCommit={controller.handleCreateCommit}
          onCreateCancel={controller.handleCreateCancel}
          onRenameNode={controller.handleRenameNode}
          onDeleteNode={controller.handleDeleteNode}
          onSelectTab={controller.selectFile}
          onCloseTab={controller.closeTab}
          onContentChange={controller.handleContentChange}
        />
      </div>

      {#if controller.showAssistant}
        <div
          role="separator"
          aria-orientation="vertical"
          class={cn(
            'hover:bg-border relative w-1 shrink-0 cursor-col-resize touch-none',
            controller.dragging && 'bg-border',
          )}
          onpointerdown={controller.handleDragStart}
          onpointermove={controller.handleDragMove}
          onpointerup={controller.handleDragEnd}
          onpointercancel={controller.handleDragEnd}
        >
          <div class="absolute inset-y-0 left-1/2 -translate-x-1/2">
            <GripVertical class="text-muted-foreground/60 size-4" />
          </div>
        </div>

        <aside
          class="border-border shrink-0 border-l"
          style:width={`${controller.assistantWidth}px`}
        >
          <SkillAiSidebar
            projectId={appStore.currentProject?.id}
            providers={controller.providers}
            skillId={controller.skill!.id}
            files={controller.draftFiles}
            onApplySuggestion={controller.handleApplyAssistantSuggestion}
          />
        </aside>
      {/if}
    </div>

    <SkillEditorStatusBar
      fileCount={controller.fileCount}
      totalSizeLabel={formatBytes(controller.totalSize)}
      selectedFile={controller.selectedFile}
      dirtyCount={controller.dirtyPaths.size}
    />
  </div>
{/if}
