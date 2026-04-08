<script lang="ts">
  import { OrganizationMembersSection } from '$lib/features/organizations'

  let { organizationId }: { organizationId: string } = $props()
</script>

<OrganizationMembersSection
  {organizationId}
  mode="members"
  heading="Members"
  description="Active members and their roles. Role changes apply to all projects in this org."
  emptyMessage="No accepted memberships yet."
/>
