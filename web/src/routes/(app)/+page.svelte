<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { pageTitle } from '$lib/i18n'
  import WorkspaceOrganizationCreation from '$lib/features/catalog-creation/components/workspace-organization-creation.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('dashboard.pageTitle'), i18nStore.locale)}</title>
</svelte:head>

<WorkspaceOrganizationCreation />
