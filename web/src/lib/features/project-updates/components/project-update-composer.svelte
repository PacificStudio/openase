<script lang="ts">
  import { cn } from '$lib/utils'
  import { Send, CircleCheck, AlertTriangle, CircleX } from '@lucide/svelte'
  import * as Select from '$ui/select'
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

  async function handleSubmit() {
    const nextTitle = title.trim()
    if (!nextTitle || creating) return

    const success = (await onSubmit?.({ status, title: nextTitle, body: nextTitle })) ?? false
    if (!success) return

    status = 'on_track'
    title = ''
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey && title.trim() && !creating) {
      e.preventDefault()
      void handleSubmit()
    }
  }

  const statusOptions: Array<{
    value: ProjectUpdateStatus
    label: string
    icon: typeof CircleCheck
    textClass: string
  }> = [
    { value: 'on_track', label: 'On track', icon: CircleCheck, textClass: 'text-emerald-600' },
    { value: 'at_risk', label: 'At risk', icon: AlertTriangle, textClass: 'text-amber-600' },
    { value: 'off_track', label: 'Off track', icon: CircleX, textClass: 'text-rose-600' },
  ]

  const currentStatusOption = $derived(
    statusOptions.find((o) => o.value === status) ?? statusOptions[0],
  )
  const CurrentStatusIcon = $derived(currentStatusOption.icon)
</script>

<div class="border-border bg-background rounded-xl border">
  <div class="flex items-center gap-2 px-3 py-2.5">
    <Select.Root
      type="single"
      value={status}
      onValueChange={(value) => {
        if (value) status = value as ProjectUpdateStatus
      }}
    >
      <Select.Trigger
        size="sm"
        class={cn(
          'w-auto shrink-0 gap-1 border-none px-2 text-xs font-medium shadow-none',
          currentStatusOption.textClass,
        )}
      >
        <CurrentStatusIcon class="size-3" />
        {currentStatusOption.label}
      </Select.Trigger>
      <Select.Content>
        {#each statusOptions as opt (opt.value)}
          {@const Icon = opt.icon}
          <Select.Item value={opt.value}>
            <Icon class={cn('size-3', opt.textClass)} />
            {opt.label}
          </Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
    <input
      type="text"
      bind:value={title}
      onkeydown={handleKeydown}
      placeholder="Post an update..."
      aria-label="New update title"
      class="text-foreground placeholder:text-muted-foreground min-w-0 flex-1 bg-transparent text-sm outline-none"
    />
    <button
      type="button"
      class={cn(
        'shrink-0 rounded-md p-1.5 transition-colors',
        title.trim() && !creating
          ? 'text-primary hover:bg-primary/10'
          : 'text-muted-foreground/40 cursor-not-allowed',
      )}
      disabled={!title.trim() || creating}
      onclick={handleSubmit}
      aria-label={creating ? 'Posting...' : 'Post update'}
    >
      <Send class="size-4" />
    </button>
  </div>
</div>
