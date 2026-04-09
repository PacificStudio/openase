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

  const statusOptions = [
    { value: '', label: 'None' },
    { value: 'open', label: 'Open' },
    { value: 'closed', label: 'Closed' },
    { value: 'merged', label: 'Merged' },
    { value: 'draft', label: 'Draft' },
    { value: 'in_progress', label: 'In progress' },
  ]

  const defaultDraft: TicketExternalLinkDraft = {
    type: '',
    url: '',
    externalId: '',
    title: '',
    status: '',
  }

  let draft = $state<TicketExternalLinkDraft>({ ...defaultDraft })

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
      draft = { ...defaultDraft }
    }
  }

  const statusLabel = $derived(statusOptions.find((o) => o.value === draft.status)?.label ?? 'None')
  const canSubmit = $derived(draft.url.trim().length > 0 && draft.externalId.trim().length > 0)
</script>

<div class="grid gap-4">
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
    <div class="space-y-1.5">
      <Label for="external-link-type">
        Type
        <span class="text-muted-foreground text-xs font-normal">(optional)</span>
      </Label>
      <Input
        id="external-link-type"
        bind:value={draft.type}
        placeholder="github_pr, jira_ticket, doc, spec"
      />
    </div>

    <div class="space-y-1.5">
      <Label for="external-link-id">
        External ID <span class="text-destructive">*</span>
      </Label>
      <Input id="external-link-id" bind:value={draft.externalId} placeholder="PR-123" />
    </div>
  </div>

  <div class="grid gap-4 sm:grid-cols-2">
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
  </div>

  <p class="text-muted-foreground text-xs">
    Type is optional freeform text. Leave it blank to use the backend default semantics.
  </p>

  <div class="flex justify-end gap-2">
    {#if onCancel}
      <Button variant="outline" onclick={onCancel} disabled={creating}>Cancel</Button>
    {/if}
    <Button onclick={handleSubmit} disabled={creating || !canSubmit}>
      {creating ? 'Adding…' : 'Add link'}
    </Button>
  </div>
</div>
