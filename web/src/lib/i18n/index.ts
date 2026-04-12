export const SUPPORTED_LOCALES = ['en', 'zh'] as const

export type AppLocale = (typeof SUPPORTED_LOCALES)[number]

const translations = {
  en: {
    'common.appName': 'OpenASE',
    'common.actions': 'Actions',
    'common.agent': 'Agent',
    'common.all': 'All',
    'common.blocked': 'Blocked',
    'common.cancel': 'Cancel',
    'common.status': 'Status',
    'common.priority': 'Priority',
    'common.retry': 'Retry',
    'common.language': 'Language',
    'common.links': 'Links',
    'common.loading': 'Loading...',
    'common.ticket': 'Ticket',
    'common.unassigned': 'Unassigned',
    'common.updated': 'Updated',
    'common.workflow': 'Workflow',
    'common.pageTitleSuffix': 'OpenASE',
    'locale.english': 'English',
    'locale.chinese': 'Chinese',
    'layout.openNavigation': 'Open navigation',
    'layout.search': 'Search...',
    'layout.searchAriaLabel': 'Search',
    'layout.projectAI': 'Project AI',
    'layout.askAI': 'Ask AI',
    'layout.newTicket': 'New Ticket',
    'layout.newTicketAriaLabel': 'New ticket',
    'layout.createTicket': 'Create ticket',
    'layout.ticketCreationUnavailable': 'Ticket creation is not available.',
    'layout.toggleTheme': 'Toggle Theme',
    'layout.settings': 'Settings',
    'layout.logout': 'Logout',
    'layout.loggingOut': 'Logging out...',
    'layout.collapse': 'Collapse',
    'layout.currentLanguage': 'Current: {language}',
    'nav.dashboard': 'Dashboard',
    'nav.overview': 'Overview',
    'nav.tickets': 'Tickets',
    'nav.agents': 'Agents',
    'nav.machines': 'Machines',
    'nav.updates': 'Updates',
    'nav.activity': 'Activity',
    'nav.workflows': 'Workflows',
    'nav.skills': 'Skills',
    'nav.scheduledJobs': 'Scheduled Jobs',
    'nav.settings': 'Settings',
    'auth.loginPageTitle': 'Login',
    'auth.signInToContinue': 'Sign in to continue',
    'auth.continueWithOIDC': 'Continue with OIDC',
    'auth.or': 'or',
    'auth.localBootstrap': 'Local Bootstrap',
    'auth.pasteBundlePlaceholder': 'Paste authorization bundle here...',
    'auth.authorize': 'Authorize',
    'auth.noAuthMethodAvailable': 'No auth method available.',
    'auth.localBootstrapParseError':
      'Paste the CLI JSON, text output, or URL from `openase auth bootstrap create-link`.',
    'auth.localAuthorizationPageTitle': 'Local Authorization',
    'auth.localAuthorizationTitle': 'Local Authorization',
    'auth.localAuthorizationDescription':
      'Complete a one-time local bootstrap authorization before entering the control plane.',
    'auth.localAuthorizationInitialStatus': 'Authorize this browser as the local instance admin.',
    'auth.localAuthorizationIncomplete':
      'This authorization bundle is incomplete. Generate a fresh local bootstrap bundle from the CLI.',
    'auth.localAuthorizationAuthorizing': 'Authorizing this browser session...',
    'auth.localAuthorizationSucceeded': 'Authorization succeeded. Redirecting into OpenASE...',
    'auth.localAuthorizationNotice':
      'The URL carries only short-lived, single-use authorization material. OpenASE stores the resulting browser session in an httpOnly cookie after redemption.',
    'auth.localAuthorizationRetrying': 'Authorizing...',
    'auth.localAuthorizationRetry': 'Retry authorization',
    'dashboard.pageTitle': 'Dashboard',
    'priority.unset': 'Unset',
    'priority.urgent': 'Urgent',
    'priority.high': 'High',
    'priority.medium': 'Medium',
    'priority.low': 'Low',
    'tickets.pageTitle': 'Tickets',
    'tickets.pageDescription':
      'Track delivery status, filter active work, and open ticket details.',
    'board.searchTicketsPlaceholder': 'Search tickets...',
    'board.anomalies': 'Anomalies',
    'board.hideEmpty': 'Hide empty',
    'board.boardView': 'Board view',
    'board.listView': 'List view',
    'board.noTicketsMatch': 'No tickets match the current filters.',
    'board.noBoardStatuses': 'No board statuses configured yet.',
    'board.noStatusesInGroup': 'No statuses in this group.',
    'board.hiddenColumns': 'Hidden ({count})',
    'board.createTicketInStatus': 'Create ticket in {status}',
    'agents.pageTitle': 'Agents',
    'agents.pageDescription': 'Manage agent definitions and monitor their runs.',
    'agents.registerAgent': 'Register Agent',
    'agents.projectContextUnavailable': 'Project context is unavailable.',
    'agents.runningSummary': '{running}/{total} agents running',
    'agents.noAgentsRegistered': 'No agents registered',
    'agents.noAgentsDescription':
      "Agents are AI workers that pick up and execute tickets. Register an agent definition to connect a Claude model to this project's workflow.",
    'agents.expand': 'Expand',
    'agents.collapse': 'Collapse',
    'agents.status.idle': 'Idle',
    'agents.status.claimed': 'Claimed',
    'agents.status.running': 'Running',
    'agents.status.paused': 'Paused',
    'agents.status.failed': 'Failed',
    'agents.status.interrupted': 'Interrupted',
    'agents.status.terminated': 'Terminated',
    'agents.runtime.launching': 'Launching',
    'agents.runtime.ready': 'Ready',
    'agents.runtime.executing': 'Executing',
    'agents.runtime.completed': 'Completed',
    'agents.runtime.errored': 'Errored',
    'agents.runtime.interrupted': 'Interrupted',
    'agents.runtime.terminated': 'Terminated',
    'agents.taskCountOne': '{count} task',
    'agents.taskCountOther': '{count} tasks',
    'agents.heartbeat': 'heartbeat {time}',
    'agents.interruptRequested': 'Interrupt Requested',
    'agents.pauseRequested': 'Pause Requested',
    'agents.retired': 'Retired',
    'agents.editAgent': 'Edit agent',
    'agents.interruptAgent': 'Interrupt agent',
    'agents.interruptRun': 'Interrupt this agent run',
    'agents.resumeAgent': 'Resume agent',
    'agents.resumeThisAgent': 'Resume this agent',
    'agents.pauseAgent': 'Pause agent',
    'agents.pauseThisAgent': 'Pause this agent',
  },
  zh: {
    'common.appName': 'OpenASE',
    'common.actions': '操作',
    'common.agent': '代理',
    'common.all': '全部',
    'common.blocked': '阻塞',
    'common.cancel': '取消',
    'common.status': '状态',
    'common.priority': '优先级',
    'common.retry': '重试',
    'common.language': '语言',
    'common.links': '链接',
    'common.loading': '加载中...',
    'common.ticket': '工单',
    'common.unassigned': '未分配',
    'common.updated': '更新时间',
    'common.workflow': '工作流',
    'common.pageTitleSuffix': 'OpenASE',
    'locale.english': '英文',
    'locale.chinese': '中文',
    'layout.openNavigation': '打开导航',
    'layout.search': '搜索...',
    'layout.searchAriaLabel': '搜索',
    'layout.projectAI': '项目 AI',
    'layout.askAI': '询问 AI',
    'layout.newTicket': '新建工单',
    'layout.newTicketAriaLabel': '新建工单',
    'layout.createTicket': '创建工单',
    'layout.ticketCreationUnavailable': '当前无法创建工单。',
    'layout.toggleTheme': '切换主题',
    'layout.settings': '设置',
    'layout.logout': '退出登录',
    'layout.loggingOut': '正在退出登录...',
    'layout.collapse': '收起',
    'layout.currentLanguage': '当前：{language}',
    'nav.dashboard': '工作台',
    'nav.overview': '概览',
    'nav.tickets': '工单',
    'nav.agents': '代理',
    'nav.machines': '机器',
    'nav.updates': '更新',
    'nav.activity': '动态',
    'nav.workflows': '工作流',
    'nav.skills': '技能',
    'nav.scheduledJobs': '定时任务',
    'nav.settings': '设置',
    'auth.loginPageTitle': '登录',
    'auth.signInToContinue': '登录后继续',
    'auth.continueWithOIDC': '通过 OIDC 继续',
    'auth.or': '或',
    'auth.localBootstrap': '本地引导授权',
    'auth.pasteBundlePlaceholder': '将授权数据粘贴到这里...',
    'auth.authorize': '授权',
    'auth.noAuthMethodAvailable': '当前没有可用的认证方式。',
    'auth.localBootstrapParseError':
      '请粘贴 `openase auth bootstrap create-link` 输出的 CLI JSON、文本结果或 URL。',
    'auth.localAuthorizationPageTitle': '本地授权',
    'auth.localAuthorizationTitle': '本地授权',
    'auth.localAuthorizationDescription': '进入控制台前，请先完成一次本地引导授权。',
    'auth.localAuthorizationInitialStatus': '请将当前浏览器授权为本地实例管理员。',
    'auth.localAuthorizationIncomplete': '授权数据不完整。请重新从 CLI 生成新的本地引导授权数据。',
    'auth.localAuthorizationAuthorizing': '正在为当前浏览器会话授权...',
    'auth.localAuthorizationSucceeded': '授权成功，正在进入 OpenASE...',
    'auth.localAuthorizationNotice':
      'URL 中只包含短时有效且一次性的授权材料。兑换完成后，OpenASE 会把浏览器会话写入 httpOnly cookie。',
    'auth.localAuthorizationRetrying': '正在授权...',
    'auth.localAuthorizationRetry': '重新授权',
    'dashboard.pageTitle': '工作台',
    'priority.unset': '未设置',
    'priority.urgent': '紧急',
    'priority.high': '高',
    'priority.medium': '中',
    'priority.low': '低',
    'tickets.pageTitle': '工单',
    'tickets.pageDescription': '跟踪交付状态、筛选活跃工作，并打开工单详情。',
    'board.searchTicketsPlaceholder': '搜索工单...',
    'board.anomalies': '异常',
    'board.hideEmpty': '隐藏空列',
    'board.boardView': '看板视图',
    'board.listView': '列表视图',
    'board.noTicketsMatch': '当前筛选条件下没有匹配的工单。',
    'board.noBoardStatuses': '当前还没有配置看板状态。',
    'board.noStatusesInGroup': '当前分组中没有状态。',
    'board.hiddenColumns': '已隐藏（{count}）',
    'board.createTicketInStatus': '在 {status} 中创建工单',
    'agents.pageTitle': '代理',
    'agents.pageDescription': '管理代理定义并监控它们的运行情况。',
    'agents.registerAgent': '注册代理',
    'agents.projectContextUnavailable': '项目上下文不可用。',
    'agents.runningSummary': '运行中 {running}/{total} 个代理',
    'agents.noAgentsRegistered': '还没有注册任何代理',
    'agents.noAgentsDescription':
      '代理是负责领取并执行工单的 AI 工作单元。注册一个代理定义，将 Claude 模型接入当前项目的工作流。',
    'agents.expand': '展开',
    'agents.collapse': '收起',
    'agents.status.idle': '空闲',
    'agents.status.claimed': '已领取',
    'agents.status.running': '运行中',
    'agents.status.paused': '已暂停',
    'agents.status.failed': '失败',
    'agents.status.interrupted': '已中断',
    'agents.status.terminated': '已终止',
    'agents.runtime.launching': '启动中',
    'agents.runtime.ready': '就绪',
    'agents.runtime.executing': '执行中',
    'agents.runtime.completed': '已完成',
    'agents.runtime.errored': '出错',
    'agents.runtime.interrupted': '已中断',
    'agents.runtime.terminated': '已终止',
    'agents.taskCountOne': '{count} 个任务',
    'agents.taskCountOther': '{count} 个任务',
    'agents.heartbeat': '心跳 {time}',
    'agents.interruptRequested': '已请求中断',
    'agents.pauseRequested': '已请求暂停',
    'agents.retired': '已退役',
    'agents.editAgent': '编辑代理',
    'agents.interruptAgent': '中断代理',
    'agents.interruptRun': '中断当前代理运行',
    'agents.resumeAgent': '恢复代理',
    'agents.resumeThisAgent': '恢复当前代理',
    'agents.pauseAgent': '暂停代理',
    'agents.pauseThisAgent': '暂停当前代理',
  },
} as const satisfies Record<AppLocale, Record<string, string>>

