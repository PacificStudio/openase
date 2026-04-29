<script lang="ts">
  import SettingsPage from '$lib/features/settings/components/settings-page.svelte'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('nav.settings'), i18nStore.locale)}</title>
</svelte:head>

<SettingsPage />
