<script lang="ts">
  import InstanceAdminPage from '$lib/features/admin/components/instance-admin-page.svelte'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<svelte:head>
  <title>
    {pageTitle(i18nStore.t('instanceAdmin.pageTitle'), i18nStore.locale)}
  </title>
</svelte:head>

<InstanceAdminPage />
