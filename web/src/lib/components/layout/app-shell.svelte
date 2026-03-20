<script lang="ts">
  import { cn } from '$lib/utils'
  import type { Snippet } from 'svelte'

  let {
    sidebar,
    children,
    rightPanel,
    sidebarCollapsed = false,
    rightPanelOpen = false,
  }: {
    sidebar: Snippet
    children: Snippet
    rightPanel?: Snippet
    sidebarCollapsed?: boolean
    rightPanelOpen?: boolean
  } = $props()
</script>

<div class="flex h-screen overflow-hidden bg-background">
  <!-- Sidebar -->
  <aside
    class={cn(
      'flex h-full flex-col border-r border-border bg-sidebar transition-[width] duration-200 ease-in-out',
      sidebarCollapsed ? 'w-[52px]' : 'w-[240px]',
    )}
  >
    {@render sidebar()}
  </aside>

  <!-- Main Content -->
  <main class="flex min-w-0 flex-1 flex-col overflow-hidden">
    {@render children()}
  </main>

  <!-- Right Panel -->
  {#if rightPanel && rightPanelOpen}
    <aside
      class="flex w-[380px] flex-col border-l border-border bg-card overflow-hidden"
    >
      {@render rightPanel()}
    </aside>
  {/if}
</div>
