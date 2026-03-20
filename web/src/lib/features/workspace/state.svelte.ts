import { defaultProjectForm, defaultWorkflowForm } from './mappers'
import type {
  BuiltinRole,
  HarnessValidationResponse,
  Organization,
  Project,
  Skill,
  Workflow,
} from './types'

export function createWorkspaceState() {
  const state = $state({
    booting: true,
    drawerOpen: false,
    orgBusy: false,
    projectBusy: false,
    workflowBusy: false,
    harnessBusy: false,
    validationBusy: false,
    skillBusy: false,
    pendingSkillName: '',
    organizations: [] as Organization[],
    projects: [] as Project[],
    workflows: [] as Workflow[],
    skills: [] as Skill[],
    builtinRoles: [] as BuiltinRole[],
    selectedOrgId: '',
    selectedProjectId: '',
    selectedWorkflowId: '',
    selectedBuiltinRoleSlug: '',
    selectedOrg: null as Organization | null,
    selectedProject: null as Project | null,
    selectedWorkflow: null as Workflow | null,
    notice: '',
    errorMessage: '',
    createOrgForm: { name: '', slug: '' },
    editOrgForm: { name: '', slug: '' },
    createProjectForm: defaultProjectForm(),
    editProjectForm: defaultProjectForm(),
    createWorkflowForm: defaultWorkflowForm(),
    editWorkflowForm: defaultWorkflowForm(),
    harnessDraft: '',
    harnessPath: '',
    harnessVersion: 0,
    harnessIssues: [] as HarnessValidationResponse['issues'],
  })

  return state
}
