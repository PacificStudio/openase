// @ts-check

/**
 * @typedef {'mobile-supported' | 'tablet-supported' | 'desktop-only'} MobileSupportMode
 * @typedef {'button' | 'heading' | 'label' | 'link' | 'placeholder' | 'test-id' | 'text'} LocatorKind
 * @typedef {{ kind: LocatorKind, value: string, exact?: boolean }} LocatorDescriptor
 * @typedef {'ticket-drawer' | 'settings-repositories' | 'activity-filters' | 'updates-composer' | 'machines-sheet' | 'agents-register'} MobileInteractionKind
 * @typedef {{
 *   kind: MobileInteractionKind,
 *   route?: string,
 *   hash?: string,
 *   opener?: LocatorDescriptor,
 *   ready?: LocatorDescriptor,
 *   input?: LocatorDescriptor,
 *   submit?: LocatorDescriptor,
 *   searchValue?: string,
 *   expectedText?: string,
 *   filterOption?: string,
 *   expectedButton?: string,
 * }} MobileInteractionPolicy
 * @typedef {{
 *   routeId: string,
 *   routePattern: string,
 *   title: string,
 *   support: MobileSupportMode,
 *   reason?: string,
 *   smoke?: {
 *     heading: string,
 *     criticalControls: LocatorDescriptor[],
 *   },
 *   interaction?: MobileInteractionPolicy,
 * }} ProjectPageMobilePolicy
 */

/** @typedef {ProjectPageMobilePolicy} ProjectPageMobilePolicyExport */

export const mobileSupportModes = /** @type {const} */ ([
  'mobile-supported',
  'tablet-supported',
  'desktop-only',
])

export const mobileInteractionKinds = /** @type {const} */ ([
  'ticket-drawer',
  'settings-repositories',
  'activity-filters',
  'updates-composer',
  'machines-sheet',
  'agents-register',
])

