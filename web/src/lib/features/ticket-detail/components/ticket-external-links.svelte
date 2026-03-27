<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import Link2 from '@lucide/svelte/icons/link-2'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { TicketExternalLink, TicketExternalLinkDraft } from '../types'

  let {
    links,
    creating = false,
    deletingId = null,
    onCreate,
    onDelete,
  }: {
    links: TicketExternalLink[]
    creating?: boolean
    deletingId?: string | null
    onCreate?: (draft: TicketExternalLinkDraft) => Promise<boolean> | boolean
    onDelete?: (linkId: string) => void
  } = $props()

  let draft = $state<TicketExternalLinkDraft>({
    type: 'github',
    url: '',
    externalId: '',
    title: '',
    status: '',
    relation: 'references',
  })
  async function handleSubmit() {
    const normalized: TicketExternalLinkDraft = {
      type: draft.type.trim().toLowerCase(),
      url: draft.url.trim(),
      externalId: draft.externalId.trim(),
      title: draft.title.trim(),
      status: draft.status.trim(),
      relation: draft.relation.trim() || 'references',
    }

    if (!normalized.type) {
      toastStore.error('Link type is required.')
      return
    }
    if (!normalized.url) {
      toastStore.error('Link URL is required.')
      return
    }
    if (!normalized.externalId) {
      toastStore.error('External ID is required.')
      return
    }

    const accepted = await onCreate?.(normalized)
    if (accepted) {
      draft = {
        ...draft,
        url: '',
        externalId: '',
        title: '',
        status: '',
      }
    }
  }
</script>

<div class="flex flex-col gap-3">
  <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
    External Links
  </span>

  {#if links.length > 0}
    <div class="space-y-2">
      {#each links as link (link.id)}
        <div class="border-border/60 bg-muted/30 rounded-md border px-2.5 py-2 text-xs">
          <div class="flex items-start justify-between gap-3">
            <a
              class="flex min-w-0 flex-1 items-start gap-2"
              href={link.url}
              target="_blank"
              rel="noreferrer"
            >
              <Link2 class="text-muted-foreground mt-0.5 size-3.5 shrink-0" />
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="text-foreground truncate">{link.title || link.externalId}</span>
                  <Badge variant="outline" class="h-4 shrink-0 py-0 text-[10px]">
                    {link.type}
                  </Badge>
                </div>
                <div
                  class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-[10px]"
                >
                  <span class="font-mono">{link.externalId}</span>
                  <span>{link.relation}</span>
                  {#if link.status}
                    <span>{link.status}</span>
                  {/if}
                </div>
              </div>
            </a>

            <Button
              variant="ghost"
              size="icon"
              class="size-7 shrink-0"
              disabled={deletingId === link.id}
              onclick={() => onDelete?.(link.id)}
            >
              <Trash2 class="size-3.5" />
            </Button>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="text-muted-foreground rounded-md border border-dashed px-3 py-4 text-xs">
      No external links attached yet.
    </div>
  {/if}

  <div class="border-border bg-muted/20 rounded-md border p-3">
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
    </div>

    <div class="mt-3 flex justify-end">
      <Button onclick={handleSubmit} disabled={creating}>
        {creating ? 'Adding…' : 'Add external link'}
      </Button>
    </div>
  </div>
</div>
