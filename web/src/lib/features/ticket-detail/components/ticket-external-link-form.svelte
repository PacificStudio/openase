<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
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

  const defaultDraft: TicketExternalLinkDraft = {
    type: 'github',
    url: '',
    externalId: '',
    title: '',
    status: '',
    relation: 'references',
  }

  let draft = $state<TicketExternalLinkDraft>({ ...defaultDraft })

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
</script>

<div class="grid gap-3 md:grid-cols-2">
  <div class="space-y-2">
    <Label for="external-link-type">Type</Label>
    <Input id="external-link-type" bind:value={draft.type} placeholder="github" />
  </div>
  <div class="space-y-2">
    <Label for="external-link-id">External ID</Label>
    <Input id="external-link-id" bind:value={draft.externalId} placeholder="PR-482" />
  </div>
  <div class="space-y-2 md:col-span-2">
    <Label for="external-link-url">URL</Label>
    <Input
      id="external-link-url"
      bind:value={draft.url}
      placeholder="https://github.com/org/repo/pull/482"
    />
  </div>
  <div class="space-y-2">
    <Label for="external-link-title">Title</Label>
    <Input id="external-link-title" bind:value={draft.title} placeholder="Follow-up PR" />
  </div>
  <div class="space-y-2">
    <Label for="external-link-status">Status</Label>
    <Input id="external-link-status" bind:value={draft.status} placeholder="open" />
  </div>
  <div class="space-y-2 md:col-span-2">
    <Label for="external-link-relation">Relation</Label>
    <Input id="external-link-relation" bind:value={draft.relation} placeholder="references" />
  </div>

  <div class="flex justify-end gap-2 md:col-span-2">
    {#if onCancel}
      <Button variant="outline" onclick={onCancel} disabled={creating}>Cancel</Button>
    {/if}
    <Button onclick={handleSubmit} disabled={creating}>
      {creating ? 'Adding…' : 'Add external link'}
    </Button>
  </div>
</div>
