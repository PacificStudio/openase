<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Archive,
		Building2,
		FolderKanban,
		LoaderCircle,
		Plus,
		Rocket,
		Save,
		Sparkles
	} from '@lucide/svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle
	} from '$lib/components/ui/card';

	type Organization = {
		id: string;
		name: string;
		slug: string;
		default_agent_provider_id?: string | null;
	};

	type Project = {
		id: string;
		organization_id: string;
		name: string;
		slug: string;
		description: string;
		status: 'planning' | 'active' | 'paused' | 'archived';
		default_workflow_id?: string | null;
		default_agent_provider_id?: string | null;
		max_concurrent_agents: number;
	};

	type OrganizationPayload = { organizations: Organization[] };
	type ProjectPayload = { projects: Project[] };

	type OrganizationForm = {
		name: string;
		slug: string;
	};

	type ProjectForm = {
		name: string;
		slug: string;
		description: string;
		status: Project['status'];
		maxConcurrentAgents: number;
	};

	const projectStatuses: Project['status'][] = ['planning', 'active', 'paused', 'archived'];
	const inputClass =
		'w-full rounded-2xl border border-border/70 bg-background/80 px-4 py-3 text-sm outline-none transition focus:border-foreground/40 focus:ring-2 focus:ring-foreground/10';
	const textAreaClass =
		'min-h-32 w-full rounded-2xl border border-border/70 bg-background/80 px-4 py-3 text-sm outline-none transition focus:border-foreground/40 focus:ring-2 focus:ring-foreground/10';

	let booting = $state(true);
	let orgBusy = $state(false);
	let projectBusy = $state(false);
	let organizations = $state<Organization[]>([]);
	let projects = $state<Project[]>([]);
	let selectedOrgId = $state('');
	let selectedProjectId = $state('');
	let selectedOrg = $state<Organization | null>(null);
	let selectedProject = $state<Project | null>(null);
	let notice = $state('');
	let errorMessage = $state('');

	let createOrgForm = $state<OrganizationForm>({
		name: '',
		slug: ''
	});
	let editOrgForm = $state<OrganizationForm>({
		name: '',
		slug: ''
	});
	let createProjectForm = $state<ProjectForm>({
		name: '',
		slug: '',
		description: '',
		status: 'planning',
		maxConcurrentAgents: 5
	});
	let editProjectForm = $state<ProjectForm>({
		name: '',
		slug: '',
		description: '',
		status: 'planning',
		maxConcurrentAgents: 5
	});

	onMount(() => {
		void bootstrap();
	});

	async function bootstrap() {
		booting = true;
		errorMessage = '';
		try {
			await loadOrganizations();
		} catch (error) {
			errorMessage = toErrorMessage(error);
		} finally {
			booting = false;
		}
	}

	async function loadOrganizations(preferredOrgId?: string) {
		const payload = await api<OrganizationPayload>('/api/v1/orgs');
		organizations = payload.organizations;

		const nextOrg =
			organizations.find((item) => item.id === preferredOrgId) ??
			organizations.find((item) => item.id === selectedOrgId) ??
			organizations[0] ??
			null;

		if (!nextOrg) {
			selectedOrgId = '';
			selectedOrg = null;
			editOrgForm = { name: '', slug: '' };
			projects = [];
			selectedProjectId = '';
			selectedProject = null;
			editProjectForm = defaultProjectForm();
			return;
		}

		selectedOrgId = nextOrg.id;
		selectedOrg = nextOrg;
		editOrgForm = toOrganizationForm(nextOrg);
		await loadProjects(nextOrg.id);
	}

	async function loadProjects(orgId: string, preferredProjectId?: string) {
		const payload = await api<ProjectPayload>(`/api/v1/orgs/${orgId}/projects`);
		projects = payload.projects;

		const nextProject =
			projects.find((item) => item.id === preferredProjectId) ??
			projects.find((item) => item.id === selectedProjectId) ??
			projects[0] ??
			null;

		if (!nextProject) {
			selectedProjectId = '';
			selectedProject = null;
			editProjectForm = defaultProjectForm();
			return;
		}

		selectedProjectId = nextProject.id;
		selectedProject = nextProject;
		editProjectForm = toProjectForm(nextProject);
	}

	async function selectOrganization(org: Organization) {
		if (org.id === selectedOrgId) {
			return;
		}

		errorMessage = '';
		selectedOrgId = org.id;
		selectedOrg = org;
		editOrgForm = toOrganizationForm(org);
		selectedProjectId = '';
		selectedProject = null;
		editProjectForm = defaultProjectForm();
		try {
			await loadProjects(org.id);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		}
	}

	function selectProject(project: Project) {
		selectedProjectId = project.id;
		selectedProject = project;
		editProjectForm = toProjectForm(project);
		errorMessage = '';
	}

	async function createOrganization() {
		orgBusy = true;
		errorMessage = '';
		notice = '';
		try {
			const payload = await api<{ organization: Organization }>('/api/v1/orgs', {
				method: 'POST',
				body: JSON.stringify({
					name: createOrgForm.name,
					slug: createOrgForm.slug
				})
			});
			createOrgForm = { name: '', slug: '' };
			notice = `Organization ${payload.organization.name} created`;
			await loadOrganizations(payload.organization.id);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		} finally {
			orgBusy = false;
		}
	}

	async function updateOrganization() {
		if (!selectedOrg) {
			return;
		}

		orgBusy = true;
		errorMessage = '';
		notice = '';
		try {
			await api<{ organization: Organization }>(`/api/v1/orgs/${selectedOrg.id}`, {
				method: 'PATCH',
				body: JSON.stringify({
					name: editOrgForm.name,
					slug: editOrgForm.slug
				})
			});
			notice = `Organization ${editOrgForm.name} updated`;
			await loadOrganizations(selectedOrg.id);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		} finally {
			orgBusy = false;
		}
	}

	async function createProject() {
		if (!selectedOrg) {
			return;
		}

		projectBusy = true;
		errorMessage = '';
		notice = '';
		try {
			const payload = await api<{ project: Project }>(`/api/v1/orgs/${selectedOrg.id}/projects`, {
				method: 'POST',
				body: JSON.stringify({
					name: createProjectForm.name,
					slug: createProjectForm.slug,
					description: createProjectForm.description,
					status: createProjectForm.status,
					max_concurrent_agents: createProjectForm.maxConcurrentAgents
				})
			});
			createProjectForm = defaultProjectForm();
			notice = `Project ${payload.project.name} created`;
			await loadProjects(selectedOrg.id, payload.project.id);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		} finally {
			projectBusy = false;
		}
	}

	async function updateProject() {
		if (!selectedProject || !selectedOrg) {
			return;
		}

		projectBusy = true;
		errorMessage = '';
		notice = '';
		try {
			await api<{ project: Project }>(`/api/v1/projects/${selectedProject.id}`, {
				method: 'PATCH',
				body: JSON.stringify({
					name: editProjectForm.name,
					slug: editProjectForm.slug,
					description: editProjectForm.description,
					status: editProjectForm.status,
					max_concurrent_agents: editProjectForm.maxConcurrentAgents
				})
			});
			notice = `Project ${editProjectForm.name} updated`;
			await loadProjects(selectedOrg.id, selectedProject.id);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		} finally {
			projectBusy = false;
		}
	}

	async function archiveProject() {
		if (!selectedProject || !selectedOrg) {
			return;
		}

		projectBusy = true;
		errorMessage = '';
		notice = '';
		try {
			await api<{ project: Project }>(`/api/v1/projects/${selectedProject.id}`, {
				method: 'DELETE'
			});
			notice = `Project ${selectedProject.name} archived`;
			await loadProjects(selectedOrg.id, selectedProject.id);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		} finally {
			projectBusy = false;
		}
	}

	function fillOrgSlug() {
		if (!createOrgForm.slug) {
			createOrgForm = { ...createOrgForm, slug: slugify(createOrgForm.name) };
		}
	}

	function fillProjectSlug() {
		if (!createProjectForm.slug) {
			createProjectForm = { ...createProjectForm, slug: slugify(createProjectForm.name) };
		}
	}

	function fillEditOrgSlug() {
		if (!editOrgForm.slug) {
			editOrgForm = { ...editOrgForm, slug: slugify(editOrgForm.name) };
		}
	}

	function fillEditProjectSlug() {
		if (!editProjectForm.slug) {
			editProjectForm = { ...editProjectForm, slug: slugify(editProjectForm.name) };
		}
	}

	function slugify(value: string) {
		return value
			.trim()
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '');
	}

	function toOrganizationForm(item: Organization): OrganizationForm {
		return {
			name: item.name,
			slug: item.slug
		};
	}

	function toProjectForm(item: Project): ProjectForm {
		return {
			name: item.name,
			slug: item.slug,
			description: item.description,
			status: item.status,
			maxConcurrentAgents: item.max_concurrent_agents
		};
	}

	function defaultProjectForm(): ProjectForm {
		return {
			name: '',
			slug: '',
			description: '',
			status: 'planning',
			maxConcurrentAgents: 5
		};
	}

	async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
		const headers = new Headers(init.headers);
		if (init.body && !headers.has('content-type')) {
			headers.set('content-type', 'application/json');
		}

		const response = await fetch(path, {
			...init,
			headers
		});

		const payload = (await response.json().catch(() => ({}))) as { error?: string };
		if (!response.ok) {
			throw new Error(payload.error ?? `request failed with status ${response.status}`);
		}

		return payload as T;
	}

	function toErrorMessage(error: unknown) {
		if (error instanceof Error) {
			return error.message;
		}

		return 'Request failed';
	}
