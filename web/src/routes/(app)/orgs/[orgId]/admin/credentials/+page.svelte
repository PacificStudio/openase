<script lang="ts">
  import { OrganizationAdminCredentialsPage } from '$lib/features/org-admin'
  import { appStore } from '$lib/stores/app.svelte'

  const organizationId = $derived(appStore.currentOrg?.id ?? '')
</script>

<OrganizationAdminCredentialsPage {organizationId} />
