<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import type { TranslationKey } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
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

  const statusOptions: Array<{ value: string; labelKey: TranslationKey }> = [
    { value: '', labelKey: 'ticketDetail.externalLink.status.none' },
    { value: 'open', labelKey: 'ticketDetail.externalLink.status.open' },
    { value: 'closed', labelKey: 'ticketDetail.externalLink.status.closed' },
    { value: 'merged', labelKey: 'ticketDetail.externalLink.status.merged' },
    { value: 'draft', labelKey: 'ticketDetail.externalLink.status.draft' },
    { value: 'in_progress', labelKey: 'ticketDetail.externalLink.status.inProgress' },
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

  const defaultStatusLabelKey: TranslationKey = 'ticketDetail.externalLink.status.none'
  const statusLabel = $derived(() => {
    const option = statusOptions.find((o) => o.value === draft.status)
    return i18nStore.t(option?.labelKey ?? defaultStatusLabelKey)
  })
  const canSubmit = $derived(draft.url.trim().length > 0 && draft.externalId.trim().length > 0)
</script>

<div class="grid gap-4">
  <div class="space-y-1.5">
    <Label for="external-link-url">
      {i18nStore.t('ticketDetail.externalLink.form.url')}
      <span class="text-destructive">*</span>
    </Label>
    <Input
      id="external-link-url"
      value={draft.url}
      oninput={(e) => handleUrlChange((e.currentTarget as HTMLInputElement).value)}
      type="url"
      placeholder={i18nStore.t('ticketDetail.externalLink.form.urlPlaceholder')}
    />
  </div>

  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label for="external-link-type">
        {i18nStore.t('ticketDetail.externalLink.form.type')}
        <span class="text-muted-foreground text-xs font-normal">
          {i18nStore.t('ticketDetail.externalLink.form.optional')}
        </span>
      </Label>
      <Input
        id="external-link-type"
        bind:value={draft.type}
        placeholder={i18nStore.t('ticketDetail.externalLink.form.typePlaceholder')}
      />
    </div>

    <div class="space-y-1.5">
      <Label for="external-link-id">
        {i18nStore.t('ticketDetail.externalLink.form.externalId')}
        <span class="text-destructive">*</span>
      </Label>
      <Input
        id="external-link-id"
        bind:value={draft.externalId}
        placeholder={i18nStore.t('ticketDetail.externalLink.form.externalIdPlaceholder')}
      />
    </div>
  </div>

  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label>
        {i18nStore.t('ticketDetail.externalLink.form.status')}
        <span class="text-muted-foreground text-xs font-normal">
          {i18nStore.t('ticketDetail.externalLink.form.optional')}
        </span>
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
            <Select.Item value={opt.value}>{i18nStore.t(opt.labelKey)}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    <div class="space-y-1.5">
      <Label for="external-link-title">
        {i18nStore.t('ticketDetail.externalLink.form.title')}
        <span class="text-muted-foreground text-xs font-normal">
          {i18nStore.t('ticketDetail.externalLink.form.optional')}
        </span>
      </Label>
      <Input
        id="external-link-title"
        bind:value={draft.title}
        placeholder={i18nStore.t('ticketDetail.externalLink.form.titlePlaceholder')}
      />
    </div>
  </div>

  <p class="text-muted-foreground text-xs">
    {i18nStore.t('ticketDetail.externalLink.form.description')}
  </p>

  <div class="flex justify-end gap-2">
    {#if onCancel}
      <Button variant="outline" onclick={onCancel} disabled={creating}>
        {i18nStore.t('ticketDetail.externalLink.form.actions.cancel')}
      </Button>
    {/if}
    <Button onclick={handleSubmit} disabled={creating || !canSubmit}>
      {creating
        ? i18nStore.t('ticketDetail.externalLink.form.actions.adding')
        : i18nStore.t('ticketDetail.externalLink.form.actions.add')}
    </Button>
  </div>
</div>
