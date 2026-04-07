<script lang="ts">
  import { OrganizationMembersSection } from '$lib/features/organizations'

  let { organizationId }: { organizationId: string } = $props()
</script>

<OrganizationMembersSection
  {organizationId}
  mode="invitations"
  heading="Invitations"
  description="Invite people into the org, resend or revoke pending invites, and rotate accept tokens while onboarding stays under org-admin control."
  emptyMessage="No pending invitations right now."
/>
