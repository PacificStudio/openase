<script lang="ts">
  import OrganizationIndexPage from '$lib/features/catalog-creation/components/organization-index-page.svelte'
  import { appStore } from '$lib/stores/app.svelte'
</script>

<svelte:head>
  <title>Organizations - OpenASE</title>
</svelte:head>

<OrganizationIndexPage organizations={appStore.organizations} />

{#if appStore.organizations.length === 0 && appStore.appContextLoading}
  <div class="mx-auto w-full max-w-6xl px-6 pb-6">
    <div class="text-muted-foreground text-sm">Loading organizations…</div>
  </div>
{/if}
