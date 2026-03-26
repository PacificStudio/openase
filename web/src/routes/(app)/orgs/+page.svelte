<script lang="ts">
  import OrganizationIndexPage from '$lib/features/catalog-creation/components/organization-index-page.svelte'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()
</script>

<svelte:head>
  <title>Organizations - OpenASE</title>
</svelte:head>

<OrganizationIndexPage organizations={data.organizations} />