/** @type {ProjectPageMobilePolicy[]} */
export const projectPageMobilePolicies = [
  {
    routeId: 'dashboard',
    routePattern: '/orgs/[orgId]/projects/[projectId]',
    title: 'Overview',
    support: 'desktop-only',
    reason:
      'The dashboard mixes dense summary panels that have not yet been committed to the phone/tablet support matrix.',
  },
  {
    routeId: 'tickets',
    routePattern: '/orgs/[orgId]/projects/[projectId]/tickets',
    title: 'Tickets',
    support: 'mobile-supported',
    smoke: {
      heading: 'Tickets',
      criticalControls: [
        { kind: 'placeholder', value: 'Search tickets...' },
        { kind: 'button', value: 'Hide empty' },
      ],
    },
    interaction: {
      kind: 'ticket-drawer',
      route: 'tickets',
      opener: { kind: 'text', value: 'ASE-101', exact: true },
      ready: { kind: 'heading', value: 'ASE-101', exact: true },
    },
  },
  {
    routeId: 'agents',
    routePattern: '/orgs/[orgId]/projects/[projectId]/agents',
    title: 'Agents',
    support: 'mobile-supported',
    smoke: {
      heading: 'Agents',
      criticalControls: [
        { kind: 'button', value: 'Register Agent' },
        { kind: 'button', value: 'coding-main', exact: true },
      ],
    },
    interaction: {
      kind: 'agents-register',
      route: 'agents',
      opener: { kind: 'button', value: 'Register Agent' },
      ready: { kind: 'heading', value: 'Register agent' },
      input: { kind: 'label', value: 'Name' },
      submit: { kind: 'button', value: 'Register agent' },
    },
  },
  {
    routeId: 'machines',
    routePattern: '/orgs/[orgId]/projects/[projectId]/machines',
    title: 'Machines',
    support: 'mobile-supported',
    smoke: {
      heading: 'Machines',
      criticalControls: [{ kind: 'button', value: 'New machine' }],
    },
    interaction: {
      kind: 'machines-sheet',
      route: 'machines',
      opener: { kind: 'button', value: 'View details' },
      ready: { kind: 'test-id', value: 'machine-editor-sheet' },
      input: { kind: 'label', value: 'Description' },
      submit: { kind: 'test-id', value: 'machine-save-button' },
      expectedText: 'Machine updated.',
    },
  },
  {
    routeId: 'updates',
    routePattern: '/orgs/[orgId]/projects/[projectId]/updates',
    title: 'Updates',
    support: 'mobile-supported',
    smoke: {
      heading: 'Updates',
      criticalControls: [
        { kind: 'label', value: 'New update body' },
        { kind: 'button', value: 'Post update' },
      ],
    },
    interaction: {
      kind: 'updates-composer',
      route: 'updates',
      input: { kind: 'label', value: 'New update body' },
      submit: { kind: 'button', value: 'Post update' },
      expectedText: 'playwright mobile update',
    },
  },
  {
    routeId: 'activity',
    routePattern: '/orgs/[orgId]/projects/[projectId]/activity',
    title: 'Activity',
    support: 'mobile-supported',
    smoke: {
      heading: 'Activity',
      criticalControls: [
        { kind: 'placeholder', value: 'Search events...' },
        { kind: 'button', value: 'All' },
      ],
    },
    interaction: {
      kind: 'activity-filters',
      route: 'activity',
      input: { kind: 'placeholder', value: 'Search events...' },
      searchValue: 'coding-main',
      filterOption: 'Agent executing',
      expectedText: 'coding-main started work.',
      expectedButton: 'Agent executing',
    },
  },
  {
    routeId: 'settings',
    routePattern: '/orgs/[orgId]/projects/[projectId]/settings',
    title: 'Settings',
    support: 'mobile-supported',
    smoke: {
      heading: 'Settings',
      criticalControls: [
        { kind: 'button', value: 'Repositories' },
        { kind: 'button', value: 'Agents' },
      ],
    },
    interaction: {
      kind: 'settings-repositories',
      route: 'settings',
      hash: '#repositories',
      opener: { kind: 'test-id', value: 'repository-open-repo-todo' },
      ready: { kind: 'test-id', value: 'repository-editor-sheet' },
      input: { kind: 'label', value: 'Default branch' },
      submit: { kind: 'test-id', value: 'repository-save-button' },
      expectedText: 'Repository updated.',
    },
  },
  {
    routeId: 'workflows',
    routePattern: '/orgs/[orgId]/projects/[projectId]/workflows',
    title: 'Workflows',
    support: 'desktop-only',
    reason:
      'The workflow editor still relies on dense side-by-side panes and code-edit affordances that are not declared mobile-safe.',
  },
  {
    routeId: 'skills',
    routePattern: '/orgs/[orgId]/projects/[projectId]/skills',
    title: 'Skills',
    support: 'desktop-only',
    reason:
      'Skill editing remains desktop-first because the file tree and editor workspace need a dedicated small-screen design pass.',
  },
  {
    routeId: 'skills/[skillId]',
    routePattern: '/orgs/[orgId]/projects/[projectId]/skills/[skillId]',
    title: 'Skill Detail',
    support: 'desktop-only',
    reason:
      'The skill detail editor requires multi-pane authoring space and currently has no mobile interaction coverage template.',
  },
  {
    routeId: 'scheduled-jobs',
    routePattern: '/orgs/[orgId]/projects/[projectId]/scheduled-jobs',
    title: 'Scheduled Jobs',
    support: 'desktop-only',
    reason:
      'Scheduled job authoring still depends on dense workflow editing controls that have not been rolled into the mobile baseline.',
  },
]

/**
 * @param {ProjectPageMobilePolicy} policy
 */
export function isResponsiveRoutePolicy(policy) {
  return policy.support !== 'desktop-only'
}

/**
 * @param {ProjectPageMobilePolicy} policy
 */
export function requiresPhoneCoverage(policy) {
  return policy.support === 'mobile-supported'
}

/**
 * @param {ProjectPageMobilePolicy} policy
 */
export function requiresTabletCoverage(policy) {
  return policy.support === 'mobile-supported' || policy.support === 'tablet-supported'
}

/**
 * @param {string} routeId
 */
export function routePatternForRouteId(routeId) {
  return routeId === 'dashboard'
    ? '/orgs/[orgId]/projects/[projectId]'
    : `/orgs/[orgId]/projects/[projectId]/${routeId}`
}
