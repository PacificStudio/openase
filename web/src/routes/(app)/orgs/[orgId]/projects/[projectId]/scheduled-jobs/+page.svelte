<script lang="ts">
  import ScheduledJobsPage from '$lib/features/scheduled-jobs/components/scheduled-jobs-page.svelte'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('nav.scheduledJobs'), i18nStore.locale)}</title>
</svelte:head>

<ScheduledJobsPage />
