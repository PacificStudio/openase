<script lang="ts">
  import type {
    GitHubRepositoryNamespaceRecord,
    GitHubRepositoryRecord,
    ProjectRepoRecord,
  } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { Plus, Link, Loader2, CheckCircle2, GitBranch, FolderGit2, Search } from '@lucide/svelte'

  let {
    mode = $bindable<'choose' | 'create' | 'link'>(),
    creating = false,
    linking = false,
    searchingRepos = false,
    newRepoName = $bindable(''),
    newRepoVisibility = $bindable<'private' | 'public'>('private'),
    newRepoDefaultBranch = $bindable('main'),
    selectedNamespace = $bindable(''),
    repoSearchQuery = $bindable(''),
    searchResults = [],
    linkRepoUrl = $bindable(''),
    linkRepoName = $bindable(''),
    linkRepoBranch = $bindable('main'),
    namespaces = [],
    repos = [],
    hasRepos = false,
    onEnterCreateMode,
    onEnterLinkMode,
    onSearchRepos,
    onSelectSearchResult,
    onCreateRepo,
    onLinkRepo,
  }: {
    mode: 'choose' | 'create' | 'link'
    creating?: boolean
    linking?: boolean
    searchingRepos?: boolean
    newRepoName?: string
    newRepoVisibility?: 'private' | 'public'
    newRepoDefaultBranch?: string
    selectedNamespace?: string
    repoSearchQuery?: string
    searchResults?: GitHubRepositoryRecord[]
    linkRepoUrl?: string
    linkRepoName?: string
    linkRepoBranch?: string
    namespaces?: GitHubRepositoryNamespaceRecord[]
    repos?: ProjectRepoRecord[]
    hasRepos?: boolean
    onEnterCreateMode?: () => void
    onEnterLinkMode?: () => void
    onSearchRepos?: () => void | Promise<void>
    onSelectSearchResult?: (repo: GitHubRepositoryRecord) => void
    onCreateRepo?: () => void | Promise<void>
    onLinkRepo?: () => void | Promise<void>
  } = $props()
</script>

