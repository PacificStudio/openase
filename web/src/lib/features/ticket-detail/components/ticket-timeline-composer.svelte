<script lang="ts">
  import MessageSquare from '@lucide/svelte/icons/message-square'
  import Send from '@lucide/svelte/icons/send'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    creating = false,
    onCreate,
  }: {
    creating?: boolean
    onCreate?: (body: string) => Promise<boolean> | boolean
  } = $props()

  let body = $state('')
  let expanded = $state(false)
  const modifierKeySymbol =
    typeof navigator !== 'undefined' && navigator.platform?.includes('Mac') ? '⌘' : 'Ctrl'

  async function handleCreate() {
    const next = body.trim()
    if (!next || creating) return

    const success = (await onCreate?.(next)) ?? false
    if (success) {
      body = ''
      expanded = false
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
      event.preventDefault()
      void handleCreate()
    }
  }
</script>

<div
  class="bg-muted/30 border-border relative z-10 mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full border"
>
  <MessageSquare class="size-3" />
</div>
<div class="min-w-0 flex-1">
  <div class="border-border bg-muted/10 rounded-lg border px-3 py-2">
    <Textarea
      rows={expanded ? 4 : 1}
      bind:value={body}
      placeholder={i18nStore.t('ticketDetail.timelineComposer.placeholder')}
      disabled={creating}
      onfocus={() => (expanded = true)}
      onkeydown={handleKeydown}
      class="min-h-0 resize-none border-0 bg-transparent p-0 text-sm shadow-none focus-visible:ring-0"
    />
    {#if expanded}
      <div class="mt-2 flex items-center justify-between">
        <span class="text-muted-foreground text-[11px]">
          {i18nStore.t('ticketDetail.timelineComposer.shortcut', {
            modifier: modifierKeySymbol,
          })}
        </span>
        <Button
          size="sm"
          class="h-7 gap-1.5 px-2.5"
          onclick={handleCreate}
          disabled={!body.trim() || creating}
        >
          <Send class="size-3" />
          {creating
            ? i18nStore.t('ticketDetail.timelineComposer.actions.posting')
            : i18nStore.t('ticketDetail.timelineComposer.actions.comment')}
        </Button>
      </div>
    {/if}
  </div>
</div>
