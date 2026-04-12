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

  const STEP_KEY = 'onboarding.step.repo'
  const KEYS = {
    actionsAddRepository: `${STEP_KEY}.actions.addRepository`,
    createCardTitle: `${STEP_KEY}.createCard.title`,
    createCardDescription: `${STEP_KEY}.createCard.description`,
    linkCardTitle: `${STEP_KEY}.linkCard.title`,
    linkCardDescription: `${STEP_KEY}.linkCard.description`,
    formNamespace: `${STEP_KEY}.forms.namespace`,
    formRepositoryName: `${STEP_KEY}.forms.repositoryName`,
    formVisibility: `${STEP_KEY}.forms.visibility`,
    formDefaultBranch: `${STEP_KEY}.forms.defaultBranch`,
    formGitUrl: `${STEP_KEY}.forms.gitUrl`,
    placeholderRepositoryName: `${STEP_KEY}.placeholders.repositoryName`,
    placeholderDefaultBranch: `${STEP_KEY}.placeholders.defaultBranch`,
    placeholderGitUrl: `${STEP_KEY}.placeholders.gitUrl`,
    placeholderNamespace: `${STEP_KEY}.placeholders.namespace`,
    visibilityPrivate: `${STEP_KEY}.visibility.private`,
    visibilityPublic: `${STEP_KEY}.visibility.public`,
    actionsCreating: `${STEP_KEY}.actions.creating`,
    actionsCreateAndLink: `${STEP_KEY}.actions.createAndLink`,
    actionsLinking: `${STEP_KEY}.actions.linking`,
    actionsLinkRepository: `${STEP_KEY}.actions.linkRepository`,
    actionsBack: `${STEP_KEY}.actions.back`,
    searchHeading: `${STEP_KEY}.search.heading`,
    searchPlaceholder: `${STEP_KEY}.search.placeholder`,
  } as const

  type StepRepoCopyKey = (typeof KEYS)[keyof typeof KEYS]

  const copy: Record<StepRepoCopyKey, string> = {
    [KEYS.actionsAddRepository]: 'Add another repository',
    [KEYS.createCardTitle]: 'Create a new repository',
    [KEYS.createCardDescription]: 'Create a new code repository on GitHub',
    [KEYS.linkCardTitle]: 'Link an existing repository',
    [KEYS.linkCardDescription]: 'Link an existing Git repository',
    [KEYS.formNamespace]: 'Namespace',
    [KEYS.formRepositoryName]: 'Repository name',
    [KEYS.formVisibility]: 'Visibility',
    [KEYS.formDefaultBranch]: 'Default branch',
    [KEYS.formGitUrl]: 'Git URL',
    [KEYS.placeholderRepositoryName]: 'my-project',
    [KEYS.placeholderDefaultBranch]: 'main',
    [KEYS.placeholderGitUrl]: 'https://github.com/owner/repo.git',
    [KEYS.visibilityPrivate]: 'Private',
    [KEYS.visibilityPublic]: 'Public',
    [KEYS.actionsCreating]: 'Creating...',
    [KEYS.actionsCreateAndLink]: 'Create and link',
    [KEYS.actionsLinking]: 'Linking...',
    [KEYS.actionsLinkRepository]: 'Link repository',
    [KEYS.actionsBack]: 'Back',
    [KEYS.searchHeading]: 'Search or browse GitHub repositories',
    [KEYS.searchPlaceholder]: 'Search repository names, or browse recently accessible repositories...',
    [KEYS.placeholderNamespace]: 'Select a namespace',
  }

  function t(key: StepRepoCopyKey) {
    return copy[key] ?? ''
  }
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
        {t(KEYS.actionsAddRepository)}
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
            <p class="text-foreground text-sm font-medium">{t(KEYS.createCardTitle)}</p>
            <p class="text-muted-foreground mt-0.5 text-xs">{t(KEYS.createCardDescription)}</p>
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
            <p class="text-foreground text-sm font-medium">{t(KEYS.linkCardTitle)}</p>
            <p class="text-muted-foreground mt-0.5 text-xs">{t(KEYS.linkCardDescription)}</p>
          </div>
        </button>
      </div>
    {:else if mode === 'create'}
      <div class="space-y-3">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">{t(KEYS.formNamespace)}</p>
            <Select.Root
              type="single"
              value={selectedNamespace}
              onValueChange={(v) => {
                if (v) selectedNamespace = v
              }}
            >
              <Select.Trigger class="h-9 w-full text-sm">
                {selectedNamespace || t(KEYS.placeholderNamespace)}
              </Select.Trigger>
              <Select.Content>
                {#each namespaces as ns (ns.login)}
                  <Select.Item value={ns.login}>{ns.login}</Select.Item>
                {/each}
              </Select.Content>
            </Select.Root>
          </div>
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">{t(KEYS.formRepositoryName)}</p>
            <Input
              bind:value={newRepoName}
              placeholder={t(KEYS.placeholderRepositoryName)}
              class="h-9 text-sm"
            />
          </div>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">{t(KEYS.formVisibility)}</p>
            <Select.Root
              type="single"
              value={newRepoVisibility}
              onValueChange={(v) => {
                if (v) newRepoVisibility = v as 'private' | 'public'
              }}
            >
              <Select.Trigger class="h-9 w-full text-sm">
                {newRepoVisibility === 'private'
                  ? t(KEYS.visibilityPrivate)
                  : t(KEYS.visibilityPublic)}
              </Select.Trigger>
              <Select.Content>
                <Select.Item value="private">
                  {t(KEYS.visibilityPrivate)}
                </Select.Item>
                <Select.Item value="public">
                  {t(KEYS.visibilityPublic)}
                </Select.Item>
              </Select.Content>
            </Select.Root>
          </div>
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">
              {t(KEYS.formDefaultBranch)}
            </p>
            <Input
              bind:value={newRepoDefaultBranch}
              placeholder={t(KEYS.placeholderDefaultBranch)}
              class="h-9 text-sm"
            />
          </div>
        </div>
        <div class="flex items-center gap-2">
          <Button
            onclick={onCreateRepo}
            disabled={creating || !newRepoName.trim() || !selectedNamespace}
          >
              {#if creating}
                <Loader2 class="mr-1.5 size-3.5 animate-spin" />
                {t(KEYS.actionsCreating)}
              {:else}
                <Plus class="mr-1.5 size-3.5" />
                {t(KEYS.actionsCreateAndLink)}
              {/if}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onclick={() => (mode = 'choose')}
          >
            {t(KEYS.actionsBack)}
          </Button>
        </div>
      </div>
    {:else}
      <div class="space-y-3">
        <div>
          <p class="text-foreground mb-1 text-xs font-medium">
            {t(KEYS.searchHeading)}
          </p>
          <div class="flex items-center gap-2">
            <Input
              bind:value={repoSearchQuery}
              placeholder={t(KEYS.searchPlaceholder)}
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
            <p class="text-foreground mb-1 text-xs font-medium">{t(KEYS.formRepositoryName)}</p>
            <Input
              bind:value={linkRepoName}
              placeholder={t(KEYS.placeholderRepositoryName)}
              class="h-9 text-sm"
            />
          </div>
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">{t(KEYS.formDefaultBranch)}</p>
            <Input
              bind:value={linkRepoBranch}
              placeholder={t(KEYS.placeholderDefaultBranch)}
              class="h-9 text-sm"
            />
          </div>
        </div>
        <div>
          <p class="text-foreground mb-1 text-xs font-medium">{t(KEYS.formGitUrl)}</p>
          <Input
            bind:value={linkRepoUrl}
            placeholder={t(KEYS.placeholderGitUrl)}
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
              {t(KEYS.actionsLinking)}
            {:else}
              <Link class="mr-1.5 size-3.5" />
              {t(KEYS.actionsLinkRepository)}
            {/if}
          </Button>
        <Button
          variant="ghost"
          size="sm"
          onclick={() => (mode = 'choose')}
        >
          {t(KEYS.actionsBack)}
        </Button>
        </div>
      </div>
    {/if}
  {/if}
</div>
