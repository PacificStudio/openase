import { translate, type AppLocale, type TranslationKey, type TranslationParams } from '$lib/i18n'

const STEP_KEY = 'onboarding.step.githubToken' as const

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

type StepGitHubTokenCopyEntry = {
  translationKey: TranslationKey
  fallback: string
}

const copy: Record<StepGitHubTokenKey, StepGitHubTokenCopyEntry> = {
  [GITHUB_TOKEN_KEYS.welcomeTitle]: {
    translationKey: 'onboarding.step.githubToken.welcomeTitle',
    fallback: 'Welcome to {projectName}',
  },
  [GITHUB_TOKEN_KEYS.welcomeDescription]: {
    translationKey: 'onboarding.step.githubToken.welcomeDescription',
    fallback:
      'The project is created. Next, we will guide you through getting it into a runnable state. After these steps complete, an agent can start working.',
  },
  [GITHUB_TOKEN_KEYS.loadingStatus]: {
    translationKey: 'onboarding.step.githubToken.loadingStatus',
    fallback: 'Loading setup status...',
  },
  [GITHUB_TOKEN_KEYS.identityVerifiedTitle]: {
    translationKey: 'onboarding.step.githubToken.identityVerifiedTitle',
    fallback: 'GitHub identity verified',
  },
  [GITHUB_TOKEN_KEYS.identityVerifiedBody]: {
    translationKey: 'onboarding.step.githubToken.identityVerifiedBody',
    fallback: 'OpenASE will use the GitHub account {login} once you confirm it.',
  },
  [GITHUB_TOKEN_KEYS.importSuccess]: {
    translationKey: 'onboarding.step.githubToken.importSuccess',
    fallback: 'GitHub token imported from gh CLI.',
  },
  [GITHUB_TOKEN_KEYS.importFailed]: {
    translationKey: 'onboarding.step.githubToken.importFailed',
    fallback: 'Import failed. Confirm gh CLI is installed locally and already signed in.',
  },
  [GITHUB_TOKEN_KEYS.confirmIdentity]: {
    translationKey: 'onboarding.step.githubToken.confirmIdentity',
    fallback: 'Use this GitHub identity for the project',
  },
  [GITHUB_TOKEN_KEYS.changeToken]: {
    translationKey: 'onboarding.step.githubToken.changeToken',
    fallback: 'Change token',
  },
  [GITHUB_TOKEN_KEYS.connectedTitle]: {
    translationKey: 'onboarding.step.githubToken.connectedTitle',
    fallback: 'GitHub connected',
  },
  [GITHUB_TOKEN_KEYS.connectedAccount]: {
    translationKey: 'onboarding.step.githubToken.connectedAccount',
    fallback: 'Account: {login}',
  },
  [GITHUB_TOKEN_KEYS.reconfigure]: {
    translationKey: 'onboarding.step.githubToken.reconfigure',
    fallback: 'Reconfigure',
  },
  [GITHUB_TOKEN_KEYS.verifyingIdentity]: {
    translationKey: 'onboarding.step.githubToken.verifyingIdentity',
    fallback: 'Verifying GitHub identity and permissions...',
  },
  [GITHUB_TOKEN_KEYS.validationFailedTitle]: {
    translationKey: 'onboarding.step.githubToken.validationFailedTitle',
    fallback: 'Token validation failed',
  },
  [GITHUB_TOKEN_KEYS.validationFailedMessage]: {
    translationKey: 'onboarding.step.githubToken.validationFailedMessage',
    fallback: 'Check the token permissions. Repository read/write access is required.',
  },
  [GITHUB_TOKEN_KEYS.tokenRequired]: {
    translationKey: 'onboarding.step.githubToken.tokenRequired',
    fallback: 'Enter a GitHub token.',
  },
  [GITHUB_TOKEN_KEYS.saveSuccess]: {
    translationKey: 'onboarding.step.githubToken.saveSuccess',
    fallback: 'GitHub token saved.',
  },
  [GITHUB_TOKEN_KEYS.saveFailed]: {
    translationKey: 'onboarding.step.githubToken.saveFailed',
    fallback: 'Failed to save the token.',
  },
  [GITHUB_TOKEN_KEYS.validationFailedToast]: {
    translationKey: 'onboarding.step.githubToken.validationFailedToast',
    fallback: 'GitHub token validation failed. Check the token permissions.',
  },
  [GITHUB_TOKEN_KEYS.verifyFailed]: {
    translationKey: 'onboarding.step.githubToken.verifyFailed',
    fallback: 'Failed to verify GitHub identity.',
  },
  [GITHUB_TOKEN_KEYS.chooseImportTitle]: {
    translationKey: 'onboarding.step.githubToken.chooseImportTitle',
    fallback: 'Import automatically from local gh',
  },
  [GITHUB_TOKEN_KEYS.chooseImportDescription]: {
    translationKey: 'onboarding.step.githubToken.chooseImportDescription',
    fallback: 'Detect and import the local gh auth token automatically.',
  },
  [GITHUB_TOKEN_KEYS.choosePasteTitle]: {
    translationKey: 'onboarding.step.githubToken.choosePasteTitle',
    fallback: 'Paste a token manually',
  },
  [GITHUB_TOKEN_KEYS.choosePasteDescription]: {
    translationKey: 'onboarding.step.githubToken.choosePasteDescription',
    fallback: 'Run gh auth token and paste the result.',
  },
  [GITHUB_TOKEN_KEYS.importHint]: {
    translationKey: 'onboarding.step.githubToken.importHint',
    fallback:
      'Make sure GitHub CLI is installed and already signed in. OpenASE will read the current token automatically.',
  },
  [GITHUB_TOKEN_KEYS.importSignInHint]: {
    translationKey: 'onboarding.step.githubToken.importSignInHint',
    fallback: 'If you are not signed in yet:',
  },
  [GITHUB_TOKEN_KEYS.importing]: {
    translationKey: 'onboarding.step.githubToken.importing',
    fallback: 'Importing...',
  },
  [GITHUB_TOKEN_KEYS.importNow]: {
    translationKey: 'onboarding.step.githubToken.importNow',
    fallback: 'Import now',
  },
  [GITHUB_TOKEN_KEYS.pastePrompt]: {
    translationKey: 'onboarding.step.githubToken.pastePrompt',
    fallback: 'Paste the token below after you obtain it:',
  },
  [GITHUB_TOKEN_KEYS.pasteCommandHint]: {
    translationKey: 'onboarding.step.githubToken.pasteCommandHint',
    fallback: 'Run this command in the terminal to get the token:',
  },
  [GITHUB_TOKEN_KEYS.tokenPlaceholder]: {
    translationKey: 'onboarding.step.githubToken.tokenPlaceholder',
    fallback: 'ghp_xxxxxxxxxxxx',
  },
  [GITHUB_TOKEN_KEYS.saving]: {
    translationKey: 'onboarding.step.githubToken.saving',
    fallback: 'Saving...',
  },
  [GITHUB_TOKEN_KEYS.saveToken]: {
    translationKey: 'onboarding.step.githubToken.saveToken',
    fallback: 'Save token',
  },
  [GITHUB_TOKEN_KEYS.back]: {
    translationKey: 'onboarding.step.githubToken.back',
    fallback: 'Back',
  },
}

export function t(locale: AppLocale, key: StepGitHubTokenKey, params?: TranslationParams) {
  const value = copy[key]
  return value ? translate(locale, value.translationKey, params) : ''
}
