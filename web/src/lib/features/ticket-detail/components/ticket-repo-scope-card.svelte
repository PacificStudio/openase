<script lang="ts">
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Ellipsis, Pencil, Trash2 } from '@lucide/svelte'
  import ExternalLink from '@lucide/svelte/icons/external-link'
  import GitBranch from '@lucide/svelte/icons/git-branch'
  import GitPullRequest from '@lucide/svelte/icons/git-pull-request'
  import Copy from '@lucide/svelte/icons/copy'
  import Check from '@lucide/svelte/icons/check'
  import { cn } from '$lib/utils'
  import type { TicketDetail } from '../types'

  type ScopeDraft = {
    branchName: string
    pullRequestUrl: string
  }

  let {
    scope,
    saving = false,
    deleting = false,
    onSave,
    onDelete,
  }: {
    scope: TicketDetail['repoScopes'][number]
    saving?: boolean
    deleting?: boolean
    onSave?: (
      scopeId: string,
      draft: {
        branchName: string
        pullRequestUrl: string
      },
    ) => void
    onDelete?: (scopeId: string) => void
  } = $props()

  let draft = $state<ScopeDraft>({
    branchName: '',
    pullRequestUrl: '',
  })
  let editOpen = $state(false)
  let copiedBranch = $state(false)

  $effect(() => {
    draft.branchName = scope.branchName
    draft.pullRequestUrl = scope.prUrl ?? ''
  })

  function updateDraft(key: keyof ScopeDraft, value: string | boolean) {
    draft = {
      ...draft,
      [key]: value,
    }
  }

  function resetDraft() {
    draft = {
      branchName: scope.branchName,
      pullRequestUrl: scope.prUrl ?? '',
    }
  }

  function handleEdit() {
    resetDraft()
    editOpen = true
  }

  function handleSave() {
    onSave?.(scope.id, draft)
    editOpen = false
  }

  function copyBranch() {
    navigator.clipboard.writeText(scope.effectiveBranchName)
    copiedBranch = true
    setTimeout(() => (copiedBranch = false), 1500)
  }

  function extractPrLabel(url: string) {
    const match = url.match(/\/pull\/(\d+)/)
    return match ? `#${match[1]}` : 'Pull Request'
  }
</script>

<div class="border-border bg-muted/20 rounded-lg border px-3 py-2.5">
  <div class="flex items-center justify-between gap-2">
    <div class="text-foreground flex items-center gap-1.5 text-xs font-medium">
      <GitBranch class="text-muted-foreground size-3.5 shrink-0" />
      <span class="break-words">{scope.repoName}</span>
    </div>

    <DropdownMenu.Root>
      <DropdownMenu.Trigger>
        {#snippet child({ props })}
          <Button variant="ghost" size="icon-xs" {...props}>
            <Ellipsis class="size-3.5" />
            <span class="sr-only">More actions</span>
          </Button>
        {/snippet}
      </DropdownMenu.Trigger>
      <DropdownMenu.Content align="end" class="w-40">
        <DropdownMenu.Item disabled={saving || deleting} onclick={handleEdit}>
          <Pencil class="mr-2 size-3.5" />
          Edit
        </DropdownMenu.Item>
        <DropdownMenu.Item
          class="text-destructive focus:text-destructive"
          disabled={deleting || saving}
          onclick={() => onDelete?.(scope.id)}
        >
          <Trash2 class="mr-2 size-3.5" />
          {deleting ? 'Deleting\u2026' : 'Delete'}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu.Root>
  </div>

  <div class="mt-2 space-y-1.5">
    <!-- Branch -->
    <div class="flex items-center gap-1.5">
      <button
        type="button"
        class="bg-muted hover:bg-muted/80 group inline-flex items-center gap-1 rounded px-1.5 py-0.5 font-mono text-[11px] transition-colors"
        title="Copy branch name"
        onclick={copyBranch}
      >
        <span class="text-foreground">{scope.effectiveBranchName}</span>
        {#if copiedBranch}
          <Check class="size-2.5 text-emerald-500" />
        {:else}
          <Copy
            class="text-muted-foreground/50 size-2.5 opacity-0 transition-opacity group-hover:opacity-100"
          />
        {/if}
      </button>
      {#if scope.branchSource === 'override'}
        <span class="text-muted-foreground text-[9px] tracking-wide uppercase">override</span>
      {/if}
    </div>

    <!-- Base branch -->
    <div class="text-muted-foreground flex items-center gap-1 text-[11px]">
      <span>base:</span>
      <code class="text-foreground/70 font-mono">{scope.defaultBranch}</code>
    </div>

    <!-- PR -->
    <div class="flex items-center gap-1.5 text-[11px]">
      <GitPullRequest
        class={cn('size-3 shrink-0', scope.prUrl ? 'text-emerald-500' : 'text-muted-foreground/40')}
      />
      {#if scope.prUrl}
        <a
          href={scope.prUrl}
          target="_blank"
          rel="noopener noreferrer"
          class="text-foreground inline-flex items-center gap-1 font-medium transition-colors hover:text-blue-400 hover:underline"
        >
          {extractPrLabel(scope.prUrl)}
          <ExternalLink class="size-2.5" />
        </a>
      {:else}
        <span class="text-muted-foreground/50">No PR</span>
      {/if}
    </div>
  </div>
</div>

<Dialog.Root bind:open={editOpen}>
  <Dialog.Content class="sm:max-w-xl">
    <Dialog.Header>
      <Dialog.Title>Edit repo scope</Dialog.Title>
      <Dialog.Description>
        Update the optional work branch override and PR link for {scope.repoName}.
      </Dialog.Description>
    </Dialog.Header>

    <div class="grid gap-3 py-4">
      <div class="space-y-2">
        <Label for={`scope-branch-${scope.id}`}>Work branch override</Label>
        <Input
          id={`scope-branch-${scope.id}`}
          value={draft.branchName}
          placeholder="Leave blank to use the generated ticket branch"
          oninput={(event) => updateDraft('branchName', event.currentTarget.value)}
        />
        <p class="text-muted-foreground text-xs">Base branch: {scope.defaultBranch}</p>
      </div>

      <div class="space-y-2">
        <Label for={`scope-pr-url-${scope.id}`}>Pull request URL</Label>
        <Input
          id={`scope-pr-url-${scope.id}`}
          value={draft.pullRequestUrl}
          placeholder="https://..."
          oninput={(event) => updateDraft('pullRequestUrl', event.currentTarget.value)}
        />
      </div>
    </div>

    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={saving}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button disabled={saving} onclick={handleSave}>
        {saving ? 'Saving\u2026' : 'Save'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