export type TranslationKey = keyof typeof translations.en
export type TranslationParams = Record<string, string | number>

const DEFAULT_LOCALE: AppLocale = 'en'
const STORAGE_KEY = 'openase.locale'
const LOCALE_LABEL_KEYS = {
  en: 'locale.english',
  zh: 'locale.chinese',
} as const satisfies Record<AppLocale, TranslationKey>

function fillTemplate(template: string, params: TranslationParams = {}) {
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    const value = params[key]
    return value == null ? match : String(value)
  })
}

export function parseLocale(input: string | null | undefined): AppLocale {
  const normalized = input?.trim().toLowerCase() ?? ''
  if (!normalized) {
    return DEFAULT_LOCALE
  }
  if (normalized.startsWith('zh')) {
    return 'zh'
  }
  if (normalized.startsWith('en')) {
    return 'en'
  }
  return DEFAULT_LOCALE
}

export function readStoredLocale(): AppLocale {
  if (typeof window === 'undefined' || typeof window.localStorage === 'undefined') {
    return DEFAULT_LOCALE
  }
  return parseLocale(window.localStorage.getItem(STORAGE_KEY))
}

export function detectPreferredLocale(): AppLocale {
  if (typeof navigator === 'undefined') {
    return DEFAULT_LOCALE
  }
  return parseLocale(navigator.language)
}

export function resolveInitialLocale(): AppLocale {
  const stored = readStoredLocale()
  if (
    stored !== DEFAULT_LOCALE ||
    (typeof window !== 'undefined' && window.localStorage.getItem(STORAGE_KEY))
  ) {
    return stored
  }
  return detectPreferredLocale()
}

export function translate(locale: AppLocale, key: TranslationKey, params?: TranslationParams) {
  return fillTemplate(translations[locale][key] ?? translations.en[key], params)
}

export function localeLabel(locale: AppLocale, displayLocale: AppLocale) {
  return translate(displayLocale, LOCALE_LABEL_KEYS[locale])
}

export function setStoredLocale(locale: AppLocale) {
  if (typeof window === 'undefined' || typeof window.localStorage === 'undefined') {
    return
  }
  window.localStorage.setItem(STORAGE_KEY, locale)
}

export function pageTitle(title: string, locale: AppLocale) {
  return `${title} - ${translate(locale, 'common.pageTitleSuffix')}`
}
