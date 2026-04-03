import type {
  OnboardingData,
  OnboardingStep,
  OnboardingStepId,
  ProjectBootstrapPreset,
} from './types'

const stepDefinitions: { id: OnboardingStepId; label: string; description: string }[] = [
  {
    id: 'github_token',
    label: '连接 GitHub',
    description: '配置 GitHub Token 以便访问代码仓库',
  },
  {
    id: 'repo',
    label: '创建或关联代码仓库',
    description: '为项目添加至少一个 Git 仓库',
  },
  {
    id: 'provider',
    label: '选择并配置 AI Provider',
    description: '至少配置一个可用的 AI 执行引擎',
  },
  {
    id: 'agent_workflow',
    label: '创建 Agent 与 Workflow',
    description: '自动创建首个可工作的 Agent 与 Workflow',
  },
  {
    id: 'first_ticket',
    label: '创建首个 Ticket',
    description: '提交第一个任务让 Agent 开始工作',
  },
  {
    id: 'ai_discovery',
    label: '体验 Project AI 与 Harness AI',
    description: '使用 AI 助手进一步优化项目配置',
  },
]

export function isStepCompleted(step: OnboardingStepId, data: OnboardingData): boolean {
  switch (step) {
    case 'github_token':
      return data.github.hasToken && data.github.probeStatus === 'valid' && data.github.confirmed
    case 'repo':
      return data.repo.repos.length > 0
    case 'provider': {
      const selectedProvider = data.provider.providers.find(
        (provider) => provider.id === data.provider.selectedProviderId,
      )
      return Boolean(
        selectedProvider &&
        (selectedProvider.availability_state === 'available' ||
          selectedProvider.availability_state === 'ready'),
      )
    }
    case 'agent_workflow':
      return data.agentWorkflow.agents.length > 0 && data.agentWorkflow.workflows.length > 0
    case 'first_ticket':
      return data.firstTicket.ticketCount > 0
    case 'ai_discovery':
      return data.aiDiscovery.completed
  }
}

export function buildOnboardingSteps(data: OnboardingData): OnboardingStep[] {
  const steps: OnboardingStep[] = []
  let foundActive = false

  for (const def of stepDefinitions) {
    const completed = isStepCompleted(def.id, data)
    let status: OnboardingStep['status']

    if (completed) {
      status = 'completed'
    } else if (!foundActive) {
      status = 'active'
      foundActive = true
    } else {
      status = 'locked'
    }

    steps.push({ ...def, status })
  }

  return steps
}

export function currentActiveStep(data: OnboardingData): OnboardingStepId | null {
  for (const def of stepDefinitions) {
    if (!isStepCompleted(def.id, data)) {
      return def.id
    }
  }
  return null
}

export function isOnboardingComplete(data: OnboardingData): boolean {
  return currentActiveStep(data) === null
}

export function getBootstrapPreset(projectStatus: string): ProjectBootstrapPreset {
  switch (projectStatus) {
    case 'In Progress':
      return {
        roleName: 'Fullstack Developer',
        roleSlug: 'fullstack-developer',
        workflowType: 'Fullstack Developer',
        pickupStatusName: 'Backlog',
        finishStatusName: 'Done',
        agentNameSuggestion: 'fullstack-developer-01',
        exampleTicketTitle: '实现项目的第一个核心功能',
      }
    case 'Planned':
    case 'Backlog':
    default:
      return {
        roleName: '产品经理',
        roleSlug: 'product-manager',
        workflowType: 'Product Manager',
        pickupStatusName: 'Backlog',
        finishStatusName: 'Done',
        agentNameSuggestion: 'product-manager-01',
        exampleTicketTitle: '梳理项目需求并输出第一版 PRD',
      }
  }
}

export function isTerminalProjectStatus(status: string): boolean {
  return ['Completed', 'Canceled', 'Archived'].includes(status)
}
