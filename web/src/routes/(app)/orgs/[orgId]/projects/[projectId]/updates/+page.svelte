<script lang="ts">
  import { ProjectUpdatesPage } from '$lib/features/project-updates'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('nav.updates'), i18nStore.locale)}</title>
</svelte:head>

<ProjectUpdatesPage />
