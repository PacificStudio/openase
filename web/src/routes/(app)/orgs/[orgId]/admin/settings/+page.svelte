<script lang="ts">
  import { OrganizationAdminSettingsPage } from '$lib/features/org-admin'
  import { appStore } from '$lib/stores/app.svelte'

  const organizationId = $derived(appStore.currentOrg?.id ?? '')
</script>

<OrganizationAdminSettingsPage {organizationId} />
