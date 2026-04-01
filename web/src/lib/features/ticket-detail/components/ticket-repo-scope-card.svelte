<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import Circle from '@lucide/svelte/icons/circle'
  import Pencil from '@lucide/svelte/icons/pencil'
  import ExternalLink from '@lucide/svelte/icons/external-link'
  import GitBranch from '@lucide/svelte/icons/git-branch'
  import GitPullRequest from '@lucide/svelte/icons/git-pull-request'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import { cn } from '$lib/utils'
  import { repoScopeCiStatusOptions, repoScopePrStatusOptions } from '../mutation-shared'
  import type { TicketDetail } from '../types'
  import {
    ciStatusConfig,
    getCiStatusLabel,
    getPrStatusLabel,
    prStatusConfig,
  } from './ticket-repo-scope-card.shared'

  type ScopeDraft = {
    branchName: string
    pullRequestUrl: string
    prStatus: string
    ciStatus: string
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
        prStatus: string
        ciStatus: string
      },
    ) => void
    onDelete?: (scopeId: string) => void
  } = $props()

  let draft = $state<ScopeDraft>({
    branchName: '',
    pullRequestUrl: '',
    prStatus: '',
    ciStatus: '',
  })
  let editing = $state(false)

  $effect(() => {
    draft.branchName = scope.branchName
    draft.pullRequestUrl = scope.prUrl ?? ''
    draft.prStatus = scope.prStatus ?? ''
    draft.ciStatus = scope.ciStatus ?? ''
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
      prStatus: scope.prStatus ?? '',
      ciStatus: scope.ciStatus ?? '',
    }
  }

  function handleEdit() {
    resetDraft()
    editing = true
  }

  function handleCancel() {
    resetDraft()
    editing = false
  }

  function handleSave() {
    onSave?.(scope.id, draft)
    editing = false
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

    <div class="flex items-center gap-1">
      {#if !editing}
        <Button
          variant="outline"
          size="sm"
          disabled={saving || deleting}
          aria-label={`Edit ${scope.repoName} scope`}
          onclick={handleEdit}
        >
          <Pencil class="size-3.5" />
          Edit
        </Button>
      {/if}

      <Button
        variant="ghost"
        size="icon-sm"
        disabled={deleting || saving}
        aria-label={`Delete ${scope.repoName} scope`}
        onclick={() => onDelete?.(scope.id)}
      >
        <Trash2 class="size-3.5" />
      </Button>
    </div>
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
          <GitPullRequest
            class={cn(
              'size-3.5 shrink-0',
              prStatusConfig[scope.prStatus ?? 'open']?.class ?? 'text-muted-foreground',
            )}
          />
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
          <Badge
            variant="outline"
            class={cn(
              'h-4 shrink-0 py-0 text-[10px]',
              scope.prStatus ? prStatusConfig[scope.prStatus]?.class : 'text-muted-foreground',
            )}
          >
            {getPrStatusLabel(scope.prStatus)}
          </Badge>
        </div>
      </dd>
    </div>

    <div class="flex items-start gap-2">
      <dt class={cn(summaryLabelClass, 'w-16 shrink-0 pt-0.5')}>CI</dt>
      <dd class="text-foreground flex items-center gap-1.5 text-xs">
        {#if scope.ciStatus}
          {@const ci = ciStatusConfig[scope.ciStatus]}
          {#if ci}
            <ci.icon class={cn('size-3.5 shrink-0', ci.class)} />
          {/if}
        {:else}
          <Circle class="text-muted-foreground size-3.5 shrink-0" />
        {/if}
        <span>{getCiStatusLabel(scope.ciStatus)}</span>
      </dd>
    </div>
  </dl>

  {#if editing}
    <div class="mt-4 grid gap-3 border-t pt-4">
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

      <div class="grid gap-3 sm:grid-cols-2">
        <div class="space-y-2">
          <Label>PR status</Label>
          <Select.Root
            type="single"
            value={draft.prStatus}
            onValueChange={(value) => {
              updateDraft('prStatus', value || '')
            }}
          >
            <Select.Trigger class="w-full">
              {repoScopePrStatusOptions.find((option) => option.value === draft.prStatus)?.label ??
                'Unset'}
            </Select.Trigger>
            <Select.Content>
              {#each repoScopePrStatusOptions as option (option.value)}
                <Select.Item value={option.value}>{option.label}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label>CI status</Label>
          <Select.Root
            type="single"
            value={draft.ciStatus}
            onValueChange={(value) => {
              updateDraft('ciStatus', value || '')
            }}
          >
            <Select.Trigger class="w-full">
              {repoScopeCiStatusOptions.find((option) => option.value === draft.ciStatus)?.label ??
                'Unset'}
            </Select.Trigger>
            <Select.Content>
              {#each repoScopeCiStatusOptions as option (option.value)}
                <Select.Item value={option.value}>{option.label}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>
      </div>

      <div class="flex justify-end gap-2">
        <Button variant="outline" size="sm" disabled={saving} onclick={handleCancel}>Cancel</Button>
        <Button size="sm" disabled={saving} onclick={handleSave}>
          {saving ? 'Saving…' : 'Save scope'}
        </Button>
      </div>
    </div>
  {/if}
</div>