</script>

<svelte:head>
	<title>OpenASE Org / Project Control Plane</title>
	<meta
		name="description"
		content="OpenASE Org and Project CRUD workspace backed by the embedded Go API."
	/>
</svelte:head>

<div class="relative overflow-hidden">
	<div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(218,165,32,0.18),transparent_26rem),radial-gradient(circle_at_bottom_right,rgba(15,118,110,0.14),transparent_28rem)]"></div>

	<section class="relative mx-auto flex min-h-screen w-full max-w-7xl flex-col gap-8 px-6 py-8 sm:px-8 lg:px-10">
		<div class="grid gap-4 lg:grid-cols-[minmax(0,1.2fr)_minmax(18rem,0.8fr)]">
			<div class="space-y-5">
				<div class="flex flex-wrap items-center gap-3">
					<Badge variant="outline">F04 vertical slice</Badge>
					<Badge variant="outline">API + Web UI</Badge>
					<Badge variant="outline">main</Badge>
				</div>

				<div class="space-y-4">
					<p class="text-sm font-medium uppercase tracking-[0.32em] text-muted-foreground">
						OpenASE control plane
					</p>
					<h1 class="max-w-4xl text-5xl leading-none font-semibold tracking-[-0.06em] text-balance sm:text-6xl">
						Organizations and projects now ship as a working management surface.
					</h1>
					<p class="max-w-3xl text-lg leading-8 text-muted-foreground">
						This slice keeps the scope narrow on purpose: create, update, inspect, and archive
						the first two core resources directly inside the embedded UI, backed by the Go API
						and Postgres schema.
					</p>
				</div>
			</div>

			<div class="grid gap-4 sm:grid-cols-3 lg:grid-cols-1">
				<Card class="border-border/80 bg-background/75 backdrop-blur">
					<CardHeader class="pb-3">
						<CardDescription>Organizations</CardDescription>
						<CardTitle class="text-4xl tracking-[-0.05em]">{organizations.length}</CardTitle>
					</CardHeader>
				</Card>
				<Card class="border-border/80 bg-background/75 backdrop-blur">
					<CardHeader class="pb-3">
						<CardDescription>Visible projects</CardDescription>
						<CardTitle class="text-4xl tracking-[-0.05em]">{projects.length}</CardTitle>
					</CardHeader>
				</Card>
				<Card class="border-border/80 bg-background/75 backdrop-blur">
					<CardHeader class="pb-3">
						<CardDescription>Active in selected org</CardDescription>
						<CardTitle class="text-4xl tracking-[-0.05em]">
							{projects.filter((item) => item.status === 'active').length}
						</CardTitle>
					</CardHeader>
				</Card>
			</div>
		</div>

		{#if notice}
			<div class="rounded-3xl border border-emerald-500/30 bg-emerald-500/10 px-5 py-4 text-sm text-emerald-950">
				{notice}
			</div>
		{/if}

		{#if errorMessage}
			<div class="rounded-3xl border border-destructive/25 bg-destructive/10 px-5 py-4 text-sm text-destructive">
				{errorMessage}
			</div>
		{/if}

		{#if booting}
			<div class="flex min-h-96 items-center justify-center rounded-[2rem] border border-border/80 bg-background/70">
				<div class="flex items-center gap-3 text-sm text-muted-foreground">
					<LoaderCircle class="size-4 animate-spin" />
					<span>Loading Org / Project control plane…</span>
				</div>
			</div>
		{:else}
			<div class="grid gap-6 xl:grid-cols-[22rem_minmax(0,1fr)]">
				<div class="space-y-6">
					<Card class="border-border/80 bg-background/80 backdrop-blur">
						<CardHeader>
							<div class="flex items-center justify-between gap-4">
								<div>
									<CardTitle class="flex items-center gap-2">
										<Building2 class="size-4" />
										<span>Organizations</span>
									</CardTitle>
									<CardDescription>Pick the workspace boundary before creating projects.</CardDescription>
								</div>
								<Badge variant="outline">{organizations.length}</Badge>
							</div>
						</CardHeader>
						<CardContent class="space-y-3">
							{#if organizations.length === 0}
								<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
									No organizations yet. Create the first one to unlock project CRUD.
								</div>
							{:else}
								{#each organizations as org}
									<button
										type="button"
										class={`w-full rounded-3xl border px-4 py-4 text-left transition ${
											org.id === selectedOrgId
												? 'border-foreground/30 bg-foreground text-background shadow-lg shadow-black/10'
												: 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
										}`}
										onclick={() => selectOrganization(org)}
									>
										<div class="flex items-start justify-between gap-4">
											<div>
												<p class="text-sm font-semibold">{org.name}</p>
												<p
													class={`mt-1 font-mono text-xs ${
														org.id === selectedOrgId ? 'text-background/75' : 'text-muted-foreground'
													}`}
												>
													/{org.slug}
												</p>
											</div>
											<Badge variant={org.id === selectedOrgId ? 'secondary' : 'outline'}>
												selected
											</Badge>
										</div>
									</button>
								{/each}
							{/if}
						</CardContent>
					</Card>

					<Card class="border-border/80 bg-background/80 backdrop-blur">
						<CardHeader>
							<CardTitle class="flex items-center gap-2">
								<Plus class="size-4" />
								<span>Create organization</span>
							</CardTitle>
							<CardDescription>Lowercase slugs keep URLs clean and API-safe.</CardDescription>
						</CardHeader>
						<CardContent>
							<form
								class="space-y-4"
								onsubmit={(event) => {
									event.preventDefault();
									void createOrganization();
								}}
							>
								<div class="space-y-2">
									<label
										class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
										for="create-org-name"
									>
										Name
									</label>
									<input
										id="create-org-name"
										class={inputClass}
										bind:value={createOrgForm.name}
										placeholder="Acme Control"
										onblur={fillOrgSlug}
									/>
								</div>
								<div class="space-y-2">
									<label
										class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
										for="create-org-slug"
									>
										Slug
									</label>
									<input
										id="create-org-slug"
										class={inputClass}
										bind:value={createOrgForm.slug}
										placeholder="acme-control"
									/>
								</div>
								<Button class="w-full" type="submit" disabled={orgBusy}>
									{#if orgBusy}
										<LoaderCircle class="mr-2 size-4 animate-spin" />
									{:else}
										<Plus class="mr-2 size-4" />
									{/if}
									Create organization
								</Button>
							</form>
						</CardContent>
					</Card>
				</div>

				<div class="space-y-6">
					<Card class="overflow-hidden border-border/80 bg-background/80 backdrop-blur">
						<CardHeader class="border-b border-border/70 bg-muted/25">
							<div class="flex flex-wrap items-center justify-between gap-4">
								<div>
									<CardTitle class="flex items-center gap-2">
										<Sparkles class="size-4" />
										<span>Selected organization</span>
									</CardTitle>
									<CardDescription>
										Update the org surface, then use it as the parent boundary for projects.
									</CardDescription>
								</div>
								{#if selectedOrg}
									<Badge variant="outline">/{selectedOrg.slug}</Badge>
								{/if}
							</div>
						</CardHeader>
						<CardContent class="p-6">
							{#if selectedOrg}
								<form
									class="grid gap-4 md:grid-cols-2"
									onsubmit={(event) => {
										event.preventDefault();
										void updateOrganization();
									}}
								>
									<div class="space-y-2">
										<label
											class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
											for="edit-org-name"
										>
											Name
										</label>
										<input
											id="edit-org-name"
											class={inputClass}
											bind:value={editOrgForm.name}
											onblur={fillEditOrgSlug}
										/>
									</div>
									<div class="space-y-2">
										<label
											class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
											for="edit-org-slug"
										>
											Slug
										</label>
										<input id="edit-org-slug" class={inputClass} bind:value={editOrgForm.slug} />
									</div>
									<div class="md:col-span-2">
										<Button type="submit" disabled={orgBusy}>
											{#if orgBusy}
												<LoaderCircle class="mr-2 size-4 animate-spin" />
											{:else}
												<Save class="mr-2 size-4" />
											{/if}
											Save organization
										</Button>
									</div>
								</form>
							{:else}
								<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
									Select or create an organization to continue.
								</div>
							{/if}
						</CardContent>
					</Card>

					<div class="grid gap-6 2xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
						<Card class="border-border/80 bg-background/80 backdrop-blur">
							<CardHeader>
								<div class="flex items-center justify-between gap-4">
								<div>
									<CardTitle class="flex items-center gap-2">
										<FolderKanban class="size-4" />
										<span>Project roster</span>
									</CardTitle>
										<CardDescription>
											Select a project to edit, or archive it when the workstream ends.
										</CardDescription>
									</div>
									<Badge variant="outline">{projects.length}</Badge>
								</div>
							</CardHeader>
							<CardContent class="space-y-3">
								{#if !selectedOrg}
									<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
										Projects unlock after you select an organization.
									</div>
								{:else if projects.length === 0}
									<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
										No projects in this organization yet.
									</div>
								{:else}
									{#each projects as project}
										<button
											type="button"
											class={`w-full rounded-3xl border px-4 py-4 text-left transition ${
												project.id === selectedProjectId
													? 'border-emerald-700/35 bg-emerald-950 text-white shadow-lg shadow-emerald-950/20'
													: 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
											}`}
											onclick={() => selectProject(project)}
										>
											<div class="flex items-start justify-between gap-4">
												<div>
													<p class="text-sm font-semibold">{project.name}</p>
													<p
														class={`mt-1 font-mono text-xs ${
															project.id === selectedProjectId
																? 'text-white/70'
																: 'text-muted-foreground'
														}`}
													>
														/{project.slug}
													</p>
												</div>
												<Badge variant={project.id === selectedProjectId ? 'secondary' : 'outline'}>
													{project.status}
												</Badge>
											</div>
											<p
												class={`mt-3 text-sm leading-6 ${
													project.id === selectedProjectId
														? 'text-white/80'
														: 'text-muted-foreground'
												}`}
											>
												{project.description || 'No description yet.'}
											</p>
										</button>
									{/each}
								{/if}
							</CardContent>
						</Card>

						<div class="space-y-6">
							<Card class="border-border/80 bg-background/80 backdrop-blur">
								<CardHeader>
									<CardTitle class="flex items-center gap-2">
										<Rocket class="size-4" />
										<span>Create project</span>
									</CardTitle>
									<CardDescription>
										A minimal project record carries routing metadata and runtime concurrency limits.
									</CardDescription>
								</CardHeader>
								<CardContent>
									<form
										class="space-y-4"
										onsubmit={(event) => {
											event.preventDefault();
											void createProject();
										}}
									>
										<div class="grid gap-4 md:grid-cols-2">
											<div class="space-y-2">
												<label
													class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
													for="create-project-name"
												>
													Name
												</label>
												<input
													id="create-project-name"
													class={inputClass}
													bind:value={createProjectForm.name}
													placeholder="Control plane"
													onblur={fillProjectSlug}
													disabled={!selectedOrg}
												/>
											</div>
											<div class="space-y-2">
												<label
													class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
													for="create-project-slug"
												>
													Slug
												</label>
												<input
													id="create-project-slug"
													class={inputClass}
													bind:value={createProjectForm.slug}
													placeholder="control-plane"
													disabled={!selectedOrg}
												/>
											</div>
										</div>
										<div class="grid gap-4 md:grid-cols-[minmax(0,1fr)_12rem]">
											<div class="space-y-2">
												<label
													class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
													for="create-project-status"
												>
													Status
												</label>
												<select
													id="create-project-status"
													class={inputClass}
													bind:value={createProjectForm.status}
													disabled={!selectedOrg}
												>
													{#each projectStatuses as status}
														<option value={status}>{status}</option>
													{/each}
												</select>
											</div>
											<div class="space-y-2">
												<label
													class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
													for="create-project-max"
												>
													Max agents
												</label>
												<input
													id="create-project-max"
													class={inputClass}
													bind:value={createProjectForm.maxConcurrentAgents}
													min="1"
													type="number"
													disabled={!selectedOrg}
												/>
											</div>
										</div>
										<div class="space-y-2">
											<label
												class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
												for="create-project-description"
											>
												Description
											</label>
											<textarea
												id="create-project-description"
												class={textAreaClass}
												bind:value={createProjectForm.description}
												placeholder="What this project owns, how it fits into the platform, and what should be automated next."
												disabled={!selectedOrg}
											></textarea>
										</div>
										<Button class="w-full" type="submit" disabled={!selectedOrg || projectBusy}>
											{#if projectBusy}
												<LoaderCircle class="mr-2 size-4 animate-spin" />
											{:else}
												<Plus class="mr-2 size-4" />
											{/if}
											Create project
										</Button>
									</form>
								</CardContent>
							</Card>

							<Card class="border-border/80 bg-background/80 backdrop-blur">
								<CardHeader>
									<CardTitle class="flex items-center gap-2">
										<Save class="size-4" />
										<span>Selected project</span>
									</CardTitle>
									<CardDescription>
										Keep the project metadata sharp, or archive it without deleting history.
									</CardDescription>
								</CardHeader>
								<CardContent>
									{#if selectedProject}
										<form
											class="space-y-4"
											onsubmit={(event) => {
												event.preventDefault();
												void updateProject();
											}}
										>
											<div class="grid gap-4 md:grid-cols-2">
												<div class="space-y-2">
													<label
														class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
														for="edit-project-name"
													>
														Name
													</label>
													<input
														id="edit-project-name"
														class={inputClass}
														bind:value={editProjectForm.name}
														onblur={fillEditProjectSlug}
													/>
												</div>
												<div class="space-y-2">
													<label
														class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
														for="edit-project-slug"
													>
														Slug
													</label>
													<input id="edit-project-slug" class={inputClass} bind:value={editProjectForm.slug} />
												</div>
											</div>
											<div class="grid gap-4 md:grid-cols-[minmax(0,1fr)_12rem]">
												<div class="space-y-2">
													<label
														class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
														for="edit-project-status"
													>
														Status
													</label>
													<select id="edit-project-status" class={inputClass} bind:value={editProjectForm.status}>
														{#each projectStatuses as status}
															<option value={status}>{status}</option>
														{/each}
													</select>
												</div>
												<div class="space-y-2">
													<label
														class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
														for="edit-project-max"
													>
														Max agents
													</label>
													<input
														id="edit-project-max"
														class={inputClass}
														bind:value={editProjectForm.maxConcurrentAgents}
														min="1"
														type="number"
													/>
												</div>
											</div>
											<div class="space-y-2">
												<label
													class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
													for="edit-project-description"
												>
													Description
												</label>
												<textarea
													id="edit-project-description"
													class={textAreaClass}
													bind:value={editProjectForm.description}
												></textarea>
											</div>
											<div class="flex flex-wrap gap-3">
												<Button type="submit" disabled={projectBusy}>
													{#if projectBusy}
														<LoaderCircle class="mr-2 size-4 animate-spin" />
													{:else}
														<Save class="mr-2 size-4" />
													{/if}
													Save project
												</Button>
												<Button
													type="button"
													variant="outline"
													disabled={projectBusy}
													onclick={archiveProject}
												>
													<Archive class="mr-2 size-4" />
													Archive project
												</Button>
											</div>
										</form>
									{:else}
										<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
											Select a project from the roster to edit or archive it.
										</div>
									{/if}
								</CardContent>
							</Card>
						</div>
					</div>
				</div>
			</div>
		{/if}
	</section>
</div>
