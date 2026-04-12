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
  import { i18nStore } from '$lib/i18n/store.svelte'

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
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('catalog.workspace.errors.createOrganization'),
      )
    } finally {
      saving = false
    }
  }

  async function handleAcceptInvitation() {
    const token = inviteToken.trim()
    if (!token) {
      toastStore.error(i18nStore.t('catalog.workspace.errors.invitationTokenRequired'))
      return
    }

    acceptingInvite = true

    try {
      const membership = await acceptOrganizationInvitation(token)
      inviteToken = ''
      await invalidateAll()
      toastStore.success(
        i18nStore.t('catalog.workspace.success.invitationAccepted', {
          role: membership.role,
        }),
      )
      await goto(organizationPath(membership.organizationID))
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('catalog.workspace.errors.acceptInvitation'),
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
            {i18nStore.t('catalog.workspace.hero.badge')}
          </p>
          <div class="space-y-2">
            <h1 class="text-foreground text-3xl font-semibold tracking-tight">
              {i18nStore.t('catalog.workspace.hero.heading')}
            </h1>
            <p class="text-muted-foreground max-w-2xl text-sm leading-6">
              {i18nStore.t('catalog.workspace.hero.description')}
            </p>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-3">
          <Card.Root class="rounded-2xl">
            <Card.Header class="pb-3">
              <Card.Title class="text-sm">
                {i18nStore.t('catalog.workspace.steps.create.title')}
              </Card.Title>
              <Card.Description>
                {i18nStore.t('catalog.workspace.steps.create.description')}
              </Card.Description>
            </Card.Header>
          </Card.Root>

          <Card.Root class="rounded-2xl">
            <Card.Header class="pb-3">
              <Card.Title class="text-sm">
                {i18nStore.t('catalog.workspace.steps.providers.title')}
              </Card.Title>
              <Card.Description>
                {i18nStore.t('catalog.workspace.steps.providers.description')}
              </Card.Description>
            </Card.Header>
          </Card.Root>

          <Card.Root class="rounded-2xl">
            <Card.Header class="pb-3">
              <Card.Title class="text-sm">
                {i18nStore.t('catalog.workspace.steps.projects.title')}
              </Card.Title>
              <Card.Description>
                {i18nStore.t('catalog.workspace.steps.projects.description')}
              </Card.Description>
            </Card.Header>
          </Card.Root>
        </div>
      </section>

      <div class="space-y-4">
        <Card.Root class="rounded-2xl">
          <Card.Header>
            <Card.Title>
              {i18nStore.t('catalog.workspace.create.title')}
            </Card.Title>
            <Card.Description>
              {i18nStore.t('catalog.workspace.create.description')}
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
                <Label for="organization-name">
                  {i18nStore.t('catalog.workspace.create.labels.organizationName')}
                </Label>
                <Input
                  id="organization-name"
                  value={draft.name}
                  placeholder={i18nStore.t('catalog.workspace.create.placeholders.organizationName')}
                  oninput={updateName}
                />
              </div>

              <div class="space-y-2">
                <Label for="organization-slug">
                  {i18nStore.t('catalog.workspace.create.labels.slug')}
                </Label>
                <Input
                  id="organization-slug"
                  value={draft.slug}
                  placeholder={i18nStore.t('catalog.workspace.create.placeholders.slug')}
                  oninput={updateSlug}
                />
                <p class="text-muted-foreground text-xs">
                  {i18nStore.t('catalog.workspace.create.slugHint')}
                </p>
              </div>

              <Button type="submit" class="w-full" disabled={saving || acceptingInvite}>
                {saving
                  ? i18nStore.t('catalog.workspace.create.actions.creating')
                  : i18nStore.t('catalog.workspace.create.actions.create')}
              </Button>
            </form>
          </Card.Content>
        </Card.Root>

        <Card.Root class="rounded-2xl border-sky-200/80 bg-sky-50/40">
          <Card.Header>
            <Card.Title>
              {i18nStore.t('catalog.workspace.invite.title')}
            </Card.Title>
            <Card.Description>
              {i18nStore.t('catalog.workspace.invite.description')}
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
                <Label for="organization-invite-token">
                  {i18nStore.t('catalog.workspace.invite.labels.token')}
                </Label>
                <Input
                  id="organization-invite-token"
                  value={inviteToken}
                  placeholder={i18nStore.t('catalog.workspace.invite.placeholders.token')}
                  oninput={(event) => {
                    inviteToken = (event.currentTarget as HTMLInputElement).value
                  }}
                />
                  <p class="text-muted-foreground text-xs">
                    {i18nStore.t('catalog.workspace.invite.hint.tokens')}
                  </p>
              </div>

              <Button
                type="submit"
                variant="secondary"
                class="w-full"
                disabled={saving || acceptingInvite}
              >
                {acceptingInvite
                  ? i18nStore.t('catalog.workspace.invite.actions.joining')
                  : i18nStore.t('catalog.workspace.invite.actions.join')}
              </Button>
            </form>
          </Card.Content>
        </Card.Root>
      </div>
    </div>
  </div>
</div>
