<script lang="ts">
  import { OrganizationAdminShell } from '$lib/features/org-admin'
  import type { LayoutData } from './$types'
  import type { Snippet } from 'svelte'

  let { data, children }: { data: LayoutData; children: Snippet } = $props()
</script>

<OrganizationAdminShell organizationId={data.organizationId} currentPath={data.currentPath}>
  {@render children()}
</OrganizationAdminShell>
