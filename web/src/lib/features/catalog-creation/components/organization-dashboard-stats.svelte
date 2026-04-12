<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    orgName,
    projectCount,
    providerCount,
  }: {
    orgName: string
    projectCount: number
    providerCount: number
  } = $props()
</script>

<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
  <div class="border-border bg-card rounded-lg border p-4">
      <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">
        {i18nStore.t('catalog.dashboard.stats.labels.org')}
      </p>
    <p class="text-foreground mt-2 text-lg font-semibold">{orgName}</p>
      <p class="text-muted-foreground mt-1 text-sm">
        {i18nStore.t('catalog.dashboard.stats.contextHint')}
      </p>
  </div>
  <div class="border-border bg-card rounded-lg border p-4">
      <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">
        {i18nStore.t('catalog.dashboard.stats.labels.projects')}
      </p>
    <p class="text-foreground mt-2 text-lg font-semibold">{projectCount}</p>
      <p class="text-muted-foreground mt-1 text-sm">
        {i18nStore.t('catalog.dashboard.stats.projectsHint')}
      </p>
  </div>
  <div class="border-border bg-card rounded-lg border p-4">
      <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">
        {i18nStore.t('catalog.dashboard.stats.labels.providers')}
      </p>
    <p class="text-foreground mt-2 text-lg font-semibold">{providerCount}</p>
      <p class="text-muted-foreground mt-1 text-sm">
        {i18nStore.t('catalog.dashboard.stats.providersHint')}
      </p>
  </div>
</div>
