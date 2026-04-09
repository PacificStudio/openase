<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation'
  import { acceptOrganizationInvitation } from '$lib/api/auth'
  import { ApiError } from '$lib/api/client'
  import { createOrganization } from '$lib/api/openase'
  import {
    createOrganizationDraft,
    parseOrganizationDraft,
    slugFromName,
    type OrganizationCreationDraft,
  } from '$lib/features/catalog-creation/model'
  import { organizationPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'

  let draft = $state<OrganizationCreationDraft>(createOrganizationDraft())
  let saving = $state(false)
  let slugDirty = $state(false)
  let inviteToken = $state('')
  let acceptingInvite = $state(false)

  function updateName(event: Event) {
    const target = event.currentTarget as HTMLInputElement
    const name = target.value
    draft = {
      ...draft,
      name,
      slug: slugDirty ? draft.slug : slugFromName(name),
    }
  }

  function updateSlug(event: Event) {
    const target = event.currentTarget as HTMLInputElement
    slugDirty = true
    draft = { ...draft, slug: target.value }
  }

  async function handleCreateOrganization() {
    const parsed = parseOrganizationDraft(draft)
    if (!parsed.ok) {
      toastStore.error(parsed.error)
      return
    }

    saving = true

    try {
      const payload = await createOrganization(parsed.value)
      await goto(organizationPath(payload.organization.id))
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create organization.',
      )
    } finally {
      saving = false
    }
  }

  async function handleAcceptInvitation() {
    const token = inviteToken.trim()
    if (!token) {
      toastStore.error('Invitation token is required.')
      return
    }

    acceptingInvite = true

    try {
      const membership = await acceptOrganizationInvitation(token)
      inviteToken = ''
      await invalidateAll()
      toastStore.success(`Invitation accepted as ${membership.role}.`)
      await goto(organizationPath(membership.organizationID))
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : 'Failed to accept organization invitation.',
      )
    } finally {
      acceptingInvite = false
    }
  }
</script>

<div data-testid="route-scroll-container" class="min-h-0 flex-1 overflow-y-auto">
  <div class="mx-auto flex min-h-full w-full max-w-6xl items-center px-6 py-12">
    <div class="grid w-full gap-6 lg:grid-cols-[minmax(0,1.1fr)_24rem]">
      <section class="space-y-6">
        <div class="space-y-3">
          <p class="text-muted-foreground text-xs tracking-[0.28em] uppercase">
            Workspace bootstrap
          </p>
          <div class="space-y-2">
            <h1 class="text-foreground text-3xl font-semibold tracking-tight">
              Create your first organization
            </h1>
            <p class="text-muted-foreground max-w-2xl text-sm leading-6">
              Start by defining a stable workspace slug. Once the organization exists, the dashboard
              unlocks project creation, provider setup, and machine routing under the same URL
              scope.
            </p>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-3">
          <Card.Root class="rounded-2xl">
            <Card.Header class="pb-3">
              <Card.Title class="text-sm">1. Create organization</Card.Title>
              <Card.Description>
                Reserve the workspace slug that all org-scoped routes will use.
              </Card.Description>
            </Card.Header>
          </Card.Root>

          <Card.Root class="rounded-2xl">
            <Card.Header class="pb-3">
              <Card.Title class="text-sm">2. Add providers</Card.Title>
              <Card.Description>
                Register LLM adapters, CLI launch commands, and auth config per organization.
              </Card.Description>
            </Card.Header>
          </Card.Root>

          <Card.Root class="rounded-2xl">
            <Card.Header class="pb-3">
              <Card.Title class="text-sm">3. Open projects</Card.Title>
              <Card.Description>
                Create a project, wire machines, and move into board, tickets, and agents.
              </Card.Description>
            </Card.Header>
          </Card.Root>
        </div>
      </section>

      <div class="space-y-4">
        <Card.Root class="rounded-2xl">
          <Card.Header>
            <Card.Title>Create organization</Card.Title>
            <Card.Description>
              This writes the initial workspace record through the live catalog API.
            </Card.Description>
          </Card.Header>

          <Card.Content>
            <form
              class="space-y-4"
              onsubmit={(event) => {
                event.preventDefault()
                void handleCreateOrganization()
              }}
            >
              <div class="space-y-2">
                <Label for="organization-name">Organization name</Label>
                <Input
                  id="organization-name"
                  value={draft.name}
                  placeholder="Better & Better"
                  oninput={updateName}
                />
              </div>

              <div class="space-y-2">
                <Label for="organization-slug">Slug</Label>
                <Input
                  id="organization-slug"
                  value={draft.slug}
                  placeholder="better-and-better"
                  oninput={updateSlug}
                />
                <p class="text-muted-foreground text-xs">
                  Use lowercase letters, numbers, and hyphens. This becomes the stable org route
                  handle.
                </p>
              </div>

              <Button type="submit" class="w-full" disabled={saving || acceptingInvite}>
                {saving ? 'Creating…' : 'Create organization'}
              </Button>
            </form>
          </Card.Content>
        </Card.Root>

        <Card.Root class="rounded-2xl border-sky-200/80 bg-sky-50/40">
          <Card.Header>
            <Card.Title>Accept invitation</Card.Title>
            <Card.Description>
              Paste an invitation token to activate your membership and open the organization
              dashboard.
            </Card.Description>
          </Card.Header>

          <Card.Content>
            <form
              class="space-y-4"
              onsubmit={(event) => {
                event.preventDefault()
                void handleAcceptInvitation()
              }}
            >
              <div class="space-y-2">
                <Label for="organization-invite-token">Invitation token</Label>
                <Input
                  id="organization-invite-token"
                  value={inviteToken}
                  placeholder="Paste the token shared by an owner or admin"
                  oninput={(event) => {
                    inviteToken = (event.currentTarget as HTMLInputElement).value
                  }}
                />
                <p class="text-muted-foreground text-xs">
                  Tokens are rotated on resend and stop working once canceled, accepted, or expired.
                </p>
              </div>

              <Button
                type="submit"
                variant="secondary"
                class="w-full"
                disabled={saving || acceptingInvite}
              >
                {acceptingInvite ? 'Joining…' : 'Join organization'}
              </Button>
            </form>
          </Card.Content>
        </Card.Root>
      </div>
    </div>
  </div>
</div>