<div class="space-y-4">
  {#if hasRepos}
    <div class="space-y-2">
      {#each repos as repo (repo.id)}
        <div
          class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 dark:border-emerald-900/50 dark:bg-emerald-950/30"
        >
          <CheckCircle2 class="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
          <FolderGit2 class="text-muted-foreground size-4 shrink-0" />
          <div class="min-w-0 flex-1">
            <p class="text-foreground truncate text-sm font-medium">{repo.name}</p>
            <p class="text-muted-foreground truncate text-xs">{repo.repository_url}</p>
          </div>
          <span class="text-muted-foreground flex items-center gap-1 text-xs">
            <GitBranch class="size-3" />
            {repo.default_branch}
          </span>
        </div>
      {/each}
    </div>

    {#if mode === 'choose'}
      <Button variant="outline" size="sm" onclick={onEnterCreateMode}>
        <Plus class="mr-1.5 size-3.5" />
        添加更多仓库
      </Button>
    {/if}
  {/if}

  {#if !hasRepos || mode !== 'choose'}
    {#if mode === 'choose'}
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <button
          type="button"
          class="border-border hover:border-primary/50 hover:bg-primary/5 flex items-start gap-3 rounded-lg border p-4 text-left transition-colors"
          onclick={onEnterCreateMode}
        >
          <div class="bg-primary/10 flex size-9 shrink-0 items-center justify-center rounded-lg">
            <Plus class="text-primary size-4" />
          </div>
          <div>
            <p class="text-foreground text-sm font-medium">创建新仓库</p>
            <p class="text-muted-foreground mt-0.5 text-xs">在 GitHub 上创建新的代码仓库</p>
          </div>
        </button>

        <button
          type="button"
          class="border-border hover:border-primary/50 hover:bg-primary/5 flex items-start gap-3 rounded-lg border p-4 text-left transition-colors"
          onclick={onEnterLinkMode}
        >
          <div class="bg-primary/10 flex size-9 shrink-0 items-center justify-center rounded-lg">
            <Link class="text-primary size-4" />
          </div>
          <div>
            <p class="text-foreground text-sm font-medium">关联已有仓库</p>
            <p class="text-muted-foreground mt-0.5 text-xs">关联一个已存在的 Git 仓库</p>
          </div>
        </button>
      </div>
    {:else if mode === 'create'}
      <div class="space-y-3">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">命名空间</p>
            <Select.Root
              type="single"
              value={selectedNamespace}
              onValueChange={(v) => {
                if (v) selectedNamespace = v
              }}
            >
              <Select.Trigger class="h-9 w-full text-sm">
                {selectedNamespace || '选择命名空间'}
              </Select.Trigger>
              <Select.Content>
                {#each namespaces as ns (ns.login)}
                  <Select.Item value={ns.login}>{ns.login}</Select.Item>
                {/each}
              </Select.Content>
            </Select.Root>
          </div>
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">仓库名称</p>
            <Input bind:value={newRepoName} placeholder="my-project" class="h-9 text-sm" />
          </div>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">可见性</p>
            <Select.Root
              type="single"
              value={newRepoVisibility}
              onValueChange={(v) => {
                if (v) newRepoVisibility = v as 'private' | 'public'
              }}
            >
              <Select.Trigger class="h-9 w-full text-sm">
                {newRepoVisibility === 'private' ? 'Private' : 'Public'}
              </Select.Trigger>
              <Select.Content>
                <Select.Item value="private">Private</Select.Item>
                <Select.Item value="public">Public</Select.Item>
              </Select.Content>
            </Select.Root>
          </div>
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">默认分支</p>
            <Input bind:value={newRepoDefaultBranch} placeholder="main" class="h-9 text-sm" />
          </div>
        </div>
        <div class="flex items-center gap-2">
          <Button
            onclick={onCreateRepo}
            disabled={creating || !newRepoName.trim() || !selectedNamespace}
          >
            {#if creating}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              创建中...
            {:else}
              <Plus class="mr-1.5 size-3.5" />
              创建并关联
            {/if}
          </Button>
          <Button variant="ghost" size="sm" onclick={() => (mode = 'choose')}>返回</Button>
        </div>
      </div>
    {:else}
      <div class="space-y-3">
        <div>
          <p class="text-foreground mb-1 text-xs font-medium">搜索或浏览 GitHub 仓库</p>
          <div class="flex items-center gap-2">
            <Input
              bind:value={repoSearchQuery}
              placeholder="搜索仓库名称，或直接浏览最近可访问仓库..."
              class="h-9 flex-1 text-sm"
              onkeydown={(e) => {
                if (e.key === 'Enter') void onSearchRepos?.()
              }}
            />
            <Button variant="outline" size="sm" onclick={onSearchRepos} disabled={searchingRepos}>
              {#if searchingRepos}
                <Loader2 class="size-3.5 animate-spin" />
              {:else}
                <Search class="size-3.5" />
              {/if}
            </Button>
          </div>
        </div>

        {#if searchResults.length > 0}
          <div class="border-border max-h-48 space-y-1 overflow-y-auto rounded-lg border p-1">
            {#each searchResults as result (result.full_name)}
              <button
                type="button"
                class="hover:bg-muted w-full rounded-md px-3 py-2 text-left transition-colors"
                onclick={() => onSelectSearchResult?.(result)}
              >
                <p class="text-foreground text-sm">{result.full_name}</p>
                <p class="text-muted-foreground text-xs">{result.visibility}</p>
              </button>
            {/each}
          </div>
        {/if}

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">仓库名称</p>
            <Input bind:value={linkRepoName} placeholder="my-project" class="h-9 text-sm" />
          </div>
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">默认分支</p>
            <Input bind:value={linkRepoBranch} placeholder="main" class="h-9 text-sm" />
          </div>
        </div>
        <div>
          <p class="text-foreground mb-1 text-xs font-medium">Git URL</p>
          <Input
            bind:value={linkRepoUrl}
            placeholder="https://github.com/owner/repo.git"
            class="h-9 text-sm"
          />
        </div>

        <div class="flex items-center gap-2">
          <Button
            onclick={onLinkRepo}
            disabled={linking || !linkRepoUrl.trim() || !linkRepoName.trim()}
          >
            {#if linking}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              关联中...
            {:else}
              <Link class="mr-1.5 size-3.5" />
              关联仓库
            {/if}
          </Button>
          <Button variant="ghost" size="sm" onclick={() => (mode = 'choose')}>返回</Button>
        </div>
      </div>
    {/if}
  {/if}
</div>
