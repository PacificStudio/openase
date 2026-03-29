<script lang="ts">
  import MessageSquare from '@lucide/svelte/icons/message-square'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'

  let {
    creating = false,
    onCreate,
  }: {
    creating?: boolean
    onCreate?: (body: string) => Promise<boolean> | boolean
  } = $props()

  let open = $state(false)
  let body = $state('')

  async function handleCreate() {
    const next = body.trim()
    if (!next || creating) return

    const success = (await onCreate?.(next)) ?? false
    if (success) {
      body = ''
      open = false
    }
  }

  function resetComposer() {
    open = false
    body = ''
  }
</script>

<div
  class="bg-muted/30 border-border relative z-10 mt-1 flex size-8 shrink-0 items-center justify-center rounded-full border"
>
  <MessageSquare class="size-4" />
</div>
<div class="min-w-0 flex-1">
  <div class="border-border bg-muted/10 rounded-xl border p-4">
    {#if open}
      <div class="mb-3 flex items-center gap-2">
        <MessageSquare class="text-muted-foreground size-4" />
        <span class="text-sm font-medium">Add a comment</span>
      </div>
      <Textarea
        rows={4}
        bind:value={body}
        placeholder="Leave a comment (Markdown supported)…"
        disabled={creating}
      />
      <div class="mt-3 flex justify-end gap-2">
        <Button size="sm" variant="outline" onclick={resetComposer} disabled={creating}>
          Cancel
        </Button>
        <Button size="sm" onclick={handleCreate} disabled={!body.trim() || creating}>
          {creating ? 'Posting…' : 'Comment'}
        </Button>
      </div>
    {:else}
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          <MessageSquare class="text-muted-foreground size-4" />
          <span class="text-sm font-medium">Comment on this ticket</span>
        </div>
        <Button size="sm" variant="outline" onclick={() => (open = true)}>Add comment</Button>
      </div>
    {/if}
  </div>
</div>
