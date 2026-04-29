<script lang="ts">
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import AgentsPage from '$lib/features/agents/components/agents-page.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('agents.pageTitle'), i18nStore.locale)}</title>
</svelte:head>

<AgentsPage />
