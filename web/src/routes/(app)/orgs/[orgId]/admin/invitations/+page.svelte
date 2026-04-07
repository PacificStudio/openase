<script lang="ts">
  import { OrganizationAdminInvitationsPage } from '$lib/features/org-admin'
  import { appStore } from '$lib/stores/app.svelte'

  const organizationId = $derived(appStore.currentOrg?.id ?? '')
</script>

<OrganizationAdminInvitationsPage {organizationId} />
