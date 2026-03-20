<script lang="ts" module>
  export type SettingsNavItem = {
    label: string
    href?: string
    description?: string
    badge?: string
    disabled?: boolean
  }
</script>

<script lang="ts">
  import type { Snippet } from 'svelte'
  import { cn } from '$lib/utils'
  import ScrollPane from './ScrollPane.svelte'
  import SurfacePanel from './SurfacePanel.svelte'

  let {
    children,
    title = 'Settings',
    description = '',
    items = [],
    currentPath = '',
  }: {
    children?: Snippet
    title?: string
    description?: string
    items?: SettingsNavItem[]
    currentPath?: string
  } = $props()

  function isActive(item: SettingsNavItem) {
    if (!item.href) {
      return false
    }

    return currentPath === item.href || currentPath.startsWith(`${item.href}/`)
  }
</script>

<div class="grid min-h-0 gap-4 xl:grid-cols-[15rem_minmax(0,1fr)]">
  <SurfacePanel class="min-h-0 xl:sticky xl:top-0 xl:max-h-full">
    {#snippet header()}
      <div>
        <p class="text-sm font-semibold">{title}</p>
        {#if description}
          <p class="text-muted-foreground mt-1 text-xs leading-5">{description}</p>
        {/if}
      </div>
    {/snippet}

    <ScrollPane class="px-3 py-3">
      <nav class="grid gap-2">
        {#each items as item}
          {#if item.href && !item.disabled}
            <a
              href={item.href}
              class={cn(
                'rounded-2xl border px-3 py-3 text-sm transition',
                isActive(item)
                  ? 'border-foreground/25 bg-foreground text-background shadow-sm'
                  : 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background',
              )}
            >
              <div class="flex items-center justify-between gap-2">
                <span class="font-medium">{item.label}</span>
                {#if item.badge}
                  <span
                    class={cn(
                      'rounded-full border px-2 py-0.5 text-[11px]',
                      isActive(item)
                        ? 'border-background/20 bg-background/10 text-background/75'
                        : 'border-border/70 text-muted-foreground',
                    )}
                  >
                    {item.badge}
                  </span>
                {/if}
              </div>
              {#if item.description}
                <p
                  class={cn(
                    'mt-1 text-xs leading-5',
                    isActive(item) ? 'text-background/75' : 'text-muted-foreground',
                  )}
                >
                  {item.description}
                </p>
              {/if}
            </a>
          {:else}
            <div class="border-border/70 bg-muted/35 rounded-2xl border px-3 py-3 text-sm">
              <div class="flex items-center justify-between gap-2">
                <span class="font-medium">{item.label}</span>
                <span class="text-muted-foreground text-[11px]">Soon</span>
              </div>
              {#if item.description}
                <p class="text-muted-foreground mt-1 text-xs leading-5">{item.description}</p>
              {/if}
            </div>
          {/if}
        {/each}
      </nav>
    </ScrollPane>
  </SurfacePanel>

  <div class="min-h-0">
    {@render children?.()}
  </div>
</div>
