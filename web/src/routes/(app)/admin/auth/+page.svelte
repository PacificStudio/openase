<script lang="ts">
  import AdminAuthPage from '$lib/features/admin-auth/components/admin-auth-page.svelte'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<svelte:head>
  <title>
    {pageTitle(i18nStore.t('adminAuth.pageTitle'), i18nStore.locale)}
  </title>
</svelte:head>

<AdminAuthPage />
