<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as InputGroup from '$ui/input-group'
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

  let urlType = $state<UrlType>(draft.repositoryURL.startsWith('file://') ? 'file' : 'remote')

  // Path portion shown in the input: strip file:// prefix and .git suffix
  function extractFilePath(url: string): string {
    let p = url.startsWith('file://') ? url.slice('file://'.length) : ''
    if (p.endsWith('.git')) p = p.slice(0, -4)
    return p
  }

  function buildFileUrl(path: string): string {
    return 'file://' + path.replace(/\/+$/, '') + '.git'
  }

  let filePath = $state(extractFilePath(draft.repositoryURL))

  function update(field: keyof RepositoryDraft, event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange?.(field, target.value)
  }

  function onFilePathInput(event: Event) {
    const path = (event.currentTarget as HTMLInputElement).value
    filePath = path
    onDraftChange?.('repositoryURL', buildFileUrl(path))
  }

  function switchUrlType(type: UrlType) {
    if (urlType === type) return
    urlType = type
    if (type === 'file') {
      filePath = ''
      onDraftChange?.('repositoryURL', buildFileUrl(''))
    } else {
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

    {#if urlType === 'file'}
      <InputGroup.Root>
        <InputGroup.Addon align="inline-start">
          <InputGroup.Text class="text-xs font-mono">file://</InputGroup.Text>
        </InputGroup.Addon>
        <InputGroup.Input
          id="repo-url"
          value={filePath}
          placeholder="/home/user/repos/backend"
          class="text-sm"
          oninput={onFilePathInput}
        />
        <InputGroup.Addon align="inline-end">
          <InputGroup.Text class="text-xs font-mono">.git</InputGroup.Text>
        </InputGroup.Addon>
      </InputGroup.Root>
      <div class="bg-amber-50 border border-amber-200 rounded-md px-2.5 py-2 text-[11px] text-amber-800 dark:bg-amber-950/30 dark:border-amber-800/50 dark:text-amber-400">
        <span class="font-semibold">Beta</span> — Local path support is experimental. In multi-machine
        setups the path must exist on every machine that runs clone or fetch operations, otherwise the
        agent will fail to prepare the workspace.
      </div>
    {:else}
      <Input
        id="repo-url"
        value={draft.repositoryURL}
        placeholder="https://github.com/acme/backend.git"
        class="h-9 text-sm"
        oninput={(event) => update('repositoryURL', event)}
      />
      <p class="text-muted-foreground text-[11px]">
        Supports <code class="font-mono">https://</code> and <code class="font-mono">git@</code>
        SSH URLs. Works with GitHub, GitLab, Gitea, and any self-hosted Git server.
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
