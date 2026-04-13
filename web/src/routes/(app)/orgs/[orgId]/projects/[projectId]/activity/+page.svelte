<script lang="ts">
  import ActivityPage from '$lib/features/activity/components/activity-page.svelte'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('nav.activity'), i18nStore.locale)}</title>
</svelte:head>

<ActivityPage />
