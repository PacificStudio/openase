<script lang="ts">
  import WorkflowsPage from '$lib/features/workflows/components/workflows-page.svelte'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('nav.workflows'), i18nStore.locale)}</title>
</svelte:head>

<WorkflowsPage />
