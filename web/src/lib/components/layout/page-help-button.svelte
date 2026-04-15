<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'
  import type { ProjectSection } from '$lib/stores/app-context'
  import { getPageHelpSection } from './page-help-content'
  import * as Popover from '$ui/popover'
  import { CircleHelp } from '@lucide/svelte'

  let { section }: { section: ProjectSection } = $props()

  const content = $derived(getPageHelpSection(section))
</script>

{#if content}
  <Popover.Root>
    <Popover.Trigger>
      {#snippet child({ props })}
        <button
          {...props}
          type="button"
          class="text-muted-foreground hover:text-foreground inline-flex size-5 shrink-0 items-center justify-center rounded-full transition-colors"
          aria-label={i18nStore.t('help.popover.trigger')}
          data-tour={`page-help-${section}`}
        >
          <CircleHelp class="size-4" />
        </button>
      {/snippet}
    </Popover.Trigger>
    <Popover.Content class="w-96 max-w-[95vw] p-0" align="start" side="bottom">
      <div class="border-border border-b px-4 py-3">
        <div class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
          {i18nStore.t('help.popover.overviewLabel')}
        </div>
        <p class="text-foreground mt-1 text-sm leading-relaxed">
          {i18nStore.t(content.overviewKey)}
        </p>
      </div>
      <div class="max-h-[60vh] overflow-y-auto px-4 py-3">
        <div class="text-muted-foreground mb-2 text-[11px] font-medium tracking-wider uppercase">
          {i18nStore.t('help.popover.itemsLabel')}
        </div>
        <ul class="space-y-3">
          {#each content.items as item (item.id)}
            <li>
              <div class="text-foreground text-sm font-medium">{i18nStore.t(item.titleKey)}</div>
              <p class="text-muted-foreground mt-0.5 text-xs leading-relaxed">
                {i18nStore.t(item.descriptionKey)}
              </p>
            </li>
          {/each}
        </ul>
      </div>
    </Popover.Content>
  </Popover.Root>
{/if}
