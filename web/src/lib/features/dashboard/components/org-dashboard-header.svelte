<script lang="ts">
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import { Check, Pencil, X } from '@lucide/svelte'
  import type { ProjectStatus } from '../types'

  const projectStatusOptions: ProjectStatus[] = [
    'Backlog',
    'Planned',
    'In Progress',
    'Completed',
    'Canceled',
    'Archived',
  ]

  const statusClassName: Record<ProjectStatus, string> = {
    Backlog:
      'border-slate-200 bg-slate-100 text-slate-700 hover:bg-slate-200 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200',
    Planned:
      'border-amber-200 bg-amber-50 text-amber-800 hover:bg-amber-100 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-200',
    'In Progress':
      'border-emerald-200 bg-emerald-50 text-emerald-800 hover:bg-emerald-100 dark:border-emerald-900/60 dark:bg-emerald-950/40 dark:text-emerald-200',
    Completed:
      'border-sky-200 bg-sky-50 text-sky-800 hover:bg-sky-100 dark:border-sky-900/60 dark:bg-sky-950/40 dark:text-sky-200',
    Canceled:
      'border-rose-200 bg-rose-50 text-rose-800 hover:bg-rose-100 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200',
    Archived:
      'border-border bg-background text-muted-foreground hover:bg-muted dark:hover:bg-muted/60',
  }

  let {
    editingInfo,
    editName,
    editDescription,
    savingInfo,
    savingStatus,
    currentStatus,
    projectName,
    projectDescription,
    onEditNameChange,
    onEditDescriptionChange,
    onStartEditInfo,
    onCancelEditInfo,
    onSaveInfo,
    onProjectStatusChange,
  }: {
    editingInfo: boolean
    editName: string
    editDescription: string
    savingInfo: boolean
    savingStatus: boolean
    currentStatus: ProjectStatus
    projectName: string
    projectDescription: string
    onEditNameChange?: (value: string) => void
    onEditDescriptionChange?: (value: string) => void
    onStartEditInfo?: () => void
    onCancelEditInfo?: () => void
    onSaveInfo?: () => void | Promise<void>
    onProjectStatusChange?: (status: ProjectStatus) => void | Promise<void>
  } = $props()
</script>

<div class="border-border border-b px-6 py-4">
  <div class="flex items-start justify-between gap-3">
    {#if editingInfo}
      <div class="min-w-0 flex-1 space-y-2">
        <Input
          value={editName}
          placeholder="Project name"
          class="text-lg font-semibold"
          oninput={(event) => onEditNameChange?.((event.currentTarget as HTMLInputElement).value)}
        />
        <Textarea
          value={editDescription}
          placeholder="Project description (optional)"
          rows={2}
          class="text-sm"
          oninput={(event) =>
            onEditDescriptionChange?.((event.currentTarget as HTMLTextAreaElement).value)}
        />
        <div class="flex items-center gap-2">
          <Button size="sm" disabled={savingInfo} onclick={() => onSaveInfo?.()}>
            <Check class="mr-1.5 size-3.5" />
            {savingInfo ? 'Saving\u2026' : 'Save'}
          </Button>
          <Button variant="ghost" size="sm" disabled={savingInfo} onclick={onCancelEditInfo}>
            <X class="mr-1.5 size-3.5" />
            Cancel
          </Button>
        </div>
      </div>
    {:else}
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <h1 class="text-foreground truncate text-lg font-semibold">{projectName}</h1>
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground shrink-0 transition-colors"
            title="Edit project info"
            onclick={onStartEditInfo}
          >
            <Pencil class="size-3.5" />
          </button>
        </div>
        {#if projectDescription}
          <p class="text-muted-foreground mt-0.5 text-sm">{projectDescription}</p>
        {:else}
          <p class="text-muted-foreground/50 mt-0.5 text-sm">No description</p>
        {/if}
      </div>
    {/if}

    <div class="flex shrink-0 items-center gap-2">
      <Select.Root
        type="single"
        value={currentStatus}
        onValueChange={(value) => {
          if (value && value !== currentStatus) void onProjectStatusChange?.(value as ProjectStatus)
        }}
      >
        <Select.Trigger
          class={cn(
            'h-auto min-h-5 w-auto rounded-full border px-2.5 py-1 text-xs font-medium shadow-none',
            statusClassName[currentStatus],
          )}
          disabled={savingStatus}
        >
          {currentStatus}
        </Select.Trigger>
        <Select.Content>
          {#each projectStatusOptions as status (status)}
            <Select.Item value={status}>{status}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>
  </div>
</div>
