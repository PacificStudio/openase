<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Textarea } from '$ui/textarea'
  import { projectUpdateStatusOptions } from '../status'
  import type { ProjectUpdateStatus } from '../types'

  let {
    creating = false,
    onSubmit,
  }: {
    creating?: boolean
    onSubmit?: (draft: {
      status: ProjectUpdateStatus
      title: string
      body: string
    }) => Promise<boolean> | boolean
  } = $props()

  let status = $state<ProjectUpdateStatus>('on_track')
  let title = $state('')
  let body = $state('')

  async function handleSubmit() {
    const nextTitle = title.trim()
    const nextBody = body.trim()
    if (!nextTitle || !nextBody || creating) return

    const success = (await onSubmit?.({ status, title: nextTitle, body: nextBody })) ?? false
    if (!success) return

    status = 'on_track'
    title = ''
    body = ''
  }
</script>

<section class="border-border bg-background rounded-2xl border shadow-sm">
  <div class="border-border border-b px-5 py-4">
    <h2 class="text-base font-semibold">Post a project update</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Use Updates for the latest human-authored project status. Runtime event logs remain in
      Activity.
    </p>
  </div>
  <div class="space-y-4 px-5 py-4">
    <div class="grid gap-3 md:grid-cols-[180px_minmax(0,1fr)]">
      <label class="space-y-1.5 text-sm">
        <span class="text-muted-foreground">Delivery status</span>
        <select
          bind:value={status}
          aria-label="New update status"
          class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-10 w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
        >
          {#each projectUpdateStatusOptions as option (option.value)}
            <option value={option.value}>{option.label}</option>
          {/each}
        </select>
      </label>
      <label class="space-y-1.5 text-sm">
        <span class="text-muted-foreground">Title</span>
        <Input bind:value={title} aria-label="New update title" placeholder="Sprint 2 rollout" />
      </label>
    </div>
    <label class="space-y-1.5 text-sm">
      <span class="text-muted-foreground">Body</span>
      <Textarea
        bind:value={body}
        aria-label="New update body"
        rows={5}
        placeholder="Summarize the latest delivery signal, risks, and next checkpoint."
      />
    </label>
    <div class="flex items-center justify-between gap-3">
      <p class="text-muted-foreground text-xs">
        This timeline is separate from system Activity and is ordered by the latest discussion.
      </p>
      <Button onclick={handleSubmit} disabled={!title.trim() || !body.trim() || creating}>
        {creating ? 'Posting…' : 'Post update'}
      </Button>
    </div>
  </div>
</section>
