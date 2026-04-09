<script lang="ts">
  import { untrack } from 'svelte'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import { cn } from '$lib/utils'
  import {
    AlertCircle,
    FileCode2,
    FolderOpen,
    FolderTree,
    GitBranch,
    RefreshCcw,
    X,
  } from '@lucide/svelte'
  import {
    getProjectConversationWorkspace,
    getProjectConversationWorkspaceFilePatch,
    getProjectConversationWorkspaceFilePreview,
    listProjectConversationWorkspaceTree,
    type ProjectConversationWorkspaceDiff,
    type ProjectConversationWorkspaceFilePatch,
    type ProjectConversationWorkspaceFilePreview,
    type ProjectConversationWorkspaceMetadata,
    type ProjectConversationWorkspaceRepoMetadata,
    type ProjectConversationWorkspaceTree,
    type ProjectConversationWorkspaceTreeEntry,
  } from '$lib/api/chat'

  let {
    conversationId = '',
    workspaceDiff = null,
    workspaceDiffLoading = false,
    onClose,
  }: {
    conversationId?: string
    workspaceDiff?: ProjectConversationWorkspaceDiff | null
    workspaceDiffLoading?: boolean
    onClose?: () => void
  } = $props()

  let metadata = $state<ProjectConversationWorkspaceMetadata | null>(null)
  let metadataLoading = $state(false)
  let metadataError = $state('')
  let tree = $state<ProjectConversationWorkspaceTree | null>(null)
  let treeLoading = $state(false)
  let treeError = $state('')
  let preview = $state<ProjectConversationWorkspaceFilePreview | null>(null)
  let patch = $state<ProjectConversationWorkspaceFilePatch | null>(null)
  let fileLoading = $state(false)
  let fileError = $state('')
  let selectedRepoPath = $state('')
  let currentTreePath = $state('')
  let selectedFilePath = $state('')
  let refreshGeneration = $state(0)
  let lastRefreshKey = $state('')
  let lastWorkspaceDiffLoading = $state(false)
  let loadRequestID = 0

  const selectedRepo = $derived(
    metadata?.repos.find((repo) => repo.path === selectedRepoPath) ?? metadata?.repos[0] ?? null,
  )
  const selectedRepoDiff = $derived(
    workspaceDiff?.repos.find((repo) => repo.path === selectedRepoPath) ?? null,
  )
  const currentEntries = $derived(tree?.entries ?? [])
  const dirtyFiles = $derived(selectedRepoDiff?.files ?? [])

  $effect(() => {
    const nextLoading = workspaceDiffLoading
    if (lastWorkspaceDiffLoading && !nextLoading && conversationId) {
      refreshGeneration += 1
    }
    lastWorkspaceDiffLoading = nextLoading
  })

  $effect(() => {
    if (!conversationId) {
      lastRefreshKey = ''
      resetBrowserState()
      return
    }

    const refreshKey = `${conversationId}:${refreshGeneration}`
    if (lastRefreshKey === refreshKey) {
      return
    }
    lastRefreshKey = refreshKey

    untrack(() => {
      void refreshWorkspace(true)
    })
  })

  function resetBrowserState() {
    metadata = null
    metadataLoading = false
    metadataError = ''
    tree = null
    treeLoading = false
    treeError = ''
    preview = null
    patch = null
    fileLoading = false
    fileError = ''
    selectedRepoPath = ''
    currentTreePath = ''
    selectedFilePath = ''
  }

  function repoDirtyLabel(repo: ProjectConversationWorkspaceRepoMetadata) {
    return repo.dirty
      ? `${repo.filesChanged} file${repo.filesChanged === 1 ? '' : 's'} changed`
      : 'Clean'
  }

  function formatTotals(added: number, removed: number) {
    return `+${added} -${removed}`
  }

  function directorySegments(path: string) {
    return path.split('/').filter((segment) => segment.length > 0)
  }

  function joinSegments(segments: string[]) {
    return segments.join('/')
  }

  function statusLabel(status: string) {
    switch (status) {
      case 'added':
        return 'A'
      case 'deleted':
        return 'D'
      case 'renamed':
        return 'R'
      case 'untracked':
        return 'U'
      default:
        return 'M'
    }
  }

  function statusClass(status: string) {
    switch (status) {
      case 'added':
      case 'untracked':
        return 'text-emerald-600'
      case 'deleted':
        return 'text-rose-600'
      case 'renamed':
        return 'text-amber-600'
      default:
        return 'text-sky-600'
    }
  }

  async function refreshWorkspace(preserveSelection: boolean) {
    const activeConversationId = conversationId
    const requestID = ++loadRequestID
    metadataLoading = true
    metadataError = ''

    try {
      const payload = await getProjectConversationWorkspace(activeConversationId)
      if (requestID !== loadRequestID || activeConversationId !== conversationId) return

      metadata = payload.workspace
      if (!payload.workspace.available || payload.workspace.repos.length === 0) {
        selectedRepoPath = ''
        currentTreePath = ''
        selectedFilePath = ''
        tree = null
        preview = null
        patch = null
        treeError = ''
        fileError = ''
        return
      }

      const nextRepoPath =
        preserveSelection &&
        payload.workspace.repos.some((repo) => repo.path === selectedRepoPath) &&
        selectedRepoPath
          ? selectedRepoPath
          : (payload.workspace.repos[0]?.path ?? '')
      const nextTreePath =
        preserveSelection && nextRepoPath === selectedRepoPath ? currentTreePath : ''
      const nextSelectedFilePath =
        preserveSelection && nextRepoPath === selectedRepoPath ? selectedFilePath : ''

      selectedRepoPath = nextRepoPath
      currentTreePath = nextTreePath
      selectedFilePath = nextSelectedFilePath
      await loadTree(nextRepoPath, nextTreePath, { preserveSelectedFile: true, requestID })
    } catch (error) {
      if (requestID !== loadRequestID || activeConversationId !== conversationId) return
      metadata = null
      tree = null
      preview = null
      patch = null
      metadataError =
        error instanceof Error ? error.message : 'Failed to load the Project AI workspace.'
    } finally {
      if (requestID === loadRequestID && activeConversationId === conversationId) {
        metadataLoading = false
      }
    }
  }

  async function loadTree(
    repoPath: string,
    path: string,
    options: { preserveSelectedFile?: boolean; requestID?: number } = {},
  ) {
    if (!repoPath || !conversationId) {
      tree = null
      return
    }

    const requestID = options.requestID ?? ++loadRequestID
    treeLoading = true
    treeError = ''

    try {
      const payload = await listProjectConversationWorkspaceTree(conversationId, {
        repoPath,
        path,
      })
      if (requestID !== loadRequestID || repoPath !== selectedRepoPath) return

      tree = payload.workspaceTree
      currentTreePath = payload.workspaceTree.path

      const canKeepSelectedFile =
        options.preserveSelectedFile &&
        selectedFilePath &&
        selectedFilePath.startsWith(currentTreePath ? `${currentTreePath}/` : '')
      if (!canKeepSelectedFile) {
        selectedFilePath = ''
        preview = null
        patch = null
        fileError = ''
      }
      if (canKeepSelectedFile && selectedFilePath) {
        await loadFile(selectedFilePath, { requestID })
      }
    } catch (error) {
      if (requestID !== loadRequestID || repoPath !== selectedRepoPath) return
      tree = null
      treeError = error instanceof Error ? error.message : 'Failed to load the workspace file tree.'
    } finally {
      if (requestID === loadRequestID && repoPath === selectedRepoPath) {
        treeLoading = false
      }
    }
  }

  async function loadFile(path: string, options: { requestID?: number } = {}) {
    if (!selectedRepoPath || !conversationId) {
      preview = null
      patch = null
      return
    }

    const requestID = options.requestID ?? ++loadRequestID
    fileLoading = true
    fileError = ''
    selectedFilePath = path

    try {
      const [previewPayload, patchPayload] = await Promise.all([
        getProjectConversationWorkspaceFilePreview(conversationId, {
          repoPath: selectedRepoPath,
          path,
        }),
        getProjectConversationWorkspaceFilePatch(conversationId, {
          repoPath: selectedRepoPath,
          path,
        }),
      ])
      if (requestID !== loadRequestID || path !== selectedFilePath) return
      preview = previewPayload.filePreview
      patch = patchPayload.filePatch
    } catch (error) {
      if (requestID !== loadRequestID || path !== selectedFilePath) return
      preview = null
      patch = null
      fileError =
        error instanceof Error ? error.message : 'Failed to load the workspace file details.'
    } finally {
      if (requestID === loadRequestID && path === selectedFilePath) {
        fileLoading = false
      }
    }
  }

  function openRepo(repoPath: string) {
    if (!repoPath || repoPath === selectedRepoPath) return
    selectedRepoPath = repoPath
    currentTreePath = ''
    selectedFilePath = ''
    preview = null
    patch = null
    void loadTree(repoPath, '', { preserveSelectedFile: false })
  }

  function openDirectory(entry: ProjectConversationWorkspaceTreeEntry) {
    if (entry.kind !== 'directory') return
    selectedFilePath = ''
    preview = null
    patch = null
    void loadTree(selectedRepoPath, entry.path, { preserveSelectedFile: false })
  }

  function openFile(entry: ProjectConversationWorkspaceTreeEntry) {
    if (entry.kind !== 'file') return
    void loadFile(entry.path)
  }

  function openDirtyFile(path: string) {
    if (!path) return
    selectedFilePath = path
    void loadFile(path)
  }
