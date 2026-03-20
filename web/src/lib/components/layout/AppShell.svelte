<script lang="ts">
  import type { Snippet } from 'svelte'
  import { cn } from '$lib/utils'

  let {
    children,
    header,
    sidebar,
    drawer,
    sidebarOpen = false,
    drawerOpen = false,
    onCloseSidebar,
    onCloseDrawer,
  }: {
    children?: Snippet
    header?: Snippet
    sidebar?: Snippet
    drawer?: Snippet
    sidebarOpen?: boolean
    drawerOpen?: boolean
    onCloseSidebar?: () => void
    onCloseDrawer?: () => void
  } = $props()
</script>

<div class="relative h-screen overflow-hidden">
  <div
    class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(194,120,3,0.18),transparent_28rem),radial-gradient(circle_at_bottom_right,rgba(13,148,136,0.14),transparent_30rem)]"
  ></div>

  {#if sidebar}
    <div
      class={cn(
        'bg-background/72 absolute inset-0 z-30 backdrop-blur-sm transition xl:hidden',
        sidebarOpen ? 'pointer-events-auto opacity-100' : 'pointer-events-none opacity-0',
      )}
      role="presentation"
      onclick={onCloseSidebar}
    ></div>
  {/if}

  {#if drawer}
    <div
      class={cn(
        'bg-background/72 absolute inset-0 z-30 backdrop-blur-sm transition xl:hidden',
        drawerOpen ? 'pointer-events-auto opacity-100' : 'pointer-events-none opacity-0',
      )}
      role="presentation"
      onclick={onCloseDrawer}
    ></div>
  {/if}

  <section class="relative mx-auto grid h-full w-full max-w-[112rem] grid-rows-[auto_minmax(0,1fr)] px-3 py-3 sm:px-4 lg:px-6">
    {#if header}
      <header class="min-h-0 pb-3">
        {@render header()}
      </header>
    {/if}

    <div class="grid min-h-0 gap-3 xl:grid-cols-[18rem_minmax(0,1fr)_23rem]">
      {#if sidebar}
        <aside
          class={cn(
            'bg-background/96 fixed inset-y-3 left-3 z-40 w-[18rem] rounded-[1.75rem] shadow-2xl shadow-black/12 transition xl:static xl:block xl:w-auto xl:bg-transparent xl:shadow-none',
            sidebarOpen ? 'translate-x-0' : '-translate-x-[110%] xl:translate-x-0',
          )}
        >
          {@render sidebar()}
        </aside>
      {/if}

      <main class="min-h-0 overflow-y-auto pr-1">
        {@render children?.()}
      </main>

      {#if drawer}
        <aside
          class={cn(
            'bg-background/96 fixed inset-y-3 right-3 z-40 w-[min(23rem,calc(100vw-1.5rem))] rounded-[1.75rem] shadow-2xl shadow-black/12 transition xl:static xl:block xl:w-auto xl:bg-transparent xl:shadow-none',
            drawerOpen ? 'translate-x-0' : 'translate-x-[110%] xl:translate-x-0',
          )}
        >
          {@render drawer()}
        </aside>
      {/if}
    </div>
  </section>
</div>
