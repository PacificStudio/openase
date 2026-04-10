<script lang="ts">
  import '@xterm/xterm/css/xterm.css'
  import { cn } from '$lib/utils'
  import {
    AlertCircle,
    LoaderCircle,
    Plus,
    SquareTerminal,
    X,
  } from '@lucide/svelte'
  import type { TerminalManager } from './terminal-manager.svelte'

  let {
    manager,
  }: {
    manager: TerminalManager
  } = $props()

  // Map of terminal id -> bound element
  let terminalElements: Record<string, HTMLDivElement> = {}

  // Mount & auto-connect new terminals
  $effect(() => {
    const insts = manager.instances
    for (const inst of insts) {
      const el = terminalElements[inst.id]
      if (el) {
        void mountAndConnect(inst.id, el)
      }
    }
  })

  async function mountAndConnect(id: string, element: HTMLDivElement) {
    await manager.mountTerminal(id, element)
    const inst = manager.instances.find((i) => i.id === id)
    if (inst && inst.status === 'idle') {
      void manager.connectTerminal(id)
    }
  }

  function handleNewTerminal() {
    manager.createInstance()
  }
</script>

<div class="flex min-h-0 flex-1 flex-col">
  <!-- Tab bar -->
  <div class="border-border flex h-8 items-center gap-0 border-b px-1">
    {#each manager.instances as inst (inst.id)}
      <div
        role="tab"
        tabindex="0"
        class={cn(
          'group relative flex h-full cursor-pointer items-center gap-1 border-b-2 px-2.5 text-[11px] transition-colors',
          inst.id === manager.activeId
            ? 'border-primary text-foreground'
            : 'border-transparent text-muted-foreground hover:text-foreground',
        )}
        aria-selected={inst.id === manager.activeId}
        onclick={() => {
          manager.activeId = inst.id
        }}
        onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter' || e.key === ' ') manager.activeId = inst.id }}
      >
        {#if inst.status === 'connecting'}
          <LoaderCircle class="size-2.5 animate-spin" />
        {:else if inst.status === 'error'}
          <AlertCircle class="size-2.5 text-red-400" />
        {/if}
        <span class="max-w-[80px] truncate">{inst.label}</span>
        <button
          type="button"
          class="ml-0.5 flex size-3.5 items-center justify-center rounded-sm opacity-0 transition-opacity hover:bg-muted group-hover:opacity-100"
          onclick={(e: MouseEvent) => { e.stopPropagation(); manager.removeInstance(inst.id) }}
          aria-label="Close terminal"
        >
          <X class="size-2" />
        </button>
      </div>
    {/each}
    <button
      type="button"
      class="text-muted-foreground hover:text-foreground ml-0.5 flex size-6 items-center justify-center rounded-sm transition-colors hover:bg-muted"
      aria-label="New terminal"
      onclick={handleNewTerminal}
    >
      <Plus class="size-3" />
    </button>
  </div>

  <!-- Terminal content area -->
  <div class="relative min-h-0 flex-1 bg-[#08131f]">
    {#each manager.instances as inst (inst.id)}
      <div
        class={cn(
          'absolute inset-0 px-2 py-2',
          inst.id === manager.activeId ? 'visible' : 'invisible',
        )}
        bind:this={terminalElements[inst.id]}
        data-testid="workspace-terminal-instance"
      ></div>
    {/each}

    <!-- Status indicator -->
    {#if manager.getActiveInstance()}
      {@const active = manager.getActiveInstance()}
      {#if active && active.status !== 'open'}
        <div class="pointer-events-none absolute inset-x-3 bottom-3 flex justify-end">
          <div
            class={cn(
              'border-border/50 bg-background/80 pointer-events-auto max-w-xs rounded-md border px-2.5 py-1.5 shadow-sm backdrop-blur',
              active.status === 'error' && 'border-destructive/40',
            )}
          >
            <div class="flex items-center gap-1.5">
              {#if active.status === 'error'}
                <AlertCircle class="text-destructive size-3 shrink-0" />
              {:else if active.status === 'connecting'}
                <LoaderCircle class="text-muted-foreground size-3 shrink-0 animate-spin" />
              {:else}
                <SquareTerminal class="text-muted-foreground size-3 shrink-0" />
              {/if}
              <p class="text-muted-foreground text-[11px]">{active.statusMessage}</p>
            </div>
          </div>
        </div>
      {/if}
    {/if}
  </div>
</div>
