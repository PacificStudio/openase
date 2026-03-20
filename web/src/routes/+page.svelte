<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
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
		Sparkles,
		TriangleAlert,
		Trash2,
		Waypoints
	} from '@lucide/svelte';
	import { connectEventStream, type SSEFrame, type StreamConnectionState } from '$lib/api/sse';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import HarnessEditor from '$lib/components/harness-editor.svelte';
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

	type TicketReference = {
		id: string;
		identifier: string;
		title: string;
		status_id: string;
		status_name: string;
	};

	type TicketDependency = {
		id: string;
		type: string;
		target: TicketReference;
	};

	type Ticket = {
		id: string;
		project_id: string;
		identifier: string;
		title: string;
		description: string;
		status_id: string;
		status_name: string;
		priority: 'urgent' | 'high' | 'medium' | 'low';
		type: string;
		workflow_id?: string | null;
		created_by: string;
		parent?: TicketReference | null;
		children: TicketReference[];
		dependencies: TicketDependency[];
		external_ref: string;
		budget_usd: number;
		cost_amount: number;
		attempt_count: number;
		consecutive_errors: number;
		next_retry_at?: string | null;
		retry_paused: boolean;
		pause_reason: string;
		created_at: string;
	};

	type TicketStatus = {
		id: string;
		project_id: string;
		name: string;
		color: string;
		icon?: string;
		position: number;
		is_default: boolean;
		description: string;
	};

	type WorkflowType = 'coding' | 'test' | 'doc' | 'security' | 'deploy' | 'refine-harness' | 'custom';

	type Workflow = {
		id: string;
		project_id: string;
		name: string;
		type: WorkflowType;
		harness_path: string;
		harness_content?: string | null;
		hooks: Record<string, unknown>;
		max_concurrent: number;
		max_retry_attempts: number;
		timeout_minutes: number;
		stall_timeout_minutes: number;
		version: number;
		is_active: boolean;
		pickup_status_id: string;
		finish_status_id?: string | null;
	};

	type HarnessDocument = {
		workflow_id: string;
		path: string;
		content: string;
		version: number;
	};

	type HarnessValidationIssue = {
		level: 'error' | 'warning' | string;
		message: string;
		line?: number;
		column?: number;
	};

	type HarnessValidationResponse = {
		valid: boolean;
		issues: HarnessValidationIssue[];
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
	type StatusPayload = { statuses: TicketStatus[] };
	type TicketPayload = { tickets: Ticket[] };
	type WorkflowListPayload = { workflows: Workflow[] };
	type WorkflowDetailPayload = { workflow: Workflow };
	type HarnessPayload = { harness: HarnessDocument };

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
	type WorkflowForm = {
		name: string;
		type: WorkflowType;
		pickupStatusId: string;
		finishStatusId: string;
		maxConcurrent: number;
		maxRetryAttempts: number;
		timeoutMinutes: number;
		stallTimeoutMinutes: number;
		isActive: boolean;
	};
	const projectStatuses: Project['status'][] = ['planning', 'active', 'paused', 'archived'];
	const workflowTypes: WorkflowType[] = [
		'coding',
		'test',
		'doc',
		'security',
		'deploy',
		'refine-harness',
		'custom'
	];
	const inputClass =
		'w-full rounded-2xl border border-border/70 bg-background/80 px-4 py-3 text-sm outline-none transition focus:border-foreground/40 focus:ring-2 focus:ring-foreground/10';
	const textAreaClass =
		'min-h-32 w-full rounded-2xl border border-border/70 bg-background/80 px-4 py-3 text-sm outline-none transition focus:border-foreground/40 focus:ring-2 focus:ring-foreground/10';
	const editorPlaceholder = `---
workflow:
  name: "coding"
  type: "coding"
status:
  pickup: "Todo"
  finish: "Done"
---

# Coding Workflow

You are handling {{ ticket.identifier }}.
`;

	let booting = $state(true);
	let orgBusy = $state(false);
	let projectBusy = $state(false);
	let workflowBusy = $state(false);
	let harnessBusy = $state(false);
	let validationBusy = $state(false);

	let organizations = $state<Organization[]>([]);
	let projects = $state<Project[]>([]);
	let ticketStatuses = $state<TicketStatus[]>([]);
	let tickets = $state<Ticket[]>([]);
	let workflows = $state<Workflow[]>([]);

	let selectedOrgId = $state('');
	let selectedProjectId = $state('');
	let selectedWorkflowId = $state('');

	let selectedOrg = $state<Organization | null>(null);
	let selectedProject = $state<Project | null>(null);
	let agents = $state<Agent[]>([]);
	let activityEvents = $state<ActivityEvent[]>([]);
	let selectedAgentId = $state('');
	let selectedWorkflow = $state<Workflow | null>(null);
	let notice = $state('');
	let errorMessage = $state('');
	let ticketBoardBusy = $state(false);
	let ticketBoardError = $state('');
	let ticketStreamState = $state<StreamConnectionState>('idle');
	let draggingTicketId = $state('');
	let dragTargetStatusId = $state('');
	let ticketMutationIds = $state<string[]>([]);
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
	let createWorkflowForm = $state<WorkflowForm>(defaultWorkflowForm());
	let editWorkflowForm = $state<WorkflowForm>(defaultWorkflowForm());

	let harnessDraft = $state('');
	let harnessPath = $state('');
	let harnessVersion = $state(0);
	let harnessIssues = $state<HarnessValidationIssue[]>([]);
	let lastValidatedContent = $state('');
	let validationRunID = 0;
	let validationTimer: ReturnType<typeof setTimeout> | null = null;
	let ticketLoadInFlight = false;
	let ticketReloadQueued = false;

	const harnessDirty = $derived(
		selectedWorkflow ? harnessDraft !== (selectedWorkflow.harness_content ?? '') : false
	);
	const harnessErrorCount = $derived(
		harnessIssues.filter((issue) => issue.level === 'error').length
	);
	const harnessWarningCount = $derived(
		harnessIssues.filter((issue) => issue.level !== 'error').length
	);

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
			resetTicketBoard();
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

	onDestroy(() => {
		clearPendingValidation();
	});

	$effect(() => {
		if (!selectedWorkflowId) {
			clearPendingValidation();
			validationBusy = false;
			lastValidatedContent = '';
			harnessIssues = [];
			return;
		}
		if (harnessDraft === lastValidatedContent) {
			return;
		}
		queueHarnessValidation(harnessDraft);
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
			clearWorkflowState();
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
			clearWorkflowState();
			return;
		}

		selectedProjectId = nextProject.id;
		selectedProject = nextProject;
		editProjectForm = toProjectForm(nextProject);
		await loadWorkflowContext(nextProject.id);
	}

	async function loadWorkflowContext(projectId: string, preferredWorkflowId?: string) {
		const [statusPayload, workflowPayload] = await Promise.all([
			api<StatusPayload>(`/api/v1/projects/${projectId}/statuses`),
			api<WorkflowListPayload>(`/api/v1/projects/${projectId}/workflows`)
		]);
		ticketStatuses = orderTicketStatuses(statusPayload.statuses);
		workflows = workflowPayload.workflows;
		createWorkflowForm = defaultWorkflowForm(ticketStatuses);
		await loadTickets(projectId);

		const nextWorkflow =
			workflows.find((item) => item.id === preferredWorkflowId) ??
			workflows.find((item) => item.id === selectedWorkflowId) ??
			workflows[0] ??
			null;

		if (!nextWorkflow) {
			resetSelectedWorkflow();
			return;
		}

		await loadWorkflowDetail(nextWorkflow.id);
	}

	async function loadWorkflowDetail(workflowId: string) {
		const payload = await api<WorkflowDetailPayload>(`/api/v1/workflows/${workflowId}`);
		selectedWorkflow = payload.workflow;
		selectedWorkflowId = payload.workflow.id;
		editWorkflowForm = toWorkflowForm(payload.workflow);
		harnessDraft = payload.workflow.harness_content ?? '';
		harnessPath = payload.workflow.harness_path;
		harnessVersion = payload.workflow.version;
		lastValidatedContent = '';
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
		clearWorkflowState();

		try {
			await loadProjects(org.id);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		}
	}

	async function loadTickets(projectId: string, options: { silent?: boolean } = {}) {
		const silent = options.silent ?? false;
		if (!silent) {
			ticketBoardBusy = true;
		}

		ticketLoadInFlight = true;
		try {
			const payload = await api<TicketPayload>(`/api/v1/projects/${projectId}/tickets`);
			if (projectId !== selectedProjectId) {
				return;
			}

			tickets = orderTickets(payload.tickets);
			ticketBoardError = '';
		} catch (error) {
			if (projectId !== selectedProjectId) {
				return;
			}
			ticketBoardError = toErrorMessage(error);
		} finally {
			const shouldReload = ticketReloadQueued;
			ticketReloadQueued = false;
			ticketLoadInFlight = false;

			if (projectId === selectedProjectId && !silent) {
				ticketBoardBusy = false;
			}
			if (shouldReload && projectId === selectedProjectId) {
				void loadTickets(projectId, { silent: true });
			}
		}
	}

	async function selectProject(project: Project) {
		if (project.id === selectedProjectId) {
			return;
		}

		selectedProjectId = project.id;
		selectedProject = project;
		editProjectForm = toProjectForm(project);
		errorMessage = '';
		clearWorkflowState();
		try {
			await loadWorkflowContext(project.id);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		}
	}

	async function selectWorkflow(workflow: Workflow) {
		if (workflow.id === selectedWorkflowId) {
			return;
		}

		errorMessage = '';
		try {
			await loadWorkflowDetail(workflow.id);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		}
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
		const closeTicketStream = connectEventStream(`/api/v1/projects/${projectId}/tickets/stream`, {
			onEvent: (frame) => handleTicketFrame(projectId, frame),
			onStateChange: (state) => {
				ticketStreamState = state;
			},
			onError: (error) => {
				ticketBoardError = toErrorMessage(error);
			}
		});

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
			closeTicketStream();
			closeAgentStream();
			closeActivityStream();
		};
	}

	function handleTicketFrame(projectId: string, frame: SSEFrame) {
		const envelope = parseStreamEnvelope(frame);
		if (!envelope || projectId !== selectedProjectId) {
			return;
		}

		queueTicketReload(projectId);
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

	function resetTicketBoard() {
		tickets = [];
		ticketBoardBusy = false;
		ticketBoardError = '';
		ticketStreamState = 'idle';
		draggingTicketId = '';
		dragTargetStatusId = '';
		ticketMutationIds = [];
		ticketLoadInFlight = false;
		ticketReloadQueued = false;
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

	async function createWorkflow() {
		if (!selectedProject) {
			return;
		}

		workflowBusy = true;
		errorMessage = '';
		notice = '';
		try {
			const payload = await api<{ workflow: Workflow }>(
				`/api/v1/projects/${selectedProject.id}/workflows`,
				{
					method: 'POST',
					body: JSON.stringify({
						name: createWorkflowForm.name,
						type: createWorkflowForm.type,
						pickup_status_id: createWorkflowForm.pickupStatusId,
						finish_status_id: createWorkflowForm.finishStatusId || null,
						max_concurrent: createWorkflowForm.maxConcurrent,
						max_retry_attempts: createWorkflowForm.maxRetryAttempts,
						timeout_minutes: createWorkflowForm.timeoutMinutes,
						stall_timeout_minutes: createWorkflowForm.stallTimeoutMinutes,
						is_active: createWorkflowForm.isActive
					})
				}
			);
			notice = `Workflow ${payload.workflow.name} created`;
			await loadWorkflowContext(selectedProject.id, payload.workflow.id);
			createWorkflowForm = defaultWorkflowForm(ticketStatuses);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		} finally {
			workflowBusy = false;
		}
	}

	async function updateWorkflow() {
		if (!selectedWorkflow || !selectedProject) {
			return;
		}

		workflowBusy = true;
		errorMessage = '';
		notice = '';
		try {
			const payload = await api<WorkflowDetailPayload>(`/api/v1/workflows/${selectedWorkflow.id}`, {
				method: 'PATCH',
				body: JSON.stringify({
					name: editWorkflowForm.name,
					type: editWorkflowForm.type,
					pickup_status_id: editWorkflowForm.pickupStatusId,
					finish_status_id: editWorkflowForm.finishStatusId || null,
					max_concurrent: editWorkflowForm.maxConcurrent,
					max_retry_attempts: editWorkflowForm.maxRetryAttempts,
					timeout_minutes: editWorkflowForm.timeoutMinutes,
					stall_timeout_minutes: editWorkflowForm.stallTimeoutMinutes,
					is_active: editWorkflowForm.isActive
				})
			});
			selectedWorkflow = {
				...payload.workflow,
				harness_content: selectedWorkflow.harness_content
			};
			workflows = workflows.map((item) => (item.id === payload.workflow.id ? payload.workflow : item));
			notice = `Workflow ${payload.workflow.name} updated`;
		} catch (error) {
			errorMessage = toErrorMessage(error);
		} finally {
			workflowBusy = false;
		}
	}

	async function deleteWorkflow() {
		if (!selectedWorkflow || !selectedProject) {
			return;
		}

		workflowBusy = true;
		errorMessage = '';
		notice = '';
		try {
			await api<{ workflow: Workflow }>(`/api/v1/workflows/${selectedWorkflow.id}`, {
				method: 'DELETE'
			});
			notice = `Workflow ${selectedWorkflow.name} deleted`;
			await loadWorkflowContext(selectedProject.id);
		} catch (error) {
			errorMessage = toErrorMessage(error);
		} finally {
			workflowBusy = false;
		}
	}

	async function saveHarness() {
		if (!selectedWorkflow) {
			return;
		}

		harnessBusy = true;
		errorMessage = '';
		notice = '';
		try {
			const workflowID = selectedWorkflow.id;
			const workflowName = selectedWorkflow.name;
			clearPendingValidation();
			const valid = await runHarnessValidation(harnessDraft);
			if (!valid) {
				errorMessage = 'Harness validation failed. Resolve YAML errors before saving.';
				return;
			}

			const payload = await api<HarnessPayload>(`/api/v1/workflows/${workflowID}/harness`, {
				method: 'PUT',
				body: JSON.stringify({
					content: harnessDraft
				})
			});

			harnessPath = payload.harness.path;
			harnessVersion = payload.harness.version;
			lastValidatedContent = harnessDraft;
			selectedWorkflow = {
				...selectedWorkflow,
				harness_content: harnessDraft,
				harness_path: payload.harness.path,
				version: payload.harness.version
			};
			workflows = workflows.map((item) =>
				item.id === workflowID
					? {
							...item,
							harness_path: payload.harness.path,
							version: payload.harness.version
						}
					: item
			);
			notice = `Harness saved for ${workflowName}`;
		} catch (error) {
			errorMessage = toErrorMessage(error);
		} finally {
			harnessBusy = false;
		}
	}

	async function validateHarnessNow() {
		clearPendingValidation();
		errorMessage = '';
		await runHarnessValidation(harnessDraft);
	}

	function queueHarnessValidation(content: string) {
		clearPendingValidation();
		validationBusy = true;
		const runID = ++validationRunID;
		validationTimer = setTimeout(() => {
			void runHarnessValidation(content, runID);
		}, 250);
	}

	async function runHarnessValidation(content: string, runID = ++validationRunID) {
		validationBusy = true;
		try {
			const response = await api<HarnessValidationResponse>('/api/v1/harness/validate', {
				method: 'POST',
				body: JSON.stringify({ content })
			});
			if (runID !== validationRunID) {
				return response.valid;
			}

			harnessIssues = response.issues;
			lastValidatedContent = content;
			return response.valid;
		} catch (error) {
			if (runID === validationRunID) {
				harnessIssues = [
					{
						level: 'error',
						message: toErrorMessage(error)
					}
				];
			}
			return false;
		} finally {
			if (runID === validationRunID) {
				validationBusy = false;
			}
		}
	}

	function clearPendingValidation() {
		if (validationTimer) {
			clearTimeout(validationTimer);
			validationTimer = null;
		}
	}

	function resetSelectedWorkflow() {
		selectedWorkflowId = '';
		selectedWorkflow = null;
		editWorkflowForm = defaultWorkflowForm(ticketStatuses);
		harnessDraft = '';
		harnessPath = '';
		harnessVersion = 0;
		harnessIssues = [];
		lastValidatedContent = '';
		clearPendingValidation();
	}

	function clearWorkflowState() {
		ticketStatuses = [];
		resetTicketBoard();
		workflows = [];
		createWorkflowForm = defaultWorkflowForm();
		resetSelectedWorkflow();
	}

	function queueTicketReload(projectId: string) {
		if (projectId !== selectedProjectId) {
			return;
		}
		if (ticketLoadInFlight) {
			ticketReloadQueued = true;
			return;
		}

		void loadTickets(projectId, { silent: true });
	}

	function orderTicketStatuses(statuses: TicketStatus[]) {
		return [...statuses].sort((left, right) => {
			const positionDelta = left.position - right.position;
			if (positionDelta !== 0) {
				return positionDelta;
			}

			return left.name.localeCompare(right.name);
		});
	}

	function orderTickets(items: Ticket[]) {
		return [...items].sort((left, right) => {
			const priorityDelta = ticketPriorityRank(left.priority) - ticketPriorityRank(right.priority);
			if (priorityDelta !== 0) {
				return priorityDelta;
			}

			const createdDelta = Date.parse(left.created_at) - Date.parse(right.created_at);
			if (!Number.isNaN(createdDelta) && createdDelta !== 0) {
				return createdDelta;
			}

			return left.identifier.localeCompare(right.identifier);
		});
	}

	function ticketPriorityRank(priority: Ticket['priority']) {
		switch (priority) {
			case 'urgent':
				return 0;
			case 'high':
				return 1;
			case 'medium':
				return 2;
			default:
				return 3;
		}
	}

	function ticketsForStatus(statusID: string) {
		return tickets.filter((ticket) => ticket.status_id === statusID);
	}

	function workflowName(workflowID?: string | null) {
		if (!workflowID) {
			return 'No workflow';
		}

		return workflows.find((workflow) => workflow.id === workflowID)?.name ?? 'Detached workflow';
	}

	function ticketDetailHref(ticketID: string) {
		if (!selectedProjectId) {
			return '/ticket';
		}

		return `/ticket?project=${encodeURIComponent(selectedProjectId)}&id=${encodeURIComponent(ticketID)}`;
	}

	function isTicketMutationPending(ticketID: string) {
		return ticketMutationIds.includes(ticketID);
	}

	function ticketPriorityBadgeClass(priority: Ticket['priority']) {
		switch (priority) {
			case 'urgent':
				return 'border-rose-500/25 bg-rose-500/10 text-rose-700';
			case 'high':
				return 'border-amber-500/25 bg-amber-500/10 text-amber-700';
			case 'medium':
				return 'border-sky-500/25 bg-sky-500/10 text-sky-700';
			default:
				return 'border-border/80 bg-background text-muted-foreground';
		}
	}

	function handleTicketDragStart(event: DragEvent, ticket: Ticket) {
		draggingTicketId = ticket.id;
		dragTargetStatusId = ticket.status_id;
		event.dataTransfer?.setData('text/plain', ticket.id);
		if (event.dataTransfer) {
			event.dataTransfer.effectAllowed = 'move';
		}
	}

	function handleStatusDragOver(event: DragEvent, statusID: string) {
		event.preventDefault();
		dragTargetStatusId = statusID;
		if (event.dataTransfer) {
			event.dataTransfer.dropEffect = 'move';
		}
	}

	async function handleStatusDrop(event: DragEvent, statusID: string) {
		event.preventDefault();
		const ticketID = draggingTicketId || event.dataTransfer?.getData('text/plain') || '';
		dragTargetStatusId = '';
		const ticket = tickets.find((item) => item.id === ticketID);
		if (!ticket) {
			return;
		}

		await moveTicketToStatus(ticket, statusID);
	}

	async function moveTicketToStatus(ticket: Ticket, statusID: string) {
		if (ticket.status_id === statusID || isTicketMutationPending(ticket.id)) {
			return;
		}

		const previousStatusID = ticket.status_id;
		const previousStatusName = ticket.status_name;
		ticketBoardError = '';
		ticketMutationIds = [...ticketMutationIds, ticket.id];
		tickets = orderTickets(
			tickets.map((item) =>
				item.id === ticket.id
					? { ...item, status_id: statusID, status_name: statusName(statusID) }
					: item
			)
		);

		try {
			const payload = await api<{ ticket: Ticket }>(`/api/v1/tickets/${ticket.id}`, {
				method: 'PATCH',
				body: JSON.stringify({ status_id: statusID })
			});
			tickets = orderTickets(
				tickets.map((item) => (item.id === payload.ticket.id ? payload.ticket : item))
			);
		} catch (error) {
			tickets = orderTickets(
				tickets.map((item) =>
					item.id === ticket.id
						? { ...item, status_id: previousStatusID, status_name: previousStatusName }
						: item
				)
			);
			ticketBoardError = toErrorMessage(error);
		} finally {
			ticketMutationIds = ticketMutationIds.filter((itemID) => itemID !== ticket.id);
			draggingTicketId = '';
			dragTargetStatusId = '';
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

	function toWorkflowForm(item: Workflow): WorkflowForm {
		return {
			name: item.name,
			type: item.type,
			pickupStatusId: item.pickup_status_id,
			finishStatusId: item.finish_status_id ?? '',
			maxConcurrent: item.max_concurrent,
			maxRetryAttempts: item.max_retry_attempts,
			timeoutMinutes: item.timeout_minutes,
			stallTimeoutMinutes: item.stall_timeout_minutes,
			isActive: item.is_active
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

	function defaultWorkflowForm(statuses: TicketStatus[] = []): WorkflowForm {
		const pickup = statuses.find((status) => status.is_default) ?? statuses[0];
		const finish =
			statuses.find((status) => status.name.toLowerCase() === 'done') ??
			statuses.find((status) => status.name.toLowerCase() === 'completed') ??
			statuses[statuses.length - 1];

		return {
			name: '',
			type: 'coding',
			pickupStatusId: pickup?.id ?? '',
			finishStatusId: finish?.id ?? '',
			maxConcurrent: 3,
			maxRetryAttempts: 3,
			timeoutMinutes: 60,
			stallTimeoutMinutes: 5,
			isActive: true
		};
	}

	function statusName(statusID?: string | null) {
		if (!statusID) {
			return 'No finish state';
		}

		return ticketStatuses.find((status) => status.id === statusID)?.name ?? 'Unknown';
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

		const payload = (await response.json().catch(() => ({}))) as { error?: string; message?: string };
		if (!response.ok) {
			throw new Error(payload.message ?? payload.error ?? `request failed with status ${response.status}`);
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
	<title>OpenASE Workflow Management</title>
	<meta
		name="description"
		content="Workflow management and harness editing for OpenASE projects, backed by the embedded Go API."
	/>
</svelte:head>

<div class="relative overflow-hidden">
	<div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(194,120,3,0.2),transparent_28rem),radial-gradient(circle_at_bottom_right,rgba(13,148,136,0.18),transparent_30rem)]"></div>

	<section class="relative mx-auto flex min-h-screen w-full max-w-7xl flex-col gap-8 px-6 py-8 sm:px-8 lg:px-10">
		<div class="grid gap-4 lg:grid-cols-[minmax(0,1.2fr)_minmax(18rem,0.8fr)]">
			<div class="space-y-5">
				<div class="flex flex-wrap items-center gap-3">
					<Badge variant="outline">F24 board</Badge>
					<Badge variant="outline">Agent console</Badge>
					<Badge variant="outline">Realtime SSE</Badge>
					<Badge variant="outline">Workflow management</Badge>
					<Badge variant="outline">Harness editor</Badge>
				</div>

				<div class="space-y-4">
					<p class="text-sm font-medium uppercase tracking-[0.32em] text-muted-foreground">
						OpenASE control plane
					</p>
					<h1 class="max-w-4xl text-5xl leading-none font-semibold tracking-[-0.06em] text-balance sm:text-6xl">
						Kanban routing, live agent telemetry, and workflow harness editing now share one control plane.
					</h1>
					<p class="max-w-3xl text-lg leading-8 text-muted-foreground">
						Pick a project, drag tickets across custom status columns, watch SSE-driven updates land
						in real time, then manage workflows and edit Git-backed harness instructions without
						leaving the same embedded surface.
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
						<CardDescription>Workflows in focus</CardDescription>
						<CardTitle class="text-4xl tracking-[-0.05em]">
							{selectedProject ? workflows.length : 0}
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
					<span>Loading workflow management surface…</span>
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
									No organizations yet. Create the first one to unlock project and workflow CRUD.
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
										onclick={() => void selectOrganization(org)}
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
											Select a project to edit, then manage its workflows and harnesses.
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
											onclick={() => void selectProject(project)}
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
										Keep project metadata sharp, or archive it without deleting history.
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
						<CardHeader class="border-b border-border/70 bg-muted/20">
							<div class="flex flex-wrap items-center justify-between gap-4">
								<div>
									<CardTitle class="flex items-center gap-2">
										<FolderKanban class="size-4" />
										<span>Kanban board</span>
									</CardTitle>
									<CardDescription>
										Custom statuses become columns. Drag a ticket between columns to persist its
										`status_id`, while SSE keeps the board fresh.
									</CardDescription>
								</div>
								<div class="flex flex-wrap items-center gap-2">
									{#if selectedProject}
										<Badge variant="outline">{tickets.length} tickets</Badge>
									{/if}
									<span class={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${streamBadgeClass(ticketStreamState)}`}>
										<Cable class="mr-1.5 size-3.5" />
										Board {ticketStreamState}
									</span>
								</div>
							</div>
						</CardHeader>
						<CardContent class="space-y-5 p-6">
							{#if ticketBoardError}
								<div class="rounded-3xl border border-destructive/25 bg-destructive/10 px-4 py-3 text-sm text-destructive">
									{ticketBoardError}
								</div>
							{/if}

							{#if !selectedProject}
								<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-8 text-sm text-muted-foreground">
									Select a project to load its custom status columns and ticket board.
								</div>
							{:else if ticketBoardBusy && tickets.length === 0}
								<div class="flex min-h-72 items-center justify-center rounded-[2rem] border border-border/70 bg-background/60">
									<div class="flex items-center gap-3 text-sm text-muted-foreground">
										<LoaderCircle class="size-4 animate-spin" />
										<span>Loading board columns and tickets…</span>
									</div>
								</div>
							{:else if ticketStatuses.length === 0}
								<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-8 text-sm text-muted-foreground">
									No ticket statuses are configured for this project yet.
								</div>
							{:else}
								<div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-muted/25 px-4 py-3 text-sm text-muted-foreground">
									<p>
										<span class="font-medium text-foreground">{selectedProject.name}</span>
										uses {ticketStatuses.length} custom columns ordered by status position.
									</p>
									<p>Drop a card into any column to trigger a ticket status change.</p>
								</div>

								<div class="overflow-x-auto pb-2">
									<div class="flex min-w-max gap-4">
										{#each ticketStatuses as status}
											<div
												class={`min-h-[30rem] w-[19rem] shrink-0 rounded-[1.75rem] border px-4 py-4 transition ${
													dragTargetStatusId === status.id
														? 'border-foreground/25 bg-background shadow-lg shadow-black/5'
														: 'border-border/70 bg-background/60'
												}`}
												style={`box-shadow: inset 0 3px 0 ${status.color};`}
												role="list"
												aria-label={`${status.name} tickets`}
												ondragover={(event) => handleStatusDragOver(event, status.id)}
												ondragenter={() => {
													dragTargetStatusId = status.id;
												}}
												ondrop={(event) => void handleStatusDrop(event, status.id)}
											>
												<div class="flex items-start justify-between gap-3">
													<div class="min-w-0">
														<div class="flex items-center gap-2">
															<span
																class="size-3 rounded-full border border-white/60"
																style={`background-color: ${status.color};`}
															></span>
															<p class="truncate text-sm font-semibold">{status.name}</p>
														</div>
														{#if status.description}
															<p class="mt-2 text-xs leading-5 text-muted-foreground">
																{status.description}
															</p>
														{/if}
													</div>
													<div class="flex flex-col items-end gap-2">
														<Badge variant={status.is_default ? 'secondary' : 'outline'}>
															{ticketsForStatus(status.id).length}
														</Badge>
														{#if status.is_default}
															<span class="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
																default
															</span>
														{/if}
													</div>
												</div>

												<div class="mt-4 space-y-3">
													{#if ticketsForStatus(status.id).length === 0}
														<div class="rounded-3xl border border-dashed border-border/70 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
															Drop a ticket here to move it into {status.name}.
														</div>
													{:else}
														{#each ticketsForStatus(status.id) as ticket}
															<button
																type="button"
																class={`w-full rounded-3xl border px-4 py-4 text-left transition ${
																	draggingTicketId === ticket.id
																		? 'border-foreground/20 bg-muted/40 opacity-60'
																		: 'border-border/70 bg-background hover:border-foreground/15 hover:bg-background'
																}`}
																draggable={!isTicketMutationPending(ticket.id)}
																ondragstart={(event) => handleTicketDragStart(event, ticket)}
																ondragend={() => {
																	draggingTicketId = '';
																	dragTargetStatusId = '';
																}}
															>
																<div class="flex items-start justify-between gap-3">
																	<div class="min-w-0">
																		<p class="font-mono text-[11px] uppercase tracking-[0.2em] text-muted-foreground">
																			{ticket.identifier}
																		</p>
																		<p class="mt-2 text-sm font-semibold text-foreground">
																			{ticket.title}
																		</p>
																	</div>
																	<span class={`inline-flex rounded-full border px-2.5 py-1 text-[11px] font-medium ${ticketPriorityBadgeClass(ticket.priority)}`}>
																		{ticket.priority}
																	</span>
																</div>

																<p class="mt-3 text-sm leading-6 text-muted-foreground">
																	{ticket.description || 'No description yet.'}
																</p>

																<div class="mt-4 flex flex-wrap gap-2 text-[11px]">
																	<span class="inline-flex rounded-full border border-border/80 bg-background px-2.5 py-1 text-muted-foreground">
																		{ticket.type}
																	</span>
																	{#if ticket.workflow_id}
																		<span class="inline-flex rounded-full border border-border/80 bg-background px-2.5 py-1 text-muted-foreground">
																			{workflowName(ticket.workflow_id)}
																		</span>
																	{/if}
																	{#if ticket.consecutive_errors > 0}
																		<span class="inline-flex rounded-full border border-amber-500/25 bg-amber-500/10 px-2.5 py-1 text-amber-700">
																			retry {ticket.consecutive_errors}
																		</span>
																	{/if}
																	{#if ticket.retry_paused}
																		<span class="inline-flex rounded-full border border-rose-500/25 bg-rose-500/10 px-2.5 py-1 text-rose-700">
																			{ticket.pause_reason || 'paused'}
																		</span>
																	{/if}
																</div>

																<div class="mt-4 flex items-center justify-between gap-3 text-xs text-muted-foreground">
																	<span>{formatTimestamp(ticket.created_at)}</span>
																	<div class="flex items-center gap-3">
																		<a
																			href={ticketDetailHref(ticket.id)}
																			class="inline-flex items-center gap-1 rounded-full border border-border/70 bg-background px-2.5 py-1 font-medium text-foreground transition hover:border-foreground/20"
																			onclick={(event) => event.stopPropagation()}
																		>
																			<Waypoints class="size-3" />
																			Detail
																		</a>
																		{#if isTicketMutationPending(ticket.id)}
																			<span class="inline-flex items-center gap-1">
																				<LoaderCircle class="size-3 animate-spin" />
																				Updating
																			</span>
																		{/if}
																	</div>
																</div>
															</button>
														{/each}
													{/if}
												</div>
											</div>
										{/each}
									</div>
								</div>
							{/if}
						</CardContent>
					</Card>

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

					<Card class="overflow-hidden border-border/80 bg-background/80 backdrop-blur">
						<CardHeader class="border-b border-border/70 bg-muted/20">
							<div class="flex flex-wrap items-center justify-between gap-4">
								<div>
									<CardTitle class="flex items-center gap-2">
										<Waypoints class="size-4" />
										<span>Workflow management</span>
									</CardTitle>
									<CardDescription>
										Manage workflow metadata and the Git-backed harness document for the selected project.
									</CardDescription>
								</div>
								{#if selectedProject}
									<Badge variant="outline">{selectedProject.name}</Badge>
								{/if}
							</div>
						</CardHeader>
						<CardContent class="p-6">
							{#if !selectedProject}
								<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-10 text-sm text-muted-foreground">
									Select a project first. Workflow and harness management is scoped per project.
								</div>
							{:else}
								<div class="grid gap-6 xl:grid-cols-[19rem_minmax(0,1fr)]">
									<div class="space-y-6">
										<Card class="border-border/80 bg-background/70">
											<CardHeader>
												<div class="flex items-center justify-between gap-4">
													<div>
														<CardTitle class="flex items-center gap-2">
															<FolderKanban class="size-4" />
															<span>Workflow roster</span>
														</CardTitle>
														<CardDescription>
															Every workflow stays attached to the current project.
														</CardDescription>
													</div>
													<Badge variant="outline">{workflows.length}</Badge>
												</div>
											</CardHeader>
											<CardContent class="space-y-3">
												{#if workflows.length === 0}
													<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-6 text-sm text-muted-foreground">
														No workflows yet. Create the first one to open the harness editor.
													</div>
												{:else}
													{#each workflows as workflow}
														<button
															type="button"
															class={`w-full rounded-3xl border px-4 py-4 text-left transition ${
																workflow.id === selectedWorkflowId
																	? 'border-sky-500/40 bg-sky-950/90 text-white shadow-lg shadow-sky-950/25'
																	: 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
															}`}
															onclick={() => void selectWorkflow(workflow)}
														>
															<div class="flex items-start justify-between gap-3">
																<div>
																	<p class="text-sm font-semibold">{workflow.name}</p>
																	<p
																		class={`mt-1 text-xs uppercase tracking-[0.2em] ${
																			workflow.id === selectedWorkflowId
																				? 'text-white/70'
																				: 'text-muted-foreground'
																		}`}
																	>
																		{workflow.type}
																	</p>
																</div>
																<Badge variant={workflow.is_active ? 'secondary' : 'outline'}>
																	v{workflow.version}
																</Badge>
															</div>
															<p
																class={`mt-3 text-xs ${
																	workflow.id === selectedWorkflowId
																		? 'text-white/75'
																		: 'text-muted-foreground'
																}`}
															>
																{statusName(workflow.pickup_status_id)} -> {statusName(workflow.finish_status_id)}
															</p>
														</button>
													{/each}
												{/if}
											</CardContent>
										</Card>

										<Card class="border-border/80 bg-background/70">
											<CardHeader>
												<CardTitle class="flex items-center gap-2">
													<Plus class="size-4" />
													<span>Create workflow</span>
												</CardTitle>
												<CardDescription>
													Start narrow: create the role, wire its statuses, then refine the harness.
												</CardDescription>
											</CardHeader>
											<CardContent>
												<form
													class="space-y-4"
													onsubmit={(event) => {
														event.preventDefault();
														void createWorkflow();
													}}
												>
													<div class="space-y-2">
														<label
															class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
															for="create-workflow-name"
														>
															Name
														</label>
														<input
															id="create-workflow-name"
															class={inputClass}
															bind:value={createWorkflowForm.name}
															placeholder="Coding Workflow"
														/>
													</div>
													<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-1">
														<div class="space-y-2">
															<label
																class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
																for="create-workflow-type"
															>
																Type
															</label>
															<select
																id="create-workflow-type"
																class={inputClass}
																bind:value={createWorkflowForm.type}
															>
																{#each workflowTypes as workflowType}
																	<option value={workflowType}>{workflowType}</option>
																{/each}
															</select>
														</div>
														<div class="space-y-2">
															<label
																class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
																for="create-workflow-pickup-status"
															>
																Pickup status
															</label>
															<select
																id="create-workflow-pickup-status"
																class={inputClass}
																bind:value={createWorkflowForm.pickupStatusId}
															>
																<option value="" disabled>Choose a status</option>
																{#each ticketStatuses as status}
																	<option value={status.id}>{status.name}</option>
																{/each}
															</select>
														</div>
													</div>
													<div class="space-y-2">
														<label
															class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
															for="create-workflow-finish-status"
														>
															Finish status
														</label>
														<select
															id="create-workflow-finish-status"
															class={inputClass}
															bind:value={createWorkflowForm.finishStatusId}
														>
															<option value="">No auto-finish status</option>
															{#each ticketStatuses as status}
																<option value={status.id}>{status.name}</option>
															{/each}
														</select>
													</div>
													<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-1">
														<div class="space-y-2">
															<label
																class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
																for="create-workflow-max-concurrent"
															>
																Max concurrent
															</label>
															<input
																id="create-workflow-max-concurrent"
																class={inputClass}
																bind:value={createWorkflowForm.maxConcurrent}
																type="number"
																min="1"
															/>
														</div>
														<div class="space-y-2">
															<label
																class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground"
																for="create-workflow-max-retries"
															>
																Max retries
															</label>
															<input
																id="create-workflow-max-retries"
																class={inputClass}
																bind:value={createWorkflowForm.maxRetryAttempts}
																type="number"
																min="0"
															/>
														</div>
													</div>
													<Button class="w-full" type="submit" disabled={workflowBusy || ticketStatuses.length === 0}>
														{#if workflowBusy}
															<LoaderCircle class="mr-2 size-4 animate-spin" />
														{:else}
															<Plus class="mr-2 size-4" />
														{/if}
														Create workflow
													</Button>
												</form>
											</CardContent>
										</Card>
									</div>

									<div class="space-y-6">
										<Card class="border-border/80 bg-background/70">
											<CardHeader>
												<div class="flex flex-wrap items-center justify-between gap-4">
													<div>
														<CardTitle class="flex items-center gap-2">
															<Sparkles class="size-4" />
															<span>Selected workflow</span>
														</CardTitle>
														<CardDescription>
															Update runtime limits and route the workflow through the right states.
														</CardDescription>
													</div>
													{#if selectedWorkflow}
														<Badge variant={selectedWorkflow.is_active ? 'secondary' : 'outline'}>
															{selectedWorkflow.is_active ? 'active' : 'inactive'}
														</Badge>
													{/if}
												</div>
											</CardHeader>
											<CardContent>
												{#if selectedWorkflow}
													<form
														class="space-y-4"
														onsubmit={(event) => {
															event.preventDefault();
															void updateWorkflow();
														}}
													>
														<div class="grid gap-4 lg:grid-cols-2">
															<div class="space-y-2">
																<label class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground" for="edit-workflow-name">
																	Name
																</label>
																<input id="edit-workflow-name" class={inputClass} bind:value={editWorkflowForm.name} />
															</div>
															<div class="space-y-2">
																<label class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground" for="edit-workflow-type">
																	Type
																</label>
																<select id="edit-workflow-type" class={inputClass} bind:value={editWorkflowForm.type}>
																	{#each workflowTypes as workflowType}
																		<option value={workflowType}>{workflowType}</option>
																	{/each}
																</select>
															</div>
														</div>
														<div class="grid gap-4 lg:grid-cols-2">
															<div class="space-y-2">
																<label class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground" for="edit-workflow-pickup-status">
																	Pickup status
																</label>
																<select id="edit-workflow-pickup-status" class={inputClass} bind:value={editWorkflowForm.pickupStatusId}>
																	{#each ticketStatuses as status}
																		<option value={status.id}>{status.name}</option>
																	{/each}
																</select>
															</div>
															<div class="space-y-2">
																<label class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground" for="edit-workflow-finish-status">
																	Finish status
																</label>
																<select id="edit-workflow-finish-status" class={inputClass} bind:value={editWorkflowForm.finishStatusId}>
																	<option value="">No auto-finish status</option>
																	{#each ticketStatuses as status}
																		<option value={status.id}>{status.name}</option>
																	{/each}
																</select>
															</div>
														</div>
														<div class="grid gap-4 lg:grid-cols-4">
															<div class="space-y-2">
																<label class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground" for="edit-workflow-max-concurrent">
																	Max concurrent
																</label>
																<input id="edit-workflow-max-concurrent" class={inputClass} bind:value={editWorkflowForm.maxConcurrent} type="number" min="1" />
															</div>
															<div class="space-y-2">
																<label class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground" for="edit-workflow-max-retries">
																	Max retries
																</label>
																<input id="edit-workflow-max-retries" class={inputClass} bind:value={editWorkflowForm.maxRetryAttempts} type="number" min="0" />
															</div>
															<div class="space-y-2">
																<label class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground" for="edit-workflow-timeout">
																	Timeout min
																</label>
																<input id="edit-workflow-timeout" class={inputClass} bind:value={editWorkflowForm.timeoutMinutes} type="number" min="1" />
															</div>
															<div class="space-y-2">
																<label class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground" for="edit-workflow-stall-timeout">
																	Stall min
																</label>
																<input id="edit-workflow-stall-timeout" class={inputClass} bind:value={editWorkflowForm.stallTimeoutMinutes} type="number" min="1" />
															</div>
														</div>
														<label class="flex items-center gap-3 rounded-2xl border border-border/70 bg-background/60 px-4 py-3 text-sm">
															<input bind:checked={editWorkflowForm.isActive} class="size-4 rounded border-border" type="checkbox" />
															<span>Workflow is active and dispatchable</span>
														</label>
														<div class="rounded-2xl border border-border/70 bg-muted/30 px-4 py-3">
															<p class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground">Harness path</p>
															<p class="mt-2 font-mono text-xs text-foreground/80">{harnessPath}</p>
														</div>
														<div class="flex flex-wrap gap-3">
															<Button type="submit" disabled={workflowBusy}>
																{#if workflowBusy}
																	<LoaderCircle class="mr-2 size-4 animate-spin" />
																{:else}
																	<Save class="mr-2 size-4" />
																{/if}
																Save workflow
															</Button>
															<Button type="button" variant="outline" disabled={workflowBusy} onclick={() => void deleteWorkflow()}>
																<Trash2 class="mr-2 size-4" />
																Delete workflow
															</Button>
														</div>
													</form>
												{:else}
													<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-8 text-sm text-muted-foreground">
														Create or select a workflow to edit its metadata and harness.
													</div>
												{/if}
											</CardContent>
										</Card>

										<Card class="border-border/80 bg-background/70">
											<CardHeader>
												<div class="flex flex-wrap items-center justify-between gap-4">
													<div>
														<CardTitle class="flex items-center gap-2">
															<Save class="size-4" />
															<span>Harness editor</span>
														</CardTitle>
														<CardDescription>
															YAML frontmatter is validated live. Save writes directly through the workflow API.
														</CardDescription>
													</div>
													<div class="flex flex-wrap items-center gap-2">
														{#if selectedWorkflow}
															<Badge variant="outline">v{harnessVersion}</Badge>
														{/if}
														<Badge variant={harnessErrorCount > 0 ? 'destructive' : 'secondary'}>
															{harnessErrorCount > 0 ? `${harnessErrorCount} error` : 'YAML ok'}
														</Badge>
														{#if harnessWarningCount > 0}
															<Badge variant="outline">{harnessWarningCount} warning</Badge>
														{/if}
													</div>
												</div>
											</CardHeader>
											<CardContent class="space-y-4">
												{#if selectedWorkflow}
													<div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-muted/25 px-4 py-3 text-sm">
														<div class="flex flex-wrap items-center gap-3 text-muted-foreground">
															<span class="font-medium text-foreground">{selectedWorkflow.name}</span>
															<span class="font-mono text-xs">{harnessPath}</span>
															{#if harnessDirty}
																<Badge variant="outline">unsaved</Badge>
															{/if}
														</div>
														<div class="flex flex-wrap gap-2">
															<Button type="button" variant="outline" disabled={validationBusy} onclick={() => void validateHarnessNow()}>
																{#if validationBusy}
																	<LoaderCircle class="mr-2 size-4 animate-spin" />
																{:else}
																	<TriangleAlert class="mr-2 size-4" />
																{/if}
																Validate
															</Button>
															<Button type="button" disabled={harnessBusy} onclick={() => void saveHarness()}>
																{#if harnessBusy}
																	<LoaderCircle class="mr-2 size-4 animate-spin" />
																{:else}
																	<Save class="mr-2 size-4" />
																{/if}
																Save harness
															</Button>
														</div>
													</div>

													<HarnessEditor bind:value={harnessDraft} issues={harnessIssues} placeholder={editorPlaceholder} />

													<div class="grid gap-3">
														{#if validationBusy}
															<div class="rounded-2xl border border-border/70 bg-background/60 px-4 py-3 text-sm text-muted-foreground">
																<div class="flex items-center gap-2">
																	<LoaderCircle class="size-4 animate-spin" />
																	<span>Checking YAML frontmatter…</span>
																</div>
															</div>
														{:else if harnessIssues.length === 0}
															<div class="rounded-2xl border border-emerald-500/25 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-900">
																Harness YAML frontmatter is valid.
															</div>
														{:else}
															{#each harnessIssues as issue}
																<div class={`rounded-2xl border px-4 py-3 text-sm ${
																	issue.level === 'error'
																		? 'border-rose-500/25 bg-rose-500/10 text-rose-900'
																		: 'border-amber-500/25 bg-amber-500/10 text-amber-900'
																}`}>
																	<p class="font-medium uppercase tracking-[0.18em]">
																		{issue.level}
																		{#if issue.line}
																			{' '}line {issue.line}
																			{#if issue.column}
																				, column {issue.column}
																			{/if}
																		{/if}
																	</p>
																	<p class="mt-1">{issue.message}</p>
																</div>
															{/each}
														{/if}
													</div>
												{:else}
													<div class="rounded-3xl border border-dashed border-border/80 bg-muted/35 px-4 py-10 text-sm text-muted-foreground">
														Select a workflow to open the harness editor.
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
