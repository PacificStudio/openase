<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import Circle from '@lucide/svelte/icons/circle'
  import CircleCheck from '@lucide/svelte/icons/circle-check'
  import CircleX from '@lucide/svelte/icons/circle-x'
  import ExternalLink from '@lucide/svelte/icons/external-link'
  import GitBranch from '@lucide/svelte/icons/git-branch'
  import GitPullRequest from '@lucide/svelte/icons/git-pull-request'
  import Loader from '@lucide/svelte/icons/loader'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import { cn } from '$lib/utils'
  import { repoScopeCiStatusOptions, repoScopePrStatusOptions } from '../mutation-shared'
  import type { TicketDetail } from '../types'

  type ScopeDraft = {
    branchName: string
    pullRequestUrl: string
    prStatus: string
    ciStatus: string
    isPrimaryScope: boolean
  }

  const prStatusConfig: Record<string, { class: string; label: string }> = {
    open: { class: 'text-green-400', label: 'Open' },
    merged: { class: 'text-purple-400', label: 'Merged' },
    closed: { class: 'text-red-400', label: 'Closed' },
    draft: { class: 'text-muted-foreground', label: 'Draft' },
  }

  const ciStatusConfig: Record<string, { icon: typeof CircleCheck; class: string }> = {
    pass: { icon: CircleCheck, class: 'text-green-400' },
    fail: { icon: CircleX, class: 'text-red-400' },
    running: { icon: Loader, class: 'text-yellow-400 animate-spin' },
    pending: { icon: Circle, class: 'text-muted-foreground' },
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
        prStatus: string
        ciStatus: string
        isPrimaryScope: boolean
      },
    ) => void
    onDelete?: (scopeId: string) => void
  } = $props()

  let draft = $derived.by<ScopeDraft>(() => ({
    branchName: scope.branchName,
    pullRequestUrl: scope.prUrl ?? '',
    prStatus: scope.prStatus ?? '',
    ciStatus: scope.ciStatus ?? '',
    isPrimaryScope: scope.isPrimaryScope,
  }))

  function updateDraft(key: keyof ScopeDraft, value: string | boolean) {
    draft = {
      ...draft,
      [key]: value,
    }
  }

  function handleSave() {
    onSave?.(scope.id, draft)
  }
</script>

<div class="border-border bg-muted/20 rounded-lg border p-4">
  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0">
      <div class="text-foreground flex items-center gap-2 text-sm font-medium">
        <GitBranch class="text-muted-foreground size-3.5" />
        <span class="truncate">{scope.repoName}</span>
        {#if scope.isPrimaryScope}
          <Badge variant="outline" class="h-4 py-0 text-[10px]">Primary</Badge>
        {/if}
      </div>

      {#if scope.prUrl}
        <div class="mt-2 flex items-center gap-2 text-[11px]">
          <GitPullRequest
            class={cn(
              'size-3.5',
              prStatusConfig[scope.prStatus ?? 'open']?.class ?? 'text-muted-foreground',
            )}
          />
          <a
            href={scope.prUrl}
            target="_blank"
            rel="noopener noreferrer"
            class="flex items-center gap-1 text-blue-400 hover:underline"
          >
            Pull Request
            <ExternalLink class="size-2.5" />
          </a>
          {#if scope.prStatus}
            <Badge
              variant="outline"
              class={cn('h-4 py-0 text-[10px]', prStatusConfig[scope.prStatus]?.class)}
            >
              {prStatusConfig[scope.prStatus]?.label ?? scope.prStatus}
            </Badge>
          {/if}
          {#if scope.ciStatus}
            {@const ci = ciStatusConfig[scope.ciStatus]}
            {#if ci}
              <ci.icon class={cn('size-3.5', ci.class)} />
            {/if}
          {/if}
        </div>
      {/if}
    </div>

    <Button variant="ghost" size="icon-sm" disabled={deleting} onclick={() => onDelete?.(scope.id)}>
      <Trash2 class="size-3.5" />
    </Button>
  </div>

  <div class="mt-4 grid gap-3">
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

    <label class="flex items-center gap-2 text-xs">
      <input
        type="checkbox"
        checked={draft.isPrimaryScope}
        onchange={(event) => updateDraft('isPrimaryScope', event.currentTarget.checked)}
      />
      <span>Primary scope</span>
    </label>

    <div class="flex justify-end">
      <Button size="sm" disabled={saving} onclick={handleSave}>
        {saving ? 'Saving…' : 'Save scope'}
      </Button>
    </div>
  </div>
</div>
