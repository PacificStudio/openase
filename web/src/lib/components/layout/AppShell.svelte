<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    children,
    header,
    sidebar,
    drawer,
    drawerOpen = false,
  }: {
    children?: Snippet
    header?: Snippet
    sidebar?: Snippet
    drawer?: Snippet
    drawerOpen?: boolean
  } = $props()
</script>

<div class="relative min-h-screen overflow-hidden">
  <div
    class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(194,120,3,0.18),transparent_28rem),radial-gradient(circle_at_bottom_right,rgba(13,148,136,0.14),transparent_30rem)]"
  ></div>

  <section
    class="relative mx-auto flex min-h-screen w-full max-w-[95rem] flex-col gap-6 px-4 py-5 sm:px-6 lg:px-8"
  >
    {#if header}
      {@render header()}
    {/if}

    <div class="grid flex-1 gap-6 lg:grid-cols-[19rem_minmax(0,1fr)_24rem]">
      <aside class="space-y-6">
        {@render sidebar?.()}
      </aside>

      <main class="space-y-6">
        {@render children?.()}
      </main>

      <aside class={`${drawerOpen ? 'block' : 'hidden'} space-y-6 lg:block`}>
        {@render drawer?.()}
      </aside>
    </div>
  </section>
</div>
