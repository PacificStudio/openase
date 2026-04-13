<script lang="ts">
  import OrgDashboard from '$lib/features/dashboard/components/org-dashboard.svelte'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('nav.dashboard'), i18nStore.locale)}</title>
</svelte:head>

<OrgDashboard />
