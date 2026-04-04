export type HumanAuthUser = {
  id: string
  primaryEmail: string
  displayName: string
  avatarURL?: string
}

export type HumanAuthSession = {
  authMode: string
  authenticated: boolean
  issuerURL?: string
  user?: HumanAuthUser
  csrfToken?: string
  roles: string[]
  permissions: string[]
}

function emptySession(): HumanAuthSession {
  return {
    authMode: 'disabled',
    authenticated: false,
    issuerURL: '',
    user: undefined,
    csrfToken: '',
    roles: [],
    permissions: [],
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
      session = next ? { ...emptySession(), ...next } : emptySession()
    },
    clear() {
      session = emptySession()
    },
  }
}

export const authStore = createAuthStore()
