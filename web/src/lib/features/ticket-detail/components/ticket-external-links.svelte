<script lang="ts">
  import { Badge } from '$ui/badge'
  import { buttonVariants, Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import Link2 from '@lucide/svelte/icons/link-2'
  import Plus from '@lucide/svelte/icons/plus'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import TicketExternalLinkForm from './ticket-external-link-form.svelte'
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

  let createOpen = $state(false)

  async function handleCreate(draft: {
    type: string
    url: string
    externalId: string
    title: string
    status: string
    relation: string
  }) {
    const accepted = (await onCreate?.(draft)) ?? false
    if (accepted) {
      createOpen = false
    }
    return accepted
  }
</script>

<div class="flex flex-col gap-3">
  <div class="flex items-center justify-between gap-3">
    <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
      External Links
    </span>

    <Dialog.Root bind:open={createOpen}>
      <Dialog.Trigger
        class={buttonVariants({ variant: 'outline', size: 'icon-sm' })}
        aria-label="Add external link"
      >
        <Plus class="size-3.5" />
      </Dialog.Trigger>
      <Dialog.Content class="sm:max-w-xl">
        <Dialog.Header>
          <Dialog.Title>Add external link</Dialog.Title>
          <Dialog.Description>
            Attach a pull request, issue, document, or other external reference to this ticket.
          </Dialog.Description>
        </Dialog.Header>

        <TicketExternalLinkForm
          {creating}
          onCreate={handleCreate}
          onCancel={() => {
            createOpen = false
          }}
        />
      </Dialog.Content>
    </Dialog.Root>
  </div>

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
                <div class="flex items-start gap-2">
                  <span class="text-foreground leading-4 break-words"
                    >{link.title || link.externalId}</span
                  >
                  <Badge variant="outline" class="mt-0.5 h-4 shrink-0 py-0 text-[10px]">
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
  {/if}
</div>
