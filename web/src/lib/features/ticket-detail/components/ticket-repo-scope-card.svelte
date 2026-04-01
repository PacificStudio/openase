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
  import { cn } from '$lib/utils'
  import type { TicketDetail } from '../types'

  type ScopeDraft = {
    branchName: string
    pullRequestUrl: string
  }

  const summaryLabelClass = 'text-muted-foreground text-[10px] font-medium uppercase tracking-wide'
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
</script>

<div class="border-border bg-muted/20 rounded-lg border p-4">
  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0">
      <div class="text-foreground flex items-center gap-2 text-sm font-medium">
        <GitBranch class="text-muted-foreground size-3.5" />
        <span class="break-words">{scope.repoName}</span>
      </div>
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

  <dl class="mt-3 space-y-2.5">
    <div class="flex items-start gap-2">
      <dt class={cn(summaryLabelClass, 'w-16 shrink-0 pt-0.5')}>Branch</dt>
      <dd class="text-foreground flex min-w-0 items-center gap-1.5 text-xs">
        <GitBranch class="text-muted-foreground size-3.5 shrink-0" />
        <span class="break-all">{scope.branchName || 'Unset'}</span>
      </dd>
    </div>

    <div class="flex items-start gap-2">
      <dt class={cn(summaryLabelClass, 'w-16 shrink-0 pt-0.5')}>PR</dt>
      <dd class="min-w-0 text-xs">
        <div class="flex items-center gap-1.5">
          <GitPullRequest class="text-muted-foreground size-3.5 shrink-0" />
          {#if scope.prUrl}
            <a
              href={scope.prUrl}
              target="_blank"
              rel="noopener noreferrer"
              class="text-blue-400 hover:underline"
            >
              Pull Request
            </a>
            <ExternalLink class="text-muted-foreground size-2.5 shrink-0" />
          {:else}
            <span class="text-muted-foreground">No PR linked</span>
          {/if}
        </div>
      </dd>
    </div>
  </dl>
</div>

<Dialog.Root bind:open={editOpen}>
  <Dialog.Content class="sm:max-w-xl">
    <Dialog.Header>
      <Dialog.Title>Edit repo scope</Dialog.Title>
      <Dialog.Description>Update the branch and PR link for {scope.repoName}.</Dialog.Description>
    </Dialog.Header>

    <div class="grid gap-3 py-4">
      <div class="space-y-2">
        <Label for={`scope-branch-${scope.id}`}>Branch</Label>
        <Input
          id={`scope-branch-${scope.id}`}
          value={draft.branchName}
          oninput={(event) => updateDraft('branchName', event.currentTarget.value)}
        />
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
