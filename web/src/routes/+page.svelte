<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Activity,
		Archive,
		Bot,
		Building2,
		Cable,
		FolderKanban,
		HeartPulse,
		LoaderCircle,
		Plus,
		Rocket,
		Save,
		Sparkles
	} from '@lucide/svelte';
	import { connectEventStream, type SSEFrame, type StreamConnectionState } from '$lib/api/sse';
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

	type Agent = {
		id: string;
		provider_id: string;
		project_id: string;
		name: string;
		status: 'idle' | 'claimed' | 'running' | 'failed' | 'terminated';
		current_ticket_id?: string | null;
		session_id: string;
		workspace_path: string;
		capabilities: string[];
		total_tokens_used: number;
		total_tickets_completed: number;
		last_heartbeat_at?: string | null;
	};

	type ActivityEvent = {
		id: string;
		project_id: string;
		ticket_id?: string | null;
		agent_id?: string | null;
		event_type: string;
		message: string;
		metadata: Record<string, unknown>;
		created_at: string;
	};

	type OrganizationPayload = { organizations: Organization[] };
	type ProjectPayload = { projects: Project[] };
	type AgentPayload = { agents: Agent[] };
	type ActivityPayload = { events: ActivityEvent[] };
	type StreamEnvelope = {
		topic: string;
		type: string;
		payload?: unknown;
		published_at: string;
	};

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

	const agentConsoleLimit = 40;
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
	let agents = $state<Agent[]>([]);
	let activityEvents = $state<ActivityEvent[]>([]);
	let selectedAgentId = $state('');
	let notice = $state('');
	let errorMessage = $state('');
	let agentConsoleBusy = $state(false);
	let agentConsoleError = $state('');
	let agentStreamState = $state<StreamConnectionState>('idle');
	let activityStreamState = $state<StreamConnectionState>('idle');
	let heartbeatNow = $state(Date.now());

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
		const timer = window.setInterval(() => {
			heartbeatNow = Date.now();
		}, 15000);
		void bootstrap();

		return () => {
			window.clearInterval(timer);
		};
	});

	$effect(() => {
		const projectId = selectedProjectId;
		if (!projectId) {
			resetAgentConsole();
			return;
		}

		void loadAgents(projectId);
		return connectProjectStreams(projectId);
	});

	$effect(() => {
		const projectId = selectedProjectId;
		if (!projectId) {
			return;
		}

		void loadActivityEvents(projectId, selectedAgentId);
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

	async function loadAgents(projectId: string) {
		agentConsoleBusy = true;
		agentConsoleError = '';

		try {
			const payload = await api<AgentPayload>(`/api/v1/projects/${projectId}/agents`);
			agents = payload.agents;
			selectedAgentId = chooseAgentSelection(payload.agents, selectedAgentId);
		} catch (error) {
			agents = [];
			selectedAgentId = '';
			agentConsoleError = toErrorMessage(error);
		} finally {
			agentConsoleBusy = false;
		}
	}

	async function loadActivityEvents(projectId: string, agentId?: string) {
		agentConsoleError = '';

		try {
			const query = new URLSearchParams({ limit: String(agentConsoleLimit) });
			if (agentId) {
				query.set('agent_id', agentId);
			}

			const payload = await api<ActivityPayload>(`/api/v1/projects/${projectId}/activity?${query.toString()}`);
			activityEvents = payload.events;
		} catch (error) {
			activityEvents = [];
			agentConsoleError = toErrorMessage(error);
		}
	}

	function connectProjectStreams(projectId: string) {
		const closeAgentStream = connectEventStream(`/api/v1/projects/${projectId}/agents/stream`, {
			onEvent: (frame) => handleAgentFrame(projectId, frame),
			onStateChange: (state) => {
				agentStreamState = state;
			},
			onError: (error) => {
				agentConsoleError = toErrorMessage(error);
			}
		});

		const closeActivityStream = connectEventStream(`/api/v1/projects/${projectId}/activity/stream`, {
			onEvent: (frame) => handleActivityFrame(projectId, frame),
			onStateChange: (state) => {
				activityStreamState = state;
			},
			onError: (error) => {
				agentConsoleError = toErrorMessage(error);
			}
		});

		return () => {
			closeAgentStream();
			closeActivityStream();
		};
	}

	function handleAgentFrame(projectId: string, frame: SSEFrame) {
		const envelope = parseStreamEnvelope(frame);
		if (!envelope || projectId !== selectedProjectId) {
			return;
		}

		const patch = parseAgentPatch(envelope.payload);
		if (!patch) {
			void loadAgents(projectId);
			return;
		}

		if (frame.event.includes('heartbeat') && !patch.last_heartbeat_at) {
			patch.last_heartbeat_at = envelope.published_at;
		}

		agents = upsertAgent(agents, patch);
		selectedAgentId = chooseAgentSelection(agents, selectedAgentId);
	}

	function handleActivityFrame(projectId: string, frame: SSEFrame) {
		const envelope = parseStreamEnvelope(frame);
		if (!envelope || projectId !== selectedProjectId) {
			return;
		}

		const activityEvent = parseActivityEvent(envelope.payload, envelope.published_at);
		if (!activityEvent) {
			void loadActivityEvents(projectId, selectedAgentId);
			return;
		}
		if (selectedAgentId && activityEvent.agent_id !== selectedAgentId) {
			return;
		}

		activityEvents = dedupeActivityEvents([activityEvent, ...activityEvents]).slice(0, agentConsoleLimit);
	}

	function resetAgentConsole() {
		agents = [];
		activityEvents = [];
		selectedAgentId = '';
		agentConsoleError = '';
		agentStreamState = 'idle';
		activityStreamState = 'idle';
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

	function chooseAgentSelection(items: Agent[], preferredAgentId: string) {
		if (preferredAgentId && items.some((item) => item.id === preferredAgentId)) {
			return preferredAgentId;
		}

		return (
			items.find((item) => item.status === 'running')?.id ??
			items.find((item) => item.status === 'claimed')?.id ??
			items[0]?.id ??
			''
		);
	}

	function parseStreamEnvelope(frame: SSEFrame): StreamEnvelope | null {
		try {
			return JSON.parse(frame.data) as StreamEnvelope;
		} catch {
			return null;
		}
	}

	function parseAgentPatch(raw: unknown): Partial<Agent> & { id: string } | null {
		const source = unwrapObject(raw, 'agent');
		if (!source) {
			return null;
		}

		const id = readString(source, 'id') ?? readString(source, 'agent_id');
		if (!id) {
			return null;
		}

		const status = readString(source, 'status');
		return compactAgentPatch({
			id,
			provider_id: readString(source, 'provider_id'),
			project_id: readString(source, 'project_id'),
			name: readString(source, 'name'),
			status: isAgentStatus(status) ? status : undefined,
			current_ticket_id: readNullableString(source, 'current_ticket_id'),
			session_id: readString(source, 'session_id'),
			workspace_path: readString(source, 'workspace_path'),
			capabilities: readStringArray(source, 'capabilities'),
			total_tokens_used: readNumber(source, 'total_tokens_used'),
			total_tickets_completed: readNumber(source, 'total_tickets_completed'),
			last_heartbeat_at: readNullableString(source, 'last_heartbeat_at')
		});
	}

	function compactAgentPatch(patch: Partial<Agent> & { id: string }) {
		return Object.fromEntries(
			Object.entries(patch).filter(([, value]) => value !== undefined)
		) as Partial<Agent> & { id: string };
	}

	function upsertAgent(items: Agent[], patch: Partial<Agent> & { id: string }) {
		const index = items.findIndex((item) => item.id === patch.id);
		if (index === -1) {
			if (!patch.name || !patch.project_id || !patch.provider_id || !patch.status) {
				return items;
			}

			return [
				...items,
				{
					id: patch.id,
					provider_id: patch.provider_id,
					project_id: patch.project_id,
					name: patch.name,
					status: patch.status,
					current_ticket_id: patch.current_ticket_id ?? null,
					session_id: patch.session_id ?? '',
					workspace_path: patch.workspace_path ?? '',
					capabilities: patch.capabilities ?? [],
					total_tokens_used: patch.total_tokens_used ?? 0,
					total_tickets_completed: patch.total_tickets_completed ?? 0,
					last_heartbeat_at: patch.last_heartbeat_at ?? null
				}
			].sort((left, right) => left.name.localeCompare(right.name));
		}

		const next = [...items];
		next[index] = { ...items[index], ...patch };
		return next;
	}

	function parseActivityEvent(raw: unknown, fallbackCreatedAt: string): ActivityEvent | null {
		const source = unwrapObject(raw, 'event');
		if (!source) {
			return null;
		}

		const id = readString(source, 'id');
		const projectId = readString(source, 'project_id');
		const eventType = readString(source, 'event_type') ?? readString(source, 'type');
		if (!id || !projectId || !eventType) {
			return null;
		}

		return {
			id,
			project_id: projectId,
			ticket_id: readNullableString(source, 'ticket_id'),
			agent_id: readNullableString(source, 'agent_id'),
			event_type: eventType,
			message: readString(source, 'message') ?? '',
			metadata: readRecord(source, 'metadata') ?? {},
			created_at: readString(source, 'created_at') ?? fallbackCreatedAt
		};
	}

	function dedupeActivityEvents(items: ActivityEvent[]) {
		const seen = new Set<string>();
		return items.filter((item) => {
			if (seen.has(item.id)) {
				return false;
			}
			seen.add(item.id);
			return true;
		});
	}

	function currentAgent() {
		return agents.find((item) => item.id === selectedAgentId) ?? null;
	}

	function runningAgentCount() {
		return agents.filter((item) => item.status === 'running').length;
	}

	function stalledAgentCount() {
		return agents.filter((item) => heartbeatTone(item.last_heartbeat_at) === 'stalled').length;
	}

	function heartbeatLabel(timestamp?: string | null) {
		if (!timestamp) {
			return 'No heartbeat';
		}

		const ageSeconds = heartbeatAgeSeconds(timestamp);
		if (ageSeconds === null) {
			return 'Invalid heartbeat';
		}
		if (ageSeconds < 60) {
			return `${ageSeconds}s ago`;
		}

		return `${Math.floor(ageSeconds / 60)}m ago`;
	}

	function heartbeatTone(timestamp?: string | null) {
		const ageSeconds = heartbeatAgeSeconds(timestamp);
		if (ageSeconds === null) {
			return 'stalled';
		}
		if (ageSeconds <= 60) {
			return 'healthy';
		}
		if (ageSeconds <= 180) {
			return 'warning';
		}
		return 'stalled';
	}

	function heartbeatAgeSeconds(timestamp?: string | null) {
		if (!timestamp) {
			return null;
		}

		const parsed = Date.parse(timestamp);
		if (Number.isNaN(parsed)) {
			return null;
		}

		return Math.max(0, Math.floor((heartbeatNow - parsed) / 1000));
	}

	function heartbeatBadgeClass(timestamp?: string | null) {
		switch (heartbeatTone(timestamp)) {
			case 'healthy':
				return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700';
			case 'warning':
				return 'border-amber-500/25 bg-amber-500/10 text-amber-700';
			default:
				return 'border-rose-500/25 bg-rose-500/10 text-rose-700';
		}
	}

	function streamBadgeClass(state: StreamConnectionState) {
		switch (state) {
			case 'live':
				return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700';
			case 'connecting':
				return 'border-sky-500/25 bg-sky-500/10 text-sky-700';
			case 'retrying':
				return 'border-amber-500/25 bg-amber-500/10 text-amber-700';
			default:
				return 'border-border/80 bg-background text-muted-foreground';
		}
	}

	function formatTimestamp(value: string) {
		const parsed = Date.parse(value);
		if (Number.isNaN(parsed)) {
			return value;
		}

		return new Intl.DateTimeFormat(undefined, {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
			month: 'short',
			day: 'numeric'
		}).format(parsed);
	}

	function selectedAgentName() {
		return currentAgent()?.name ?? 'All agents';
	}

	function unwrapObject(raw: unknown, nestedKey: string) {
		if (!isRecord(raw)) {
			return null;
		}

		const nested = raw[nestedKey];
		if (isRecord(nested)) {
			return nested;
		}

		return raw;
	}

	function readString(source: Record<string, unknown>, key: string) {
		const value = source[key];
		return typeof value === 'string' && value.trim() ? value : undefined;
	}

	function readNullableString(source: Record<string, unknown>, key: string) {
		const value = source[key];
		if (value === null) {
			return null;
		}
		return typeof value === 'string' ? value : undefined;
	}

	function readNumber(source: Record<string, unknown>, key: string) {
		const value = source[key];
		return typeof value === 'number' ? value : undefined;
	}

	function readStringArray(source: Record<string, unknown>, key: string) {
		const value = source[key];
		if (!Array.isArray(value) || !value.every((item) => typeof item === 'string')) {
			return undefined;
		}

		return [...value];
	}

	function readRecord(source: Record<string, unknown>, key: string) {
		const value = source[key];
		return isRecord(value) ? value : undefined;
	}

	function isRecord(value: unknown): value is Record<string, unknown> {
		return typeof value === 'object' && value !== null && !Array.isArray(value);
	}

	function isAgentStatus(value?: string): value is Agent['status'] {
		return value === 'idle' || value === 'claimed' || value === 'running' || value === 'failed' || value === 'terminated';
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
					<Badge variant="outline">F26 vertical slice</Badge>
					<Badge variant="outline">Agent console</Badge>
					<Badge variant="outline">SSE ready</Badge>
					<Badge variant="outline">main</Badge>
				</div>

				<div class="space-y-4">
					<p class="text-sm font-medium uppercase tracking-[0.32em] text-muted-foreground">
						OpenASE control plane
					</p>
					<h1 class="max-w-4xl text-5xl leading-none font-semibold tracking-[-0.06em] text-balance sm:text-6xl">
						The control plane now exposes a live agent console inside the Web UI.
					</h1>
					<p class="max-w-3xl text-lg leading-8 text-muted-foreground">
						This cut stays narrow on purpose: select a project, inspect agent state, follow the
						output feed, and watch heartbeat freshness over the same embedded Go API and SSE
						surface that later workflows will publish into.
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

					<Card class="overflow-hidden border-border/80 bg-background/80 backdrop-blur">
						<CardHeader class="border-b border-border/70 bg-muted/25">
							<div class="flex flex-wrap items-center justify-between gap-4">
								<div>
									<CardTitle class="flex items-center gap-2">
										<Bot class="size-4" />
										<span>Agent console</span>
									</CardTitle>
									<CardDescription>
										Status snapshot, realtime stream state, and heartbeat aging for the selected project.
									</CardDescription>
								</div>
								<div class="flex flex-wrap items-center gap-2">
									<span class={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${streamBadgeClass(agentStreamState)}`}>
										<Cable class="mr-1.5 size-3.5" />
										Agents {agentStreamState}
									</span>
									<span class={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${streamBadgeClass(activityStreamState)}`}>
										<Activity class="mr-1.5 size-3.5" />
										Output {activityStreamState}
									</span>
								</div>
							</div>
						</CardHeader>
						<CardContent class="space-y-6 p-6">
							{#if agentConsoleError}
								<div class="rounded-3xl border border-destructive/25 bg-destructive/10 px-4 py-3 text-sm text-destructive">
									{agentConsoleError}
								</div>
							{/if}

							{#if !selectedProject}
								<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
									Select a project to load agents, output, and heartbeat telemetry.
								</div>
							{:else}
								<div class="grid gap-4 md:grid-cols-3">
									<Card class="border-border/70 bg-background/70">
										<CardHeader class="pb-3">
											<CardDescription>Registered agents</CardDescription>
											<CardTitle class="text-3xl tracking-[-0.05em]">{agents.length}</CardTitle>
										</CardHeader>
									</Card>
									<Card class="border-border/70 bg-background/70">
										<CardHeader class="pb-3">
											<CardDescription>Running now</CardDescription>
											<CardTitle class="text-3xl tracking-[-0.05em]">{runningAgentCount()}</CardTitle>
										</CardHeader>
									</Card>
									<Card class="border-border/70 bg-background/70">
										<CardHeader class="pb-3">
											<CardDescription>Stalled heartbeat</CardDescription>
											<CardTitle class="text-3xl tracking-[-0.05em]">{stalledAgentCount()}</CardTitle>
										</CardHeader>
									</Card>
								</div>

								<div class="grid gap-6 xl:grid-cols-[22rem_minmax(0,1fr)]">
									<div class="space-y-4">
										<div class="flex items-center justify-between gap-3">
											<div>
												<p class="text-sm font-semibold">Agent roster</p>
												<p class="text-sm text-muted-foreground">
													{selectedProject.name} / {selectedProject.slug}
												</p>
											</div>
											{#if agentConsoleBusy}
												<LoaderCircle class="size-4 animate-spin text-muted-foreground" />
											{/if}
										</div>

										{#if agents.length === 0}
											<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
												No agents are registered for this project yet.
											</div>
										{:else}
											<div class="space-y-3">
												{#each agents as agent}
													<button
														type="button"
														class={`w-full rounded-3xl border px-4 py-4 text-left transition ${
															agent.id === selectedAgentId
																? 'border-sky-600/30 bg-sky-950 text-white shadow-lg shadow-sky-950/20'
																: 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
														}`}
														onclick={() => {
															selectedAgentId = agent.id;
														}}
													>
														<div class="flex items-start justify-between gap-4">
															<div>
																<p class="text-sm font-semibold">{agent.name}</p>
																<p class={`mt-1 text-xs ${agent.id === selectedAgentId ? 'text-white/70' : 'text-muted-foreground'}`}>
																	{agent.status} · {heartbeatLabel(agent.last_heartbeat_at)}
																</p>
															</div>
															<span class={`inline-flex items-center rounded-full border px-2.5 py-1 text-[11px] font-medium ${
																agent.id === selectedAgentId ? 'border-white/15 bg-white/10 text-white' : heartbeatBadgeClass(agent.last_heartbeat_at)
															}`}>
																<HeartPulse class="mr-1 size-3" />
																{heartbeatTone(agent.last_heartbeat_at)}
															</span>
														</div>
														<p class={`mt-3 text-sm ${agent.id === selectedAgentId ? 'text-white/80' : 'text-muted-foreground'}`}>
															{agent.capabilities.length > 0 ? agent.capabilities.join(' · ') : 'No capabilities declared'}
														</p>
													</button>
												{/each}
											</div>
										{/if}
									</div>

									<div class="space-y-6">
										<Card class="border-border/70 bg-background/70">
											<CardHeader>
												<div class="flex flex-wrap items-center justify-between gap-4">
													<div>
														<CardTitle class="flex items-center gap-2">
															<Activity class="size-4" />
															<span>{selectedAgentName()}</span>
														</CardTitle>
														<CardDescription>
															Focused agent status, latest heartbeat, and execution coordinates.
														</CardDescription>
													</div>
													{#if currentAgent()}
														<Badge variant="outline">{currentAgent()?.status}</Badge>
													{:else}
														<Badge variant="outline">No agent selected</Badge>
													{/if}
												</div>
											</CardHeader>
											<CardContent>
												{#if currentAgent()}
													<div class="grid gap-4 md:grid-cols-2">
														<div class="rounded-3xl border border-border/70 bg-background px-4 py-4">
															<p class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground">Heartbeat</p>
															<p class="mt-3 text-2xl font-semibold tracking-[-0.04em]">
																{heartbeatLabel(currentAgent()?.last_heartbeat_at)}
															</p>
															<p class="mt-2 text-sm text-muted-foreground">
																Last seen at {formatTimestamp(currentAgent()?.last_heartbeat_at ?? '')}
															</p>
														</div>
														<div class="rounded-3xl border border-border/70 bg-background px-4 py-4">
															<p class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground">Session</p>
															<p class="mt-3 break-all font-mono text-sm">
																{currentAgent()?.session_id || 'No session id yet'}
															</p>
															<p class="mt-2 text-sm text-muted-foreground">
																Tokens {currentAgent()?.total_tokens_used ?? 0} · Completed {currentAgent()?.total_tickets_completed ?? 0}
															</p>
														</div>
														<div class="rounded-3xl border border-border/70 bg-background px-4 py-4 md:col-span-2">
															<p class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground">Workspace</p>
															<p class="mt-3 break-all font-mono text-sm">
																{currentAgent()?.workspace_path || 'Workspace path is not set'}
															</p>
															<p class="mt-2 text-sm text-muted-foreground">
																Current ticket {currentAgent()?.current_ticket_id ?? 'unassigned'}
															</p>
														</div>
													</div>
												{:else}
													<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
														Pick an agent from the roster to focus the console feed.
													</div>
												{/if}
											</CardContent>
										</Card>

										<Card class="border-border/70 bg-background/70">
											<CardHeader>
												<div class="flex flex-wrap items-center justify-between gap-4">
													<div>
														<CardTitle class="flex items-center gap-2">
															<Activity class="size-4" />
															<span>Realtime output</span>
														</CardTitle>
														<CardDescription>
															Recent activity events for {selectedAgentName().toLowerCase()}.
														</CardDescription>
													</div>
													<Badge variant="outline">{activityEvents.length} lines</Badge>
												</div>
											</CardHeader>
											<CardContent>
												{#if activityEvents.length === 0}
													<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
														No activity lines yet. The console will append new events as the backend publishes into `activity.events`.
													</div>
												{:else}
													<div class="max-h-[32rem] space-y-3 overflow-y-auto pr-1">
														{#each activityEvents as item}
															<div class="rounded-3xl border border-border/70 bg-background px-4 py-4">
																<div class="flex flex-wrap items-center justify-between gap-3">
																	<p class="text-sm font-semibold">{item.event_type}</p>
																	<p class="text-xs text-muted-foreground">{formatTimestamp(item.created_at)}</p>
																</div>
																<p class="mt-3 whitespace-pre-wrap text-sm leading-6 text-foreground/90">
																	{item.message || 'No message payload'}
																</p>
																{#if Object.keys(item.metadata).length > 0}
																	<pre class="mt-3 overflow-x-auto rounded-2xl bg-muted/60 px-3 py-3 text-xs text-muted-foreground">{JSON.stringify(item.metadata, null, 2)}</pre>
																{/if}
															</div>
														{/each}
													</div>
												{/if}
											</CardContent>
										</Card>
									</div>
								</div>
							{/if}
						</CardContent>
					</Card>
				</div>
			</div>
		{/if}
	</section>
</div>
