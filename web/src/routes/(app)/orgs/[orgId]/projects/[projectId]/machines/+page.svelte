<script lang="ts">
  import { MachinesPage } from '$lib/features/machines'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('nav.machines'), i18nStore.locale)}</title>
</svelte:head>

<MachinesPage />
