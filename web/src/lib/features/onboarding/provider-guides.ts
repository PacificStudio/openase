import type { AgentProvider } from '$lib/api/contracts'

export type GuideKey = 'claude-code' | 'codex' | 'gemini'
export type GuideCommand = {
  label: string
  command: string
}

export type ProviderGuide = {
  key: GuideKey
  title: string
  adapterTypes: string[]
  docsUrl: string
  docsLabel: string
  recommendedModel: string
  installCommands: GuideCommand[]
  authCommands: GuideCommand[]
  verifyCommands: GuideCommand[]
  commonFixHints: string[]
}

export type DetectionStatus = 'yes' | 'no' | 'pending'

export const providerGuides: ProviderGuide[] = [
  {
    key: 'claude-code',
    title: 'Claude Code',
    adapterTypes: ['claude-code-cli', 'claude_code'],
    docsUrl: 'https://code.claude.com/docs/en/getting-started',
    docsLabel: 'Claude Code 官方文档',
    recommendedModel: 'claude-opus-4-6',
    installCommands: [
      {
        label: '官方推荐安装',
        command: 'curl -fsSL https://claude.ai/install.sh | bash',
      },
    ],
    authCommands: [
      {
        label: '首次登录',
        command: 'claude',
      },
    ],
    verifyCommands: [
      {
        label: '确认 CLI 可执行',
        command: 'claude --version',
      },
    ],
    commonFixHints: [
      'Windows 环境需要先安装 Git for Windows，Claude Code 会依赖 Git Bash。',
      '如果账号没有 Claude Code 权限，登录后仍会保持不可用状态。',
    ],
  },
  {
    key: 'codex',
    title: 'OpenAI Codex',
    adapterTypes: ['codex-app-server', 'codex'],
    docsUrl: 'https://developers.openai.com/codex/cli',
    docsLabel: 'Codex CLI 官方文档',
    recommendedModel: 'gpt-5.4',
    installCommands: [
      {
        label: '全局安装',
        command: 'npm i -g @openai/codex',
      },
    ],
    authCommands: [
      {
        label: '登录 ChatGPT / API',
        command: 'codex --login',
      },
    ],
    verifyCommands: [
      {
        label: '确认 CLI 可执行',
        command: 'codex --version',
      },
    ],
    commonFixHints: [
      '如果命令存在但无法执行，优先确认 npm 全局 bin 目录已经加入 PATH。',
      '如果登录状态丢失，重新运行 `codex --login`，或检查 API key / ChatGPT 计划权限。',
    ],
  },
  {
    key: 'gemini',
    title: 'Gemini CLI',
    adapterTypes: ['gemini-cli', 'gemini_cli'],
    docsUrl: 'https://github.com/google-gemini/gemini-cli',
    docsLabel: 'Gemini CLI 官方仓库',
    recommendedModel: 'gemini-2.5-pro',
    installCommands: [
      {
        label: '全局安装',
        command: 'npm install -g @google/gemini-cli',
      },
    ],
    authCommands: [
      {
        label: '首次登录 / 选择鉴权方式',
        command: 'gemini',
      },
    ],
    verifyCommands: [
      {
        label: '确认 CLI 可执行',
        command: 'gemini --version',
      },
    ],
    commonFixHints: [
      '首次启动时可以选择 Google 账号登录，也可以改用 `GEMINI_API_KEY` / Vertex AI。',
      '如果命令不存在，先确认 Node 18+ 与 npm 全局 bin 目录都在 PATH 中。',
    ],
  },
]

export const availabilityLabel: Record<string, { text: string; className: string }> = {
  available: { text: '可用', className: 'text-emerald-600 dark:text-emerald-400' },
  ready: { text: '可用', className: 'text-emerald-600 dark:text-emerald-400' },
  unavailable: { text: '不可用', className: 'text-destructive' },
  stale: { text: '状态过期', className: 'text-amber-600 dark:text-amber-400' },
  unknown: { text: '待检测', className: 'text-muted-foreground' },
}

export const cliDetectionLabel: Record<DetectionStatus, string> = {
  yes: '已检测到',
  no: '未检测到',
  pending: '待配置',
}

export const authDetectionLabel: Record<DetectionStatus, string> = {
  yes: '已登录',
  no: '未登录',
  pending: '待确认',
}

export function matchesGuide(provider: AgentProvider, guide: ProviderGuide): boolean {
  return guide.adapterTypes.includes(provider.adapter_type)
}

export function guideForProvider(provider: AgentProvider): ProviderGuide | null {
  return providerGuides.find((guide) => matchesGuide(provider, guide)) ?? null
}

export function guideProviders(providers: AgentProvider[], guide: ProviderGuide): AgentProvider[] {
  return providers.filter((provider) => matchesGuide(provider, guide))
}

export function isProviderAvailable(provider: AgentProvider): boolean {
  return provider.availability_state === 'available' || provider.availability_state === 'ready'
}

export function primaryGuideProvider(
  items: AgentProvider[],
  selectedId: string,
): AgentProvider | null {
  return (
    items.find((provider) => provider.id === selectedId) ??
    items.find((provider) => isProviderAvailable(provider)) ??
    items[0] ??
    null
  )
}

export function uniqueMachineIds(items: AgentProvider[]): string[] {
  return Array.from(
    new Set(
      items
        .map((provider) => provider.machine_id)
        .filter((value): value is string => Boolean(value && value.trim().length > 0)),
    ),
  )
}

export function cliDetectionState(items: AgentProvider[]): DetectionStatus {
  if (items.some((provider) => isProviderAvailable(provider))) {
    return 'yes'
  }
  if (items.some((provider) => provider.availability_reason === 'cli_missing')) {
    return 'no'
  }
  if (items.length === 0) {
    return 'pending'
  }
  return 'yes'
}

export function authDetectionState(items: AgentProvider[]): DetectionStatus {
  if (items.some((provider) => isProviderAvailable(provider))) {
    return 'yes'
  }
  if (items.some((provider) => provider.availability_reason === 'not_logged_in')) {
    return 'no'
  }
  return 'pending'
}

export function providerStatus(provider: AgentProvider | null) {
  if (!provider) {
    return availabilityLabel.unknown
  }
  return availabilityLabel[provider.availability_state] ?? availabilityLabel.unknown
}

export function reasonSpecificHints(provider: AgentProvider | null): string[] {
  switch (provider?.availability_reason) {
    case 'machine_offline':
      return ['绑定机器当前离线，先恢复机器在线状态，再回到这里执行重新检测。']
    case 'machine_degraded':
      return ['机器已连接但处于 degraded 状态，先修复主机健康问题，再重新检测。']
    case 'machine_maintenance':
      return ['机器处于维护模式，退出维护后再重新检测。']
    case 'cli_missing':
      return ['OpenASE 已找到这个 Provider，但在目标机器 PATH 中没有检测到对应 CLI。']
    case 'not_logged_in':
      return ['CLI 已安装，但当前认证缺失或已过期。重新执行登录命令后再检测。']
    case 'not_ready':
      return ['CLI 存在但 readiness probe 仍未通过，优先先跑一次验证命令确认具体报错。']
    case 'config_incomplete':
      return ['当前 Provider 注册信息不完整，检查 CLI command、模型和机器绑定是否都已填写。']
    case 'l4_snapshot_missing':
    case 'stale_l4_snapshot':
      return ['OpenASE 还没有拿到可用的最新机器快照，执行重新检测会重新拉取快照。']
    default:
      return []
  }
}
