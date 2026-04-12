<script lang="ts">
  import { Clock, Search as SearchIcon, X } from '@lucide/svelte'
  import { cn } from '$lib/utils'
  import type {
    ProjectConversationWorkspaceSearchResult,
    ProjectConversationWorkspaceTreeEntry,
  } from '$lib/api/chat'
  import { fileIcon } from './project-conversation-workspace-browser-helpers'

  type RecentFile = { repoPath: string; filePath: string }
  type SearchResult = { path: string; kind: 'recent' | 'tree' | 'search' }

  let {
    selectedRepoPath = '',
    recentFiles = [],
    treeNodes = new Map(),
    onSearchPaths,
    onSelectFile,
  }: {
    selectedRepoPath?: string
    recentFiles?: RecentFile[]
    treeNodes?: Map<string, ProjectConversationWorkspaceTreeEntry[]>
    onSearchPaths?: (
      query: string,
      limit?: number,
    ) => Promise<ProjectConversationWorkspaceSearchResult[]>
    onSelectFile?: (path: string) => void
  } = $props()

  let query = $state('')
  let focused = $state(false)
  let inputEl = $state<HTMLInputElement | null>(null)
  let panelEl = $state<HTMLDivElement | null>(null)
  let remoteResults = $state<SearchResult[]>([])
  let searching = $state(false)
  let searchRequestID = 0

  /** Every loaded file path across all tree nodes (deduped). */
  const allLoadedFilePaths = $derived.by(() => {
    const seen = new Set<string>()
    const out: string[] = []
    for (const entries of treeNodes.values()) {
      for (const entry of entries) {
        if (entry.kind === 'file' && !seen.has(entry.path)) {
          seen.add(entry.path)
          out.push(entry.path)
        }
      }
    }
    return out
  })

  const recentForRepo = $derived(
    recentFiles.filter((item) => item.repoPath === selectedRepoPath).map((item) => item.filePath),
  )

  const localResults = $derived.by<SearchResult[]>(() => {
    const trimmed = query.trim().toLowerCase()
    if (trimmed === '') {
      return recentForRepo.slice(0, 8).map((path) => ({ path, kind: 'recent' }))
    }
    const recentSet = new Set(recentForRepo)
    const out: SearchResult[] = []
    for (const path of recentForRepo) {
      if (path.toLowerCase().includes(trimmed)) {
        out.push({ path, kind: 'recent' })
      }
    }
    for (const path of allLoadedFilePaths) {
      if (recentSet.has(path)) continue
      if (path.toLowerCase().includes(trimmed)) {
        out.push({ path, kind: 'tree' })
      }
    }
    return out.slice(0, 20)
  })

  const trimmedQuery = $derived(query.trim())

  const results = $derived(
    trimmedQuery === '' ? localResults : onSearchPaths ? remoteResults : localResults,
  )

  // Panel is visible when focused and there is something to show. Typing with
  // no matches still shows the panel so the empty state can be rendered.
  const showPanel = $derived(focused && (query !== '' || recentForRepo.length > 0))

  $effect(() => {
    const currentQuery = trimmedQuery
    const search = onSearchPaths
    const currentRepoPath = selectedRepoPath
    if (!search || currentQuery === '' || !currentRepoPath) {
      searchRequestID += 1
      remoteResults = []
      searching = false
      return
    }

    const requestID = ++searchRequestID
    searching = true
    void search(currentQuery, 20)
      .then((results) => {
        if (requestID !== searchRequestID) {
          return
        }
        remoteResults = results.map((result) => ({
          path: result.path,
          kind: 'search',
        }))
      })
      .catch(() => {
        if (requestID !== searchRequestID) {
          return
        }
        remoteResults = []
      })
      .finally(() => {
        if (requestID !== searchRequestID) {
          return
        }
        searching = false
      })
  })

  function onInputBlur(event: FocusEvent) {
    // Ignore blurs caused by clicks inside the result panel — those are
    // handled by the button's onclick.
    const related = event.relatedTarget as Node | null
    if (related && panelEl && panelEl.contains(related)) {
      return
    }
    focused = false
  }

  function clearQuery() {
    query = ''
    inputEl?.focus()
  }

  function handlePick(path: string) {
    onSelectFile?.(path)
    query = ''
    focused = false
    inputEl?.blur()
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key !== 'Escape') return
    if (query) {
      query = ''
      return
    }
    inputEl?.blur()
  }

  function fileNameOf(path: string): string {
    return path.split('/').pop() ?? path
  }

  function dirOf(path: string): string {
    const idx = path.lastIndexOf('/')
    return idx === -1 ? '' : path.slice(0, idx)
  }
</script>

<div class="relative px-2 py-1.5">
  <div
    class={cn(
      'flex h-[22px] items-center gap-1 rounded-sm px-1.5 transition-colors',
      focused ? 'bg-background' : 'bg-muted/40 hover:bg-muted/60',
    )}
  >
    <SearchIcon class="text-muted-foreground/50 size-2.5 shrink-0" />
    <input
      bind:this={inputEl}
      bind:value={query}
      type="text"
      placeholder="Search…"
      class="placeholder:text-muted-foreground/40 min-w-0 flex-1 bg-transparent text-[11px] outline-none"
      onfocus={() => (focused = true)}
      onblur={onInputBlur}
      onkeydown={onKeydown}
      data-testid="workspace-browser-search-input"
    />
    {#if query}
      <button
        type="button"
        class="text-muted-foreground/40 hover:text-muted-foreground shrink-0"
        onclick={clearQuery}
        aria-label="Clear search"
      >
        <X class="size-2.5" />
      </button>
    {/if}
  </div>

  {#if showPanel}
    <div
      bind:this={panelEl}
      class="border-border bg-popover text-popover-foreground absolute inset-x-2 top-full z-30 max-h-[280px] overflow-y-auto rounded-md border py-0.5 shadow-md"
      role="listbox"
      tabindex="-1"
      data-testid="workspace-browser-search-panel"
    >
      {#if query === ''}
        <div
          class="text-muted-foreground/50 px-2 pt-1 pb-0.5 text-[9px] font-semibold tracking-wider uppercase"
        >
          Recent
        </div>
      {/if}

      {#if searching}
        <div class="text-muted-foreground/50 px-2.5 py-1.5 text-[10px]">Searching…</div>
      {:else if results.length === 0}
        <div class="text-muted-foreground/50 px-2.5 py-1.5 text-[10px]">
          {query === '' ? 'No recent files' : `No matches for "${query}"`}
        </div>
      {:else}
        {#each results as result (result.path)}
          {@const iconInfo = fileIcon(fileNameOf(result.path))}
          {@const dir = dirOf(result.path)}
          <button
            type="button"
            class="hover:bg-accent hover:text-accent-foreground flex w-full items-center gap-1.5 px-2 py-[3px] text-left text-[11px]"
            onmousedown={(event) => event.preventDefault()}
            onclick={() => handlePick(result.path)}
          >
            {#if result.kind === 'recent' && query === ''}
              <Clock class="text-muted-foreground/40 size-2.5 shrink-0" />
            {:else}
              <iconInfo.icon class={cn('size-2.5 shrink-0', iconInfo.colorClass)} />
            {/if}
            <span class="min-w-0 truncate">{fileNameOf(result.path)}</span>
            {#if dir}
              <span class="text-muted-foreground/40 min-w-0 truncate text-[9px]">{dir}</span>
            {/if}
          </button>
        {/each}
      {/if}
    </div>
  {/if}
</div>
