<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'

  goto(`/orgs/${page.params.orgId}/admin/members`, { replaceState: true })
</script>
