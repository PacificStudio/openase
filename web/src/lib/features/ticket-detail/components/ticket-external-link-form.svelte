<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import type { TicketExternalLinkDraft } from '../types'

  let {
    creating = false,
    onCreate,
    onCancel,
  }: {
    creating?: boolean
    onCreate?: (draft: TicketExternalLinkDraft) => Promise<boolean> | boolean
    onCancel?: () => void
  } = $props()

  const typeOptions = [
    { value: 'github_pr', label: 'GitHub PR' },
    { value: 'github_issue', label: 'GitHub Issue' },
    { value: 'gitlab_mr', label: 'GitLab MR' },
    { value: 'gitlab_issue', label: 'GitLab Issue' },
    { value: 'jira_ticket', label: 'Jira Ticket' },
    { value: 'custom', label: 'Custom' },
  ]

  const relationOptions = [
    { value: 'resolves', label: 'Resolves' },
    { value: 'related', label: 'Related' },
    { value: 'caused_by', label: 'Caused by' },
  ]

  const statusOptions = [
    { value: '', label: 'None' },
    { value: 'open', label: 'Open' },
    { value: 'closed', label: 'Closed' },
    { value: 'merged', label: 'Merged' },
    { value: 'draft', label: 'Draft' },
    { value: 'in_progress', label: 'In progress' },
  ]

  const defaultDraft: TicketExternalLinkDraft = {
    type: 'github_pr',
    url: '',
    externalId: '',
    title: '',
    status: '',
    relation: 'resolves',
  }

  let draft = $state<TicketExternalLinkDraft>({ ...defaultDraft })

  // Auto-detect type and externalId when URL changes
  function handleUrlChange(url: string) {
    draft.url = url

    const ghPR = url.match(/github\.com\/([^/]+\/[^/]+)\/pull\/(\d+)/)
    if (ghPR) {
      draft.type = 'github_pr'
      draft.externalId = `PR-${ghPR[2]}`
      return
    }

    const ghIssue = url.match(/github\.com\/([^/]+\/[^/]+)\/issues\/(\d+)/)
    if (ghIssue) {
      draft.type = 'github_issue'
      draft.externalId = `#${ghIssue[2]}`
      return
    }

    const glMR = url.match(/gitlab\.com\/([^/]+\/[^/]+)\/-\/merge_requests\/(\d+)/)
    if (glMR) {
      draft.type = 'gitlab_mr'
      draft.externalId = `!${glMR[2]}`
      return
    }

    const glIssue = url.match(/gitlab\.com\/([^/]+\/[^/]+)\/-\/issues\/(\d+)/)
    if (glIssue) {
      draft.type = 'gitlab_issue'
      draft.externalId = `#${glIssue[2]}`
      return
    }

    if (/atlassian\.net|jira\.com/.test(url)) draft.type = 'jira_ticket'
  }

  async function handleSubmit() {
    const accepted = (await onCreate?.(draft)) ?? false
    if (accepted) {
      draft = {
        ...defaultDraft,
        type: draft.type || defaultDraft.type,
        relation: draft.relation || defaultDraft.relation,
      }
    }
  }

  const typeLabel = $derived(typeOptions.find((o) => o.value === draft.type)?.label ?? draft.type)
  const relationLabel = $derived(
    relationOptions.find((o) => o.value === draft.relation)?.label ?? draft.relation,
  )
  const statusLabel = $derived(statusOptions.find((o) => o.value === draft.status)?.label ?? 'None')

  const canSubmit = $derived(
    draft.url.trim().length > 0 &&
      draft.type &&
      draft.relation &&
      draft.externalId.trim().length > 0,
  )
</script>

<div class="grid gap-4">
  <!-- URL — required -->
  <div class="space-y-1.5">
    <Label for="external-link-url">
      URL <span class="text-destructive">*</span>
    </Label>
    <Input
      id="external-link-url"
      value={draft.url}
      oninput={(e) => handleUrlChange((e.currentTarget as HTMLInputElement).value)}
      type="url"
      placeholder="https://github.com/org/repo/pull/123"
    />
  </div>

  <div class="grid gap-4 sm:grid-cols-2">
    <!-- Type — required -->
    <div class="space-y-1.5">
      <Label>Type <span class="text-destructive">*</span></Label>
      <Select.Root
        type="single"
        value={draft.type}
        onValueChange={(v) => {
          if (v) draft.type = v
        }}
      >
        <Select.Trigger class="w-full">{typeLabel}</Select.Trigger>
        <Select.Content>
          {#each typeOptions as opt (opt.value)}
            <Select.Item value={opt.value}>{opt.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    <!-- Relation — required -->
    <div class="space-y-1.5">
      <Label>Relation <span class="text-destructive">*</span></Label>
      <Select.Root
        type="single"
        value={draft.relation}
        onValueChange={(v) => {
          if (v) draft.relation = v
        }}
      >
        <Select.Trigger class="w-full">{relationLabel}</Select.Trigger>
        <Select.Content>
          {#each relationOptions as opt (opt.value)}
            <Select.Item value={opt.value}>{opt.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  <div class="grid gap-4 sm:grid-cols-2">
    <!-- External ID — required, auto-filled for GitHub -->
    <div class="space-y-1.5">
      <Label for="external-link-id">
        External ID <span class="text-destructive">*</span>
      </Label>
      <Input id="external-link-id" bind:value={draft.externalId} placeholder="PR-123" />
    </div>

    <!-- Status — optional -->
    <div class="space-y-1.5">
      <Label>
        Status
        <span class="text-muted-foreground text-xs font-normal">(optional)</span>
      </Label>
      <Select.Root
        type="single"
        value={draft.status || ''}
        onValueChange={(v) => {
          draft.status = v ?? ''
        }}
      >
        <Select.Trigger class="w-full">{statusLabel}</Select.Trigger>
        <Select.Content>
          {#each statusOptions as opt (opt.value)}
            <Select.Item value={opt.value}>{opt.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  <!-- Title — optional -->
  <div class="space-y-1.5">
    <Label for="external-link-title">
      Title
      <span class="text-muted-foreground text-xs font-normal">(optional)</span>
    </Label>
    <Input
      id="external-link-title"
      bind:value={draft.title}
      placeholder="Short description of this link"
    />
  </div>

  <div class="flex justify-end gap-2">
    {#if onCancel}
      <Button variant="outline" onclick={onCancel} disabled={creating}>Cancel</Button>
    {/if}
    <Button onclick={handleSubmit} disabled={creating || !canSubmit}>
      {creating ? 'Adding…' : 'Add link'}
    </Button>
  </div>
</div>
