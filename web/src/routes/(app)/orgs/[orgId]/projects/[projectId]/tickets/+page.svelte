<script lang="ts">
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import TicketsPage from '$lib/features/tickets/components/tickets-page.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('tickets.pageTitle'), i18nStore.locale)}</title>
</svelte:head>

<TicketsPage />
