<script lang="ts">
  import { OrganizationMembersSection } from '$lib/features/organizations'

  let { organizationId }: { organizationId: string } = $props()
</script>

<OrganizationMembersSection
  {organizationId}
  mode="members"
  heading="Members"
  description="Manage accepted members, clarify who belongs to the organization, and review which role changes flow into descendant projects."
  emptyMessage="No accepted or historical memberships yet."
/>
