<script lang="ts">
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    projectCount,
    providerCount,
    href = null,
  }: {
    projectCount: number
    providerCount: number
    href?: string | null
  } = $props()
</script>

<Card.Root class="rounded-2xl">
  <Card.Header>
    <Card.Title>
      {i18nStore.t('catalog.machine.panel.title')}
    </Card.Title>
    <Card.Description>
      {i18nStore.t('catalog.machine.panel.description')}
    </Card.Description>
  </Card.Header>

  <Card.Content class="space-y-3">
    <div class="grid grid-cols-2 gap-3 text-sm">
      <div class="border-border bg-background rounded-xl border px-3 py-3">
        <div class="text-muted-foreground text-xs tracking-[0.18em] uppercase">
          {i18nStore.t('catalog.machine.panel.labels.projects')}
        </div>
        <div class="text-foreground mt-2 text-xl font-semibold">{projectCount}</div>
      </div>
      <div class="border-border bg-background rounded-xl border px-3 py-3">
        <div class="text-muted-foreground text-xs tracking-[0.18em] uppercase">
          {i18nStore.t('catalog.machine.panel.labels.providers')}
        </div>
        <div class="text-foreground mt-2 text-xl font-semibold">{providerCount}</div>
      </div>
    </div>

    <p class="text-muted-foreground text-sm leading-6">
      {i18nStore.t('catalog.machine.panel.hint')}
    </p>

    <Button href={href ?? undefined} disabled={!href} class="w-full">
      {href
        ? i18nStore.t('catalog.machine.panel.actions.openConsole')
        : i18nStore.t('catalog.machine.panel.actions.createProjectFirst')}
    </Button>
  </Card.Content>
</Card.Root>
