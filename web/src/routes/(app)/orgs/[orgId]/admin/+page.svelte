<script lang="ts">
  import { OrganizationAdminPage } from '$lib/features/organizations'

  let { params }: { params: { orgId: string } } = $props()
</script>

<svelte:head>
  <title>Organization Admin - OpenASE</title>
</svelte:head>

<OrganizationAdminPage organizationId={params.orgId} />
