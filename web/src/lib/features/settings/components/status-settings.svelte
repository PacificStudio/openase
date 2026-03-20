<script lang="ts">
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Separator } from '$ui/separator'
  import {
    GripVertical,
    Trash2,
    Plus,
    Circle,
    Loader,
    CheckCircle,
    Eye,
    Archive,
  } from '@lucide/svelte'
  import type { Component } from 'svelte'

  type StatusItem = {
    id: string
    name: string
    color: string
    icon: Component
  }

  const iconMap: Record<string, Component> = {
    circle: Circle,
    loader: Loader,
    check: CheckCircle,
    eye: Eye,
    archive: Archive,
  }

  let statuses = $state<StatusItem[]>([
    { id: '1', name: 'Backlog', color: '#6b7280', icon: Archive },
    { id: '2', name: 'Todo', color: '#3b82f6', icon: Circle },
    { id: '3', name: 'In Progress', color: '#f59e0b', icon: Loader },
    { id: '4', name: 'In Review', color: '#8b5cf6', icon: Eye },
    { id: '5', name: 'Done', color: '#10b981', icon: CheckCircle },
  ])

  let confirmDeleteId = $state<string | null>(null)

  let dragIndex = $state<number | null>(null)

  function addStatus() {
    statuses = [
      ...statuses,
      {
        id: crypto.randomUUID(),
        name: 'New Status',
        color: '#6b7280',
        icon: Circle,
      },
    ]
  }

  function removeStatus(id: string) {
    if (confirmDeleteId !== id) {
      confirmDeleteId = id
      return
    }
    statuses = statuses.filter((s) => s.id !== id)
    confirmDeleteId = null
  }

  function handleDragStart(index: number) {
    dragIndex = index
  }

  function handleDragOver(e: DragEvent, index: number) {
    e.preventDefault()
    if (dragIndex === null || dragIndex === index) return
    const updated = [...statuses]
    const [moved] = updated.splice(dragIndex, 1)
    updated.splice(index, 0, moved)
    statuses = updated
    dragIndex = index
  }

  function handleDragEnd() {
    dragIndex = null
  }
</script>

<div class="max-w-lg space-y-6">
  <div>
    <h2 class="text-base font-semibold text-foreground">Statuses</h2>
    <p class="mt-1 text-sm text-muted-foreground">
      Manage the columns on your board. Drag to reorder.
    </p>
  </div>

  <Separator />

  <div class="space-y-1">
    {#each statuses as status, i (status.id)}
      <div
        draggable="true"
        ondragstart={() => handleDragStart(i)}
        ondragover={(e) => handleDragOver(e, i)}
        ondragend={handleDragEnd}
        class={cn(
          'group flex items-center gap-2 rounded-md border border-transparent px-2 py-2',
          'hover:border-border hover:bg-muted/30 transition-colors',
          dragIndex === i && 'opacity-50',
        )}
      >
        <GripVertical
          class="size-4 shrink-0 cursor-grab text-muted-foreground/50"
        />
        <input
          type="color"
          bind:value={status.color}
          class="size-6 shrink-0 cursor-pointer rounded border-0 bg-transparent p-0"
        />
        <Input
          bind:value={status.name}
          class="h-8 flex-1 text-sm"
        />
        <Button
          variant="ghost"
          size="icon-xs"
          class={cn(
            'shrink-0',
            confirmDeleteId === status.id
              ? 'text-red-500 hover:text-red-400'
              : 'text-muted-foreground opacity-0 group-hover:opacity-100',
          )}
          onclick={() => removeStatus(status.id)}
          aria-label={confirmDeleteId === status.id ? 'Confirm delete' : 'Delete status'}
        >
          <Trash2 class="size-3.5" />
        </Button>
      </div>
    {/each}
  </div>

  {#if confirmDeleteId}
    <p class="text-xs text-red-400">
      Click the trash icon again to confirm deletion.
    </p>
  {/if}

  <Button variant="outline" size="sm" onclick={addStatus}>
    <Plus class="mr-1.5 size-3.5" />
    Add status
  </Button>
</div>