</script>

<div
  class="bg-background flex h-full min-h-0 w-full flex-col"
  data-testid="project-conversation-workspace-browser"
>
  <div class="border-border flex items-center gap-2 border-b px-3 py-2">
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-1.5">
        <FolderTree class="text-muted-foreground size-3.5 shrink-0" />
        <h3 class="text-sm font-semibold">Workspace browser</h3>
      </div>
      <p class="text-muted-foreground truncate text-[11px]">
        {metadata?.workspacePath || 'Read-only view of the active Project AI workdir'}
      </p>
    </div>
    <Button
      variant="ghost"
      size="sm"
      class="text-muted-foreground size-7 p-0"
      aria-label="Refresh workspace browser"
      onclick={() => void refreshWorkspace(true)}
      disabled={!conversationId || metadataLoading}
    >
      <RefreshCcw class={cn('size-3.5', metadataLoading && 'animate-spin')} />
    </Button>
    {#if onClose}
      <Button
        variant="ghost"
        size="sm"
        class="text-muted-foreground size-7 p-0"
        aria-label="Close workspace browser"
        onclick={onClose}
      >
        <X class="size-3.5" />
      </Button>
    {/if}
  </div>

  {#if !conversationId}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      Start or reopen a Project AI conversation to inspect its workspace.
    </div>
  {:else if metadataLoading && !metadata}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      Loading workspace…
    </div>
  {:else if metadataError}
    <div class="flex flex-1 items-center justify-center px-6">
      <div class="max-w-sm space-y-3 text-center">
        <div
          class="bg-destructive/10 text-destructive mx-auto flex size-10 items-center justify-center rounded-full"
        >
          <AlertCircle class="size-4" />
        </div>
        <p class="text-sm font-medium">Workspace browser unavailable</p>
        <p class="text-muted-foreground text-sm">{metadataError}</p>
      </div>
    </div>
  {:else if !metadata?.available}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      The workspace will appear after Project AI provisions the conversation workdir.
    </div>
  {:else}
    <div class="grid min-h-0 flex-1 grid-cols-[18rem_minmax(0,1fr)]">
      <div class="border-border flex min-h-0 flex-col border-r">
        <div class="border-border flex flex-wrap gap-2 border-b p-3">
          {#each metadata?.repos ?? [] as repo}
            <button
              type="button"
              class={cn(
                'border-border bg-background hover:bg-muted/40 flex min-w-0 flex-1 flex-col rounded-lg border px-3 py-2 text-left transition-colors',
                repo.path === selectedRepoPath && 'border-primary bg-primary/5',
              )}
              onclick={() => openRepo(repo.path)}
            >
              <span class="truncate text-sm font-medium">{repo.name}</span>
              <span class="text-muted-foreground truncate font-mono text-[10px]">{repo.path}</span>
              <span class="mt-1 truncate font-mono text-[10px]">
                {repoDirtyLabel(repo)} · {formatTotals(repo.added, repo.removed)}
              </span>
            </button>
          {/each}
        </div>

        {#if selectedRepo}
          <div class="border-border space-y-2 border-b px-3 py-2 text-[11px]">
            <div class="flex items-center gap-1.5">
              <GitBranch class="text-muted-foreground size-3" />
              <span class="font-medium">{selectedRepo.branch}</span>
            </div>
            <div class="text-muted-foreground flex flex-wrap items-center gap-1 font-mono">
              <span>{selectedRepo.headCommit}</span>
              <span aria-hidden="true">·</span>
              <span class="min-w-0 truncate">{selectedRepo.headSummary}</span>
            </div>
          </div>
        {/if}

        <div class="min-h-0 flex-1">
          <ScrollArea class="h-full">
            <div class="space-y-3 p-3">
              <section class="space-y-2">
                <div class="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
                  Tree
                </div>
                <div class="flex flex-wrap gap-1">
                  <button
                    type="button"
                    class={cn(
                      'border-border hover:bg-muted/40 rounded-full border px-2 py-0.5 text-[11px]',
                      currentTreePath === '' && 'border-primary bg-primary/5',
                    )}
                    onclick={() =>
                      void loadTree(selectedRepoPath, '', { preserveSelectedFile: true })}
                  >
                    repo root
                  </button>
                  {#each directorySegments(currentTreePath) as segment, index}
                    <button
                      type="button"
                      class="border-border hover:bg-muted/40 rounded-full border px-2 py-0.5 text-[11px]"
                      onclick={() =>
                        void loadTree(
                          selectedRepoPath,
                          joinSegments(directorySegments(currentTreePath).slice(0, index + 1)),
                          { preserveSelectedFile: true },
                        )}
                    >
                      {segment}
                    </button>
                  {/each}
                </div>

                {#if treeLoading && !tree}
                  <p class="text-muted-foreground text-sm">Loading files…</p>
                {:else if treeError}
                  <p class="text-destructive text-sm">{treeError}</p>
                {:else if currentEntries.length === 0}
                  <p class="text-muted-foreground text-sm">This directory is empty.</p>
                {:else}
                  <div class="space-y-1">
                    {#each currentEntries as entry}
                      <button
                        type="button"
                        class={cn(
                          'hover:bg-muted/40 flex w-full items-center gap-2 rounded-md px-2 py-1 text-left text-sm transition-colors',
                          entry.path === selectedFilePath && 'bg-primary/5 text-primary',
                        )}
                        onclick={() =>
                          entry.kind === 'directory' ? openDirectory(entry) : openFile(entry)}
                      >
                        {#if entry.kind === 'directory'}
                          <FolderOpen class="text-muted-foreground size-3.5 shrink-0" />
                        {:else}
                          <FileCode2 class="text-muted-foreground size-3.5 shrink-0" />
                        {/if}
                        <span class="min-w-0 flex-1 truncate font-mono text-[12px]"
                          >{entry.name}</span
                        >
                        {#if entry.kind === 'file'}
                          <span class="text-muted-foreground shrink-0 text-[10px]"
                            >{entry.sizeBytes} B</span
                          >
                        {/if}
                      </button>
                    {/each}
                  </div>
                {/if}
              </section>

              <section class="space-y-2">
                <div class="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
                  Git status
                </div>
                {#if dirtyFiles.length === 0}
                  <p class="text-muted-foreground text-sm">No changed files in this repo.</p>
                {:else}
                  <div class="space-y-1">
                    {#each dirtyFiles as file}
                      <button
                        type="button"
                        class={cn(
                          'hover:bg-muted/40 flex w-full items-center gap-2 rounded-md px-2 py-1 text-left text-sm transition-colors',
                          file.path === selectedFilePath && 'bg-primary/5 text-primary',
                        )}
                        onclick={() => openDirtyFile(file.path)}
                      >
                        <span
                          class={cn(
                            'w-3 shrink-0 font-mono text-[10px] font-bold',
                            statusClass(file.status),
                          )}
                        >
                          {statusLabel(file.status)}
                        </span>
                        <span class="min-w-0 flex-1 truncate font-mono text-[12px]"
                          >{file.path}</span
                        >
                        <span class="text-muted-foreground shrink-0 font-mono text-[10px]">
                          {formatTotals(file.added, file.removed)}
                        </span>
                      </button>
                    {/each}
                  </div>
                {/if}
              </section>
            </div>
          </ScrollArea>
        </div>
      </div>

      <div class="min-h-0">
        <ScrollArea class="h-full">
          <div class="space-y-4 p-4">
            {#if !selectedRepo}
              <p class="text-muted-foreground text-sm">Select a repo to browse its files.</p>
            {:else}
              <div class="space-y-1">
                <p class="text-xs font-medium tracking-wide uppercase">Selection</p>
                <p class="font-mono text-sm">
                  {selectedFilePath || currentTreePath || selectedRepo.path}
                </p>
              </div>

              {#if fileError}
                <div class="border-destructive/20 bg-destructive/5 rounded-lg border p-3">
                  <p class="text-destructive text-sm">{fileError}</p>
                </div>
              {:else if selectedFilePath}
                <section class="space-y-3">
                  <div class="border-border bg-muted/20 rounded-lg border">
                    <div class="border-border flex items-center justify-between border-b px-3 py-2">
                      <div>
                        <p class="text-sm font-medium">Preview</p>
                        {#if preview}
                          <p class="text-muted-foreground text-[11px]">
                            {preview.mediaType} · {preview.sizeBytes} B
                          </p>
                        {/if}
                      </div>
                      {#if fileLoading}
                        <span class="text-muted-foreground text-[11px]">Loading…</span>
                      {/if}
                    </div>

                    {#if preview?.previewKind === 'binary'}
                      <p class="text-muted-foreground px-3 py-4 text-sm">
                        Binary files are not rendered inline in the read-only browser.
                      </p>
                    {:else if preview}
                      <pre
                        class="overflow-x-auto p-3 font-mono text-[12px] leading-5 whitespace-pre-wrap">{preview.content}</pre>
                    {:else if fileLoading}
                      <p class="text-muted-foreground px-3 py-4 text-sm">Loading preview…</p>
                    {:else}
                      <p class="text-muted-foreground px-3 py-4 text-sm">
                        Choose a file to load its preview.
                      </p>
                    {/if}
                  </div>

                  <div class="border-border bg-muted/20 rounded-lg border">
                    <div class="border-border flex items-center justify-between border-b px-3 py-2">
                      <div>
                        <p class="text-sm font-medium">Diff</p>
                        {#if patch}
                          <p class="text-muted-foreground text-[11px]">
                            {patch.status} · {patch.diffKind}
                          </p>
                        {/if}
                      </div>
                    </div>

                    {#if patch?.diffKind === 'text'}
                      <pre
                        class="overflow-x-auto p-3 font-mono text-[12px] leading-5 whitespace-pre-wrap">{patch.diff}</pre>
                    {:else if patch?.diffKind === 'binary'}
                      <p class="text-muted-foreground px-3 py-4 text-sm">
                        Binary changes are detected, but the diff body is not rendered inline.
                      </p>
                    {:else if patch?.diffKind === 'none'}
                      <p class="text-muted-foreground px-3 py-4 text-sm">
                        No diff against `HEAD` for this file.
                      </p>
                    {:else}
                      <p class="text-muted-foreground px-3 py-4 text-sm">
                        Choose a file to load its git diff.
                      </p>
                    {/if}
                  </div>
                </section>
              {:else}
                <div
                  class="text-muted-foreground rounded-lg border border-dashed px-4 py-8 text-center text-sm"
                >
                  Select a file from the tree or git status list to inspect it.
                </div>
              {/if}
            {/if}
          </div>
        </ScrollArea>
      </div>
    </div>
  {/if}
</div>
