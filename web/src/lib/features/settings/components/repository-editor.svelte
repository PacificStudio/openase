<script lang="ts">
  import { untrack } from 'svelte'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { cn } from '$lib/utils'
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { i18nStore } from '$lib/i18n/store.svelte'
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

  function detectUrlType(repositoryURL: string): UrlType {
    return repositoryURL.startsWith('file://') ? 'file' : 'remote'
  }

  let urlType = $state<UrlType>(untrack(() => detectUrlType(draft.repositoryURL)))
  let lastRepositoryURL = $state(untrack(() => draft.repositoryURL))

  $effect(() => {
    const nextRepositoryURL = draft.repositoryURL
    if (nextRepositoryURL === lastRepositoryURL) {
      return
    }

    lastRepositoryURL = nextRepositoryURL
    urlType = detectUrlType(nextRepositoryURL)
  })

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
          ? i18nStore.t('settings.repositoryEditor.heading.edit', {
              name: selectedRepo?.name ?? i18nStore.t('settings.repositoryEditor.labels.boundRepo'),
            })
          : i18nStore.t('settings.repositoryEditor.heading.create')}
      </h3>
      <p class="text-muted-foreground mt-1 text-xs">
        {i18nStore.t('settings.repositoryEditor.hints.identityDescription')}
      </p>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label for="repo-name">{i18nStore.t('settings.repositoryEditor.labels.name')}</Label>
        <Input
          id="repo-name"
          value={draft.name}
          placeholder={i18nStore.t('settings.repositoryEditor.placeholders.nameExample')}
          oninput={(event) => updateTextField('name', event)}
        />
      </div>

      <div class="space-y-2">
        <Label for="repo-default-branch">{i18nStore.t('settings.repositoryEditor.labels.defaultBranch')}</Label>
        <Input
          id="repo-default-branch"
          value={draft.defaultBranch}
          placeholder={i18nStore.t('settings.repositoryEditor.placeholders.defaultBranch')}
          oninput={(event) => updateTextField('defaultBranch', event)}
        />
        <p class="text-muted-foreground text-xs">
          {i18nStore.t('settings.repositoryEditor.hints.defaultBranchNormalization')}
        </p>
      </div>
    </div>

    <div class="space-y-2">
      <Label for="repo-url">{i18nStore.t('settings.repositoryEditor.labels.repositoryUrl')}</Label>
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
          {i18nStore.t('settings.repositoryEditor.buttons.remote')}
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
          {i18nStore.t('settings.repositoryEditor.buttons.localPath')}
        </button>
      </div>
      <Input
        id="repo-url"
        value={draft.repositoryURL}
        placeholder={
          urlType === 'file'
            ? i18nStore.t('settings.repositoryEditor.placeholders.fileUrl')
            : i18nStore.t('settings.repositoryEditor.placeholders.remoteUrl')
        }
        oninput={(event) => updateTextField('repositoryURL', event)}
      />
      {#if urlType === 'file'}
        <p class="text-muted-foreground text-xs">
          {i18nStore.t('settings.repositoryEditor.hints.localRepoIntro')}{' '}
          <code class="font-mono text-[11px]">file:///srv/git/backend.git</code>
        </p>
      {:else}
        <p class="text-muted-foreground text-xs">
          {i18nStore.t('settings.repositoryEditor.hints.remoteRepoSupports')}
          <code class="font-mono text-[11px]">https://</code>
          {i18nStore.t('settings.repositoryEditor.hints.remoteRepoSupportsAnd')}
          <code class="font-mono text-[11px]">git@</code>
          {i18nStore.t('settings.repositoryEditor.hints.remoteRepoSupportsRest')}
        </p>
      {/if}
    </div>
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">
        {i18nStore.t('settings.repositoryEditor.heading.workspaceMapping')}
      </h3>
      <p class="text-muted-foreground mt-1 text-xs">
        {i18nStore.t('settings.repositoryEditor.hints.workspaceMapping')}
      </p>
    </div>

    <div class="space-y-2">
      <Label for="repo-workspace-dirname">
        {i18nStore.t('settings.repositoryEditor.labels.workspaceDirname')}
      </Label>
      <Input
        id="repo-workspace-dirname"
        value={draft.workspaceDirname}
        placeholder={i18nStore.t('settings.repositoryEditor.placeholders.workspaceDirname')}
        oninput={(event) => updateTextField('workspaceDirname', event)}
      />
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('settings.repositoryEditor.hints.workspaceDirname')}
      </p>
    </div>
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">
        {i18nStore.t('settings.repositoryEditor.heading.metadata')}
      </h3>
      <p class="text-muted-foreground mt-1 text-xs">
        {i18nStore.t('settings.repositoryEditor.hints.metadataDescription', { count: reposCount })}
      </p>
    </div>

    <div class="space-y-2">
      <Label for="repo-labels">{i18nStore.t('settings.repositoryEditor.labels.labels')}</Label>
      <Textarea
        id="repo-labels"
        rows={4}
        value={draft.labels}
        placeholder={i18nStore.t('settings.repositoryEditor.placeholders.labels')}
        oninput={(event) => updateTextField('labels', event)}
      />
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('settings.repositoryEditor.hints.labelsDescription')}
      </p>
    </div>
  </section>
</div>
