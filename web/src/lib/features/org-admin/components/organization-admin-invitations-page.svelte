<script lang="ts">
  import { OrganizationMembersSection } from '$lib/features/organizations'

  let { organizationId }: { organizationId: string } = $props()
</script>

<OrganizationMembersSection
  {organizationId}
  mode="invitations"
  heading="Invitations"
  description="Pending invitations. Resend or cancel at any time."
  emptyMessage="No pending invitations."
/>
