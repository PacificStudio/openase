<script lang="ts">
  import { cn } from '$lib/utils'
  import { Send, CircleCheck, AlertTriangle, CircleX } from '@lucide/svelte'
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
    activeClass: string
  }> = [
    {
      value: 'on_track',
      label: 'On track',
      icon: CircleCheck,
      activeClass:
        'border-emerald-400 bg-emerald-50 text-emerald-700 dark:border-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-300',
    },
    {
      value: 'at_risk',
      label: 'At risk',
      icon: AlertTriangle,
      activeClass:
        'border-amber-400 bg-amber-50 text-amber-700 dark:border-amber-600 dark:bg-amber-950/40 dark:text-amber-300',
    },
    {
      value: 'off_track',
      label: 'Off track',
      icon: CircleX,
      activeClass:
        'border-rose-400 bg-rose-50 text-rose-700 dark:border-rose-600 dark:bg-rose-950/40 dark:text-rose-300',
    },
  ]
</script>

<div class="border-border bg-background rounded-xl border">
  <div class="flex items-center gap-1 px-3 pt-2.5">
    {#each statusOptions as opt (opt.value)}
      {@const Icon = opt.icon}
      <button
        type="button"
        class={cn(
          'flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium transition-colors',
          status === opt.value
            ? opt.activeClass
            : 'text-muted-foreground hover:bg-muted border-transparent',
        )}
        onclick={() => (status = opt.value)}
        aria-label={`Set status: ${opt.label}`}
      >
        <Icon class="size-3" />
        {opt.label}
      </button>
    {/each}
  </div>
  <div class="flex items-center gap-2 px-3 pt-1.5 pb-2.5">
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
