const STEP_KEY = 'onboarding.step.githubToken'

export const GITHUB_TOKEN_KEYS = {
  welcomeTitle: `${STEP_KEY}.welcomeTitle`,
  welcomeDescription: `${STEP_KEY}.welcomeDescription`,
  loadingStatus: `${STEP_KEY}.loadingStatus`,
  identityVerifiedTitle: `${STEP_KEY}.identityVerifiedTitle`,
  identityVerifiedBody: `${STEP_KEY}.identityVerifiedBody`,
  importSuccess: `${STEP_KEY}.importSuccess`,
  importFailed: `${STEP_KEY}.importFailed`,
  confirmIdentity: `${STEP_KEY}.confirmIdentity`,
  changeToken: `${STEP_KEY}.changeToken`,
  connectedTitle: `${STEP_KEY}.connectedTitle`,
  connectedAccount: `${STEP_KEY}.connectedAccount`,
  reconfigure: `${STEP_KEY}.reconfigure`,
  verifyingIdentity: `${STEP_KEY}.verifyingIdentity`,
  validationFailedTitle: `${STEP_KEY}.validationFailedTitle`,
  validationFailedMessage: `${STEP_KEY}.validationFailedMessage`,
  tokenRequired: `${STEP_KEY}.tokenRequired`,
  saveSuccess: `${STEP_KEY}.saveSuccess`,
  saveFailed: `${STEP_KEY}.saveFailed`,
  validationFailedToast: `${STEP_KEY}.validationFailedToast`,
  verifyFailed: `${STEP_KEY}.verifyFailed`,
  chooseImportTitle: `${STEP_KEY}.chooseImportTitle`,
  chooseImportDescription: `${STEP_KEY}.chooseImportDescription`,
  choosePasteTitle: `${STEP_KEY}.choosePasteTitle`,
  choosePasteDescription: `${STEP_KEY}.choosePasteDescription`,
  importHint: `${STEP_KEY}.importHint`,
  importSignInHint: `${STEP_KEY}.importSignInHint`,
  importing: `${STEP_KEY}.importing`,
  importNow: `${STEP_KEY}.importNow`,
  pastePrompt: `${STEP_KEY}.pastePrompt`,
  pasteCommandHint: `${STEP_KEY}.pasteCommandHint`,
  tokenPlaceholder: `${STEP_KEY}.tokenPlaceholder`,
  saving: `${STEP_KEY}.saving`,
  saveToken: `${STEP_KEY}.saveToken`,
  back: `${STEP_KEY}.back`,
} as const

export type StepGitHubTokenKey = (typeof GITHUB_TOKEN_KEYS)[keyof typeof GITHUB_TOKEN_KEYS]

const copy: Record<
  StepGitHubTokenKey,
  string | ((params?: Record<string, string | number>) => string)
> = {
  [GITHUB_TOKEN_KEYS.welcomeTitle]: ({ projectName } = { projectName: '' }) =>
    `Welcome to ${projectName}`,
  [GITHUB_TOKEN_KEYS.welcomeDescription]:
    'The project is created. Next, we will guide you through getting it into a runnable state. After these steps complete, an agent can start working.',
  [GITHUB_TOKEN_KEYS.loadingStatus]: 'Loading setup status...',
  [GITHUB_TOKEN_KEYS.identityVerifiedTitle]: 'GitHub identity verified',
  [GITHUB_TOKEN_KEYS.identityVerifiedBody]: ({ login } = { login: '' }) =>
    `OpenASE will use the GitHub account ${login} once you confirm it.`,
  [GITHUB_TOKEN_KEYS.importSuccess]: 'GitHub token imported from gh CLI.',
  [GITHUB_TOKEN_KEYS.importFailed]:
    'Import failed. Confirm gh CLI is installed locally and already signed in.',
  [GITHUB_TOKEN_KEYS.confirmIdentity]: 'Use this GitHub identity for the project',
  [GITHUB_TOKEN_KEYS.changeToken]: 'Change token',
  [GITHUB_TOKEN_KEYS.connectedTitle]: 'GitHub connected',
  [GITHUB_TOKEN_KEYS.connectedAccount]: ({ login } = { login: '' }) => `Account: ${login}`,
  [GITHUB_TOKEN_KEYS.reconfigure]: 'Reconfigure',
  [GITHUB_TOKEN_KEYS.verifyingIdentity]: 'Verifying GitHub identity and permissions...',
  [GITHUB_TOKEN_KEYS.validationFailedTitle]: 'Token validation failed',
  [GITHUB_TOKEN_KEYS.validationFailedMessage]:
    'Check the token permissions. Repository read/write access is required.',
  [GITHUB_TOKEN_KEYS.tokenRequired]: 'Enter a GitHub token.',
  [GITHUB_TOKEN_KEYS.saveSuccess]: 'GitHub token saved.',
  [GITHUB_TOKEN_KEYS.saveFailed]: 'Failed to save the token.',
  [GITHUB_TOKEN_KEYS.validationFailedToast]:
    'GitHub token validation failed. Check the token permissions.',
  [GITHUB_TOKEN_KEYS.verifyFailed]: 'Failed to verify GitHub identity.',
  [GITHUB_TOKEN_KEYS.chooseImportTitle]: 'Import automatically from local gh',
  [GITHUB_TOKEN_KEYS.chooseImportDescription]:
    'Detect and import the local gh auth token automatically.',
  [GITHUB_TOKEN_KEYS.choosePasteTitle]: 'Paste a token manually',
  [GITHUB_TOKEN_KEYS.choosePasteDescription]: 'Run gh auth token and paste the result.',
  [GITHUB_TOKEN_KEYS.importHint]:
    'Make sure GitHub CLI is installed and already signed in. OpenASE will read the current token automatically.',
  [GITHUB_TOKEN_KEYS.importSignInHint]: 'If you are not signed in yet:',
  [GITHUB_TOKEN_KEYS.importing]: 'Importing...',
  [GITHUB_TOKEN_KEYS.importNow]: 'Import now',
  [GITHUB_TOKEN_KEYS.pastePrompt]: 'Paste the token below after you obtain it:',
  [GITHUB_TOKEN_KEYS.pasteCommandHint]: 'Run this command in the terminal to get the token:',
  [GITHUB_TOKEN_KEYS.tokenPlaceholder]: 'ghp_xxxxxxxxxxxx',
  [GITHUB_TOKEN_KEYS.saving]: 'Saving...',
  [GITHUB_TOKEN_KEYS.saveToken]: 'Save token',
  [GITHUB_TOKEN_KEYS.back]: 'Back',
}

export function t(key: StepGitHubTokenKey, params?: Record<string, string | number>) {
  const value = copy[key]
  if (typeof value === 'function') {
    return value(params)
  }
  return value ?? ''
}
