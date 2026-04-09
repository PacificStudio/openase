<script lang="ts">
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { cn } from '$lib/utils'
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import RepositoryGitHubBrowser from './repository-github-browser.svelte'
  import RepositoryGitHubCreate from './repository-github-create.svelte'
  import type {
    GitHubRepositoryCreateDraft,
    GitHubRepositoryNamespace,
    GitHubRepositoryRecord,
    RepositoryDraft,
    RepositoryEditorMode,
  } from '../repositories-model'

  let {
    mode,
    selectedRepo,
    draft,
    reposCount = 0,
    saving = false,
    githubRepos = [],
    githubRepoQuery = '',
    githubReposLoading = false,
    githubReposLoadingMore = false,
    githubReposNextCursor = '',
    githubRepoError = '',
    githubBindingRepoFullName = '',
    githubNamespaces = [],
    githubNamespacesLoading = false,
    githubCreateDraft,
    githubCreating = false,
    onDraftChange,
    onGitHubRepoQueryChange,
    onGitHubRepoSearch,
    onGitHubRepoLoadMore,
    onBindGitHubRepo,
    onGitHubCreateDraftChange,
    onCreateGitHubRepoAndBind,
  }: {
    mode: RepositoryEditorMode
    selectedRepo: ProjectRepoRecord | null
    draft: RepositoryDraft
    reposCount?: number
    saving?: boolean
    githubRepos?: GitHubRepositoryRecord[]
    githubRepoQuery?: string
    githubReposLoading?: boolean
    githubReposLoadingMore?: boolean
    githubReposNextCursor?: string
    githubRepoError?: string
    githubBindingRepoFullName?: string
    githubNamespaces?: GitHubRepositoryNamespace[]
    githubNamespacesLoading?: boolean
    githubCreateDraft: GitHubRepositoryCreateDraft
    githubCreating?: boolean
    onDraftChange?: (field: keyof RepositoryDraft, value: string | boolean) => void
    onGitHubRepoQueryChange?: (value: string) => void
    onGitHubRepoSearch?: () => void
    onGitHubRepoLoadMore?: () => void
    onBindGitHubRepo?: (repo: GitHubRepositoryRecord) => void
    onGitHubCreateDraftChange?: (field: keyof GitHubRepositoryCreateDraft, value: string) => void
    onCreateGitHubRepoAndBind?: () => void
  } = $props()

  type UrlType = 'remote' | 'file'

  let urlType = $state<UrlType>(draft.repositoryURL.startsWith('file://') ? 'file' : 'remote')

  function updateTextField(field: keyof RepositoryDraft, event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange?.(field, target.value)
  }

  function switchUrlType(type: UrlType) {
    if (urlType === type) return
    urlType = type
    const current = draft.repositoryURL.trim()
    if (type === 'file' && (!current || current === 'https://' || current === 'git@')) {
      onDraftChange?.('repositoryURL', 'file://')
    } else if (type === 'remote' && (!current || current === 'file://')) {
      onDraftChange?.('repositoryURL', '')
    }
  }
</script>

<div class="space-y-5" aria-busy={saving}>
  {#if mode === 'create'}
    <RepositoryGitHubBrowser
      repos={githubRepos}
      query={githubRepoQuery}
      loading={githubReposLoading}
      loadingMore={githubReposLoadingMore}
      nextCursor={githubReposNextCursor}
      error={githubRepoError}
      bindingRepoFullName={githubBindingRepoFullName}
      onQueryChange={onGitHubRepoQueryChange}
      onSearch={onGitHubRepoSearch}
      onLoadMore={onGitHubRepoLoadMore}
      onBind={onBindGitHubRepo}
    />

    <RepositoryGitHubCreate
      namespaces={githubNamespaces}
      draft={githubCreateDraft}
      loadingNamespaces={githubNamespacesLoading}
      creating={githubCreating}
      onDraftChange={onGitHubCreateDraftChange}
      onCreate={onCreateGitHubRepoAndBind}
    />
  {/if}

  <section class="space-y-4">
    <div>
      <h3 class="text-foreground text-sm font-semibold">
        {mode === 'edit'
          ? `Repository identity · ${selectedRepo?.name ?? 'bound repo'}`
          : 'Manual repository binding'}
      </h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Basic Git coordinates that OpenASE uses for repo scopes and workspace preparation.
      </p>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label for="repo-name">Name</Label>
        <Input
          id="repo-name"
          value={draft.name}
          placeholder="backend"
          oninput={(event) => updateTextField('name', event)}
        />
      </div>

      <div class="space-y-2">
        <Label for="repo-default-branch">Default branch</Label>
        <Input
          id="repo-default-branch"
          value={draft.defaultBranch}
          placeholder="main"
          oninput={(event) => updateTextField('defaultBranch', event)}
        />
        <p class="text-muted-foreground text-xs">Blank input normalizes to `main`.</p>
      </div>
    </div>

    <div class="space-y-2">
      <Label for="repo-url">Repository URL</Label>
      <div class="bg-muted flex rounded-md p-0.5 text-xs">
        <button
          type="button"
          class={cn(
            'flex-1 rounded px-3 py-1.5 font-medium transition-colors',
            urlType === 'remote'
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground',
          )}
          onclick={() => switchUrlType('remote')}
        >
          Remote
        </button>
        <button
          type="button"
          class={cn(
            'flex-1 rounded px-3 py-1.5 font-medium transition-colors',
            urlType === 'file'
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground',
          )}
          onclick={() => switchUrlType('file')}
        >
          Local path
        </button>
      </div>
      <Input
        id="repo-url"
        value={draft.repositoryURL}
        placeholder={urlType === 'file'
          ? 'file:///home/user/repos/backend.git'
          : 'https://github.com/acme/backend.git'}
        oninput={(event) => updateTextField('repositoryURL', event)}
      />
      {#if urlType === 'file'}
        <p class="text-muted-foreground text-xs">
          Uses a local Git repository on the machine running the agent. The path must be accessible
          from that machine at clone/fetch time. Example: <code class="font-mono text-[11px]"
            >file:///srv/git/backend.git</code
          >
        </p>
      {:else}
        <p class="text-muted-foreground text-xs">
          Supports <code class="font-mono text-[11px]">https://</code> and
          <code class="font-mono text-[11px]">git@</code> SSH URLs. Works with GitHub, GitLab, Gitea,
          and any hosted or self-hosted Git server reachable from the agent machine.
        </p>
      {/if}
    </div>
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Workspace mapping</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Optional checkout path that runtime tasks can use for repo-specific workspaces.
      </p>
    </div>

    <div class="space-y-2">
      <Label for="repo-workspace-dirname">Workspace dirname</Label>
      <Input
        id="repo-workspace-dirname"
        value={draft.workspaceDirname}
        placeholder="services/backend"
        oninput={(event) => updateTextField('workspaceDirname', event)}
      />
      <p class="text-muted-foreground text-xs">
        Leave empty to let the runtime derive the workspace path automatically.
      </p>
    </div>
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Metadata</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Labels help group repositories for workflows and ticket repo scopes across {reposCount}
        bound repos.
      </p>
    </div>

    <div class="space-y-2">
      <Label for="repo-labels">Labels</Label>
      <Textarea
        id="repo-labels"
        rows={4}
        value={draft.labels}
        placeholder={`go, backend, api\nworker`}
        oninput={(event) => updateTextField('labels', event)}
      />
      <p class="text-muted-foreground text-xs">
        Separate labels with commas or new lines. Empty entries are removed and duplicates are
        collapsed.
      </p>
    </div>
  </section>
</div>
