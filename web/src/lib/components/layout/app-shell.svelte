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

<div class="bg-background flex h-screen overflow-hidden">
  <!-- Sidebar -->
  <aside
    class={cn(
      'border-border bg-sidebar flex h-full flex-col border-r transition-[width] duration-200 ease-in-out',
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
    <aside class="border-border bg-card flex w-[380px] flex-col overflow-hidden border-l">
      {@render rightPanel()}
    </aside>
  {/if}
</div>
