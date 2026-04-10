<script lang="ts">
  import { untrack } from 'svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { ChevronDown } from '@lucide/svelte'
  import { cn } from '$lib/utils'
  import type { RepositoryDraft } from '../repositories-model'

  let {
    draft,
    compact = false,
    onDraftChange,
  }: {
    draft: RepositoryDraft
    compact?: boolean
    onDraftChange?: (field: keyof RepositoryDraft, value: string | boolean) => void
  } = $props()

  let showAdvanced = $state(false)

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

  function update(field: keyof RepositoryDraft, event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange?.(field, target.value)
  }

  function switchUrlType(type: UrlType) {
    if (urlType === type) return
    urlType = type
    // Pre-fill with scheme hint when field is empty or only contains the other scheme hint
    const current = draft.repositoryURL.trim()
    if (type === 'file' && (!current || current === 'https://' || current === 'git@')) {
      onDraftChange?.('repositoryURL', 'file://')
    } else if (type === 'remote' && (!current || current === 'file://')) {
      onDraftChange?.('repositoryURL', '')
    }
  }
</script>

<div class="space-y-4" data-testid="repository-editor-form">
  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label for="repo-name" class="text-xs">Name</Label>
      <Input
        id="repo-name"
        value={draft.name}
        placeholder="backend"
        class="h-9 text-sm"
        oninput={(event) => update('name', event)}
      />
    </div>
    <div class="space-y-1.5">
      <Label for="repo-default-branch" class="text-xs">Default branch</Label>
      <Input
        id="repo-default-branch"
        value={draft.defaultBranch}
        placeholder="main"
        class="h-9 text-sm"
        oninput={(event) => update('defaultBranch', event)}
      />
    </div>
  </div>

  <div class="space-y-1.5">
    <Label class="text-xs">Repository URL</Label>
    <div class="bg-muted flex rounded-md p-0.5 text-xs">
      <button
        type="button"
        class={cn(
          'flex-1 rounded px-3 py-1 font-medium transition-colors',
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
          'flex-1 rounded px-3 py-1 font-medium transition-colors',
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
      class="h-9 text-sm"
      oninput={(event) => update('repositoryURL', event)}
    />
    {#if urlType === 'file'}
      <p class="text-muted-foreground text-[11px]">
        Uses a local Git repository on the machine running the agent. The path must be accessible
        from that machine at clone/fetch time.
      </p>
    {:else}
      <p class="text-muted-foreground text-[11px]">
        Supports <code class="font-mono">https://</code> and <code class="font-mono">git@</code>
        SSH URLs. Works with GitHub, GitLab, Gitea, and any hosted or self-hosted Git server.
      </p>
    {/if}
  </div>

  {#if !compact}
    <div class="space-y-1.5">
      <Label for="repo-labels" class="text-xs">Labels</Label>
      <Textarea
        id="repo-labels"
        rows={2}
        value={draft.labels}
        placeholder="go, backend, api"
        class="text-sm"
        oninput={(event) => update('labels', event)}
      />
    </div>

    <div>
      <Button
        variant="ghost"
        size="sm"
        class="text-muted-foreground hover:text-foreground h-7 gap-1 px-0 text-xs"
        onclick={() => (showAdvanced = !showAdvanced)}
      >
        Advanced
        <ChevronDown class="size-3 transition-transform {showAdvanced ? 'rotate-180' : ''}" />
      </Button>

      {#if showAdvanced}
        <div class="mt-2 space-y-1.5">
          <Label for="repo-workspace-dirname" class="text-xs">Workspace path</Label>
          <Input
            id="repo-workspace-dirname"
            value={draft.workspaceDirname}
            placeholder="Auto-derived from repo name"
            class="h-9 text-sm"
            oninput={(event) => update('workspaceDirname', event)}
          />
          <p class="text-muted-foreground text-[11px]">
            Override the checkout subdirectory for monorepo setups. Leave empty for default.
          </p>
        </div>
      {/if}
    </div>
  {/if}
</div>
