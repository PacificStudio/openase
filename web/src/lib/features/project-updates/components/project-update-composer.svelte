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
    onSubmit?: (draft: { status: ProjectUpdateStatus; body: string }) => Promise<boolean> | boolean
  } = $props()

  let status = $state<ProjectUpdateStatus>('on_track')
  let body = $state('')

  async function handleSubmit() {
    const nextBody = body.trim()
    if (!nextBody || creating) return

    const success = (await onSubmit?.({ status, body: nextBody })) ?? false
    if (!success) return

    status = 'on_track'
    body = ''
    if (textareaRef) {
      textareaRef.style.height = '20px'
      textareaRef.style.overflowY = 'hidden'
    }
  }

  let textareaRef = $state<HTMLTextAreaElement | null>(null)

  function autoResize() {
    if (!textareaRef) return
    textareaRef.style.height = 'auto'
    const lineHeight = 20
    const maxHeight = lineHeight * 4
    textareaRef.style.height = `${Math.min(textareaRef.scrollHeight, maxHeight)}px`
    textareaRef.style.overflowY = textareaRef.scrollHeight > maxHeight ? 'auto' : 'hidden'
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey && body.trim() && !creating) {
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
  <div class="flex items-end gap-2 px-3 py-2.5">
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
    <textarea
      bind:this={textareaRef}
      bind:value={body}
      onkeydown={handleKeydown}
      oninput={autoResize}
      placeholder="Write an update..."
      aria-label="New update body"
      rows="1"
      class="text-foreground placeholder:text-muted-foreground min-w-0 flex-1 resize-none bg-transparent text-sm leading-5 outline-none"
      style="height: 20px; overflow-y: hidden;"
    ></textarea>
    <button
      type="button"
      class={cn(
        'shrink-0 rounded-md p-1.5 transition-colors',
        body.trim() && !creating
          ? 'text-primary hover:bg-primary/10'
          : 'text-muted-foreground/40 cursor-not-allowed',
      )}
      disabled={!body.trim() || creating}
      onclick={handleSubmit}
      aria-label={creating ? 'Posting...' : 'Post update'}
    >
      <Send class="size-4" />
    </button>
  </div>
</div>
