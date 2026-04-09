export type HumanAuthUser = {
  id: string
  primaryEmail: string
  displayName: string
  avatarURL?: string
}

export type HumanAuthMethod = 'oidc' | 'local_bootstrap_link'

export type HumanAuthCapabilities = {
  availableAuthMethods: HumanAuthMethod[]
  currentAuthMethod: HumanAuthMethod | ''
}

export type HumanAuthSession = {
  authMode: string
  loginRequired: boolean
  authenticated: boolean
  principalKind: string
  authCapabilities: HumanAuthCapabilities
  authConfigured: boolean
  sessionGovernanceAvailable: boolean
  canManageAuth: boolean
  issuerURL?: string
  user?: HumanAuthUser
  csrfToken?: string
  roles: string[]
  permissions: string[]
}

function emptySession(): HumanAuthSession {
  return {
    authMode: 'disabled',
    loginRequired: false,
    authenticated: false,
    principalKind: 'anonymous',
    authCapabilities: {
      availableAuthMethods: [],
      currentAuthMethod: '',
    },
    authConfigured: false,
    sessionGovernanceAvailable: false,
    canManageAuth: false,
    issuerURL: '',
    user: undefined,
    csrfToken: '',
    roles: [],
    permissions: [],
  }
}

function fallbackAuthMethod(next: Partial<HumanAuthSession>): HumanAuthMethod | '' {
  if (next.loginRequired === true) {
    return 'oidc'
  }
  if (next.loginRequired === false) {
    return 'local_bootstrap_link'
  }
  if (next.authMode === 'oidc') {
    return 'oidc'
  }
  if (next.authMode || next.principalKind || next.authenticated) {
    return 'local_bootstrap_link'
  }
  return ''
}

function normalizeAuthCapabilities(next: Partial<HumanAuthSession>): HumanAuthCapabilities {
  const availableAuthMethods = Array.isArray(next.authCapabilities?.availableAuthMethods)
    ? Array.from(
        new Set(
          next.authCapabilities.availableAuthMethods.filter(
            (value): value is HumanAuthMethod =>
              value === 'oidc' || value === 'local_bootstrap_link',
          ),
        ),
      )
    : []
  const currentAuthMethod =
    next.authCapabilities?.currentAuthMethod === 'oidc' ||
    next.authCapabilities?.currentAuthMethod === 'local_bootstrap_link'
      ? next.authCapabilities.currentAuthMethod
      : ''
  const fallbackMethod = fallbackAuthMethod(next)

  return {
    availableAuthMethods:
      availableAuthMethods.length > 0
        ? currentAuthMethod && !availableAuthMethods.includes(currentAuthMethod)
          ? [...availableAuthMethods, currentAuthMethod]
          : availableAuthMethods
        : fallbackMethod
          ? [fallbackMethod]
          : [],
    currentAuthMethod: currentAuthMethod || availableAuthMethods[0] || fallbackMethod,
  }
}

function createAuthStore() {
  let session = $state<HumanAuthSession>(emptySession())

  return {
    get session() {
      return session
    },
    get authMode() {
      return session.authMode
    },
    get authenticated() {
      return session.authenticated
    },
    get loginRequired() {
      return session.loginRequired
    },
    get principalKind() {
      return session.principalKind
    },
    get authCapabilities() {
      return session.authCapabilities
    },
    get availableAuthMethods() {
      return session.authCapabilities.availableAuthMethods
    },
    get currentAuthMethod() {
      return session.authCapabilities.currentAuthMethod
    },
    get requiresAuthorization() {
      return session.authCapabilities.availableAuthMethods.length > 0 || session.loginRequired
    },
    get usesOIDC() {
      return session.authCapabilities.currentAuthMethod === 'oidc'
    },
    get usesLocalBootstrap() {
      return session.authCapabilities.currentAuthMethod === 'local_bootstrap_link'
    },
    get authConfigured() {
      return session.authConfigured
    },
    get sessionGovernanceAvailable() {
      return session.sessionGovernanceAvailable
    },
    get canManageAuth() {
      return session.canManageAuth
    },
    get issuerURL() {
      return session.issuerURL ?? ''
    },
    get user() {
      return session.user
    },
    get csrfToken() {
      return session.csrfToken ?? ''
    },
    get roles() {
      return session.roles
    },
    get permissions() {
      return session.permissions
    },
    hydrate(next: HumanAuthSession | null | undefined) {
      session = next
        ? {
            ...emptySession(),
            ...next,
            authCapabilities: normalizeAuthCapabilities(next),
          }
        : emptySession()
    },
    clear() {
      session = emptySession()
    },
  }
}

export const authStore = createAuthStore()
