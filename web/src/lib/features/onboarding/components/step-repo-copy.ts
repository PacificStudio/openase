const STEP_KEY = 'onboarding.step.repo'

export const REPO_STEP_KEYS = {
  actionsAddRepository: `${STEP_KEY}.actions.addRepository`,
  createCardTitle: `${STEP_KEY}.createCard.title`,
  createCardDescription: `${STEP_KEY}.createCard.description`,
  linkCardTitle: `${STEP_KEY}.linkCard.title`,
  linkCardDescription: `${STEP_KEY}.linkCard.description`,
  formNamespace: `${STEP_KEY}.forms.namespace`,
  formRepositoryName: `${STEP_KEY}.forms.repositoryName`,
  formVisibility: `${STEP_KEY}.forms.visibility`,
  formDefaultBranch: `${STEP_KEY}.forms.defaultBranch`,
  formGitUrl: `${STEP_KEY}.forms.gitUrl`,
  placeholderRepositoryName: `${STEP_KEY}.placeholders.repositoryName`,
  placeholderDefaultBranch: `${STEP_KEY}.placeholders.defaultBranch`,
  placeholderGitUrl: `${STEP_KEY}.placeholders.gitUrl`,
  placeholderNamespace: `${STEP_KEY}.placeholders.namespace`,
  visibilityPrivate: `${STEP_KEY}.visibility.private`,
  visibilityPublic: `${STEP_KEY}.visibility.public`,
  actionsCreating: `${STEP_KEY}.actions.creating`,
  actionsCreateAndLink: `${STEP_KEY}.actions.createAndLink`,
  actionsLinking: `${STEP_KEY}.actions.linking`,
  actionsLinkRepository: `${STEP_KEY}.actions.linkRepository`,
  actionsBack: `${STEP_KEY}.actions.back`,
  searchHeading: `${STEP_KEY}.search.heading`,
  searchPlaceholder: `${STEP_KEY}.search.placeholder`,
  searchTokenScope: `${STEP_KEY}.search.tokenScope`,
  searchPermissionLimited: `${STEP_KEY}.search.permissionLimited`,
  searchNoMatch: `${STEP_KEY}.search.noMatch`,
  searchNoRepositories: `${STEP_KEY}.search.noRepositories`,
  searchLoadMore: `${STEP_KEY}.search.loadMore`,
  searchSearching: `${STEP_KEY}.search.searching`,
} as const

export type StepRepoCopyKey = (typeof REPO_STEP_KEYS)[keyof typeof REPO_STEP_KEYS]

const copy: Record<StepRepoCopyKey, string> = {
  [REPO_STEP_KEYS.actionsAddRepository]: 'Add another repository',
  [REPO_STEP_KEYS.createCardTitle]: 'Create a new repository',
  [REPO_STEP_KEYS.createCardDescription]: 'Create a new code repository on GitHub',
  [REPO_STEP_KEYS.linkCardTitle]: 'Link an existing repository',
  [REPO_STEP_KEYS.linkCardDescription]: 'Link an existing Git repository',
  [REPO_STEP_KEYS.formNamespace]: 'Namespace',
  [REPO_STEP_KEYS.formRepositoryName]: 'Repository name',
  [REPO_STEP_KEYS.formVisibility]: 'Visibility',
  [REPO_STEP_KEYS.formDefaultBranch]: 'Default branch',
  [REPO_STEP_KEYS.formGitUrl]: 'Git URL',
  [REPO_STEP_KEYS.placeholderRepositoryName]: 'my-project',
  [REPO_STEP_KEYS.placeholderDefaultBranch]: 'main',
  [REPO_STEP_KEYS.placeholderGitUrl]: 'https://github.com/owner/repo.git',
  [REPO_STEP_KEYS.visibilityPrivate]: 'Private',
  [REPO_STEP_KEYS.visibilityPublic]: 'Public',
  [REPO_STEP_KEYS.actionsCreating]: 'Creating...',
  [REPO_STEP_KEYS.actionsCreateAndLink]: 'Create and link',
  [REPO_STEP_KEYS.actionsLinking]: 'Linking...',
  [REPO_STEP_KEYS.actionsLinkRepository]: 'Link repository',
  [REPO_STEP_KEYS.actionsBack]: 'Back',
  [REPO_STEP_KEYS.searchHeading]: 'Search or browse GitHub repositories',
  [REPO_STEP_KEYS.searchPlaceholder]: 'Search accessible repositories (try owner/repo)',
  [REPO_STEP_KEYS.searchTokenScope]:
    "Repositories shown here are those your project's GitHub token can access—not every repository on GitHub.",
  [REPO_STEP_KEYS.searchPermissionLimited]:
    "Results only include repositories your project's GitHub token can access. Repos outside that scope won't appear, even if you search by name.",
  [REPO_STEP_KEYS.searchNoMatch]: 'No repositories match your search.',
  [REPO_STEP_KEYS.searchNoRepositories]: 'No repositories to browse with the current token.',
  [REPO_STEP_KEYS.searchLoadMore]: 'Load more',
  [REPO_STEP_KEYS.searchSearching]: 'Searching…',
  [REPO_STEP_KEYS.placeholderNamespace]: 'Select a namespace',
}

export function t(key: StepRepoCopyKey) {
  return copy[key] ?? ''
}
