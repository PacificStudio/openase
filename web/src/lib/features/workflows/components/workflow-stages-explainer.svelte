<script lang="ts">
  import * as Collapsible from '$ui/collapsible'
  import { ChevronRight, CircleDashed, Loader2, Eye, CircleCheck } from '@lucide/svelte'
  import { t } from './i18n'
  import type { TranslationKey } from '$lib/i18n'

  let { open = $bindable(false) }: { open?: boolean } = $props()

  type StageCard = {
    id: 'todo' | 'inProgress' | 'inReview' | 'done'
    icon: typeof CircleDashed
    labelKey: TranslationKey
    descKey: TranslationKey
    gateKey: TranslationKey
  }

  const stages: StageCard[] = [
    {
      id: 'todo',
      icon: CircleDashed,
      labelKey: 'workflows.creation.stages.todo.label',
      descKey: 'workflows.creation.stages.todo.description',
      gateKey: 'workflows.creation.stages.todo.gate',
    },
    {
      id: 'inProgress',
      icon: Loader2,
      labelKey: 'workflows.creation.stages.inProgress.label',
      descKey: 'workflows.creation.stages.inProgress.description',
      gateKey: 'workflows.creation.stages.inProgress.gate',
    },
    {
      id: 'inReview',
      icon: Eye,
      labelKey: 'workflows.creation.stages.inReview.label',
      descKey: 'workflows.creation.stages.inReview.description',
      gateKey: 'workflows.creation.stages.inReview.gate',
    },
    {
      id: 'done',
      icon: CircleCheck,
      labelKey: 'workflows.creation.stages.done.label',
      descKey: 'workflows.creation.stages.done.description',
      gateKey: 'workflows.creation.stages.done.gate',
    },
  ]
</script>

<Collapsible.Root bind:open>
  <Collapsible.Trigger>
    {#snippet child({ props })}
      <button
        {...props}
        type="button"
        data-tour="workflow-create-stages-explainer"
        class="text-muted-foreground hover:text-foreground flex items-center gap-1 text-sm transition-colors"
      >
        <ChevronRight class="size-4 transition-transform {open ? 'rotate-90' : ''}" />
        {t('workflows.creation.stages.toggle')}
      </button>
    {/snippet}
  </Collapsible.Trigger>
  <Collapsible.Content>
    <div class="mt-3 space-y-3">
      <p class="text-muted-foreground text-xs leading-relaxed">
        {t('workflows.creation.stages.intro')}
      </p>
      <ol class="space-y-2">
        {#each stages as stage, index (stage.id)}
          <li class="bg-muted/40 flex gap-3 rounded-md border px-3 py-2">
            <div class="flex shrink-0 flex-col items-center">
              <div
                class="bg-background text-muted-foreground flex size-6 items-center justify-center rounded-full border text-xs font-medium"
              >
                {index + 1}
              </div>
            </div>
            <div class="min-w-0 flex-1 space-y-1">
              <div class="flex items-center gap-2 text-sm font-medium">
                <stage.icon class="text-muted-foreground size-4" />
                {t(stage.labelKey)}
              </div>
              <p class="text-muted-foreground text-xs leading-relaxed">
                {t(stage.descKey)}
              </p>
              <p class="text-xs leading-relaxed">
                <span class="font-medium">{t('workflows.creation.stages.gateLabel')}</span>
                <span class="text-muted-foreground">{t(stage.gateKey)}</span>
              </p>
            </div>
          </li>
        {/each}
      </ol>
    </div>
  </Collapsible.Content>
</Collapsible.Root>
