<script lang="ts">
  import { OrganizationAdminRolesPage } from '$lib/features/org-admin'
  import { appStore } from '$lib/stores/app.svelte'

  const organizationId = $derived(appStore.currentOrg?.id ?? '')
</script>

<OrganizationAdminRolesPage {organizationId} />
