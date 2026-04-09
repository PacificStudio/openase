<script lang="ts">
  import { OrganizationMembersSection } from '$lib/features/organizations'

  let { organizationId }: { organizationId: string } = $props()
</script>

<OrganizationMembersSection {organizationId} />
