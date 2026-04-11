import type { SecurityAuthSettings } from '$lib/api/contracts'

export type OIDCFormState = {
  issuerURL: string
  clientID: string
  clientSecret: string
  redirectMode: 'auto' | 'fixed'
  fixedRedirectURL: string
  scopesText: string
  allowedDomainsText: string
  bootstrapAdminEmailsText: string
  sessionTTL: string
  sessionIdleTTL: string
}

export type OIDCDraftMutationBody = {
  issuer_url: string
  client_id: string
  client_secret: string
  redirect_mode: 'auto' | 'fixed'
  fixed_redirect_url: string
  scopes: string[]
  allowed_email_domains: string[]
  bootstrap_admin_emails: string[]
  session_ttl: string
  session_idle_ttl: string
}

export const oidcSessionFieldCopy = {
  sessionTTLDescription:
    'Absolute browser session lifetime. Use Go durations like 8h, 30m, or 0. 0 or 0s means never expires.',
  sessionIdleTTLDescription:
    'Sliding timeout after no browser activity. When Session TTL is greater than 0, Idle TTL must be less than or equal to Session TTL. 0 or 0s means never expires.',
}

export function createOIDCFormState(): OIDCFormState {
  return {
    issuerURL: '',
    clientID: '',
    clientSecret: '',
    redirectMode: 'auto',
    fixedRedirectURL: '',
    scopesText: '',
    allowedDomainsText: '',
    bootstrapAdminEmailsText: '',
    sessionTTL: '',
    sessionIdleTTL: '',
  }
}

export function oidcDraftSignature(auth: SecurityAuthSettings): string {
  return JSON.stringify({
    oidc_draft: auth.oidc_draft,
    session_policy: auth.session_policy,
  })
}

export function oidcDraftFormFromAuth(auth: SecurityAuthSettings): OIDCFormState {
  return {
    issuerURL: auth.oidc_draft.issuer_url,
    clientID: auth.oidc_draft.client_id,
    clientSecret: '',
    redirectMode: auth.oidc_draft.redirect_mode === 'fixed' ? 'fixed' : 'auto',
    fixedRedirectURL: auth.oidc_draft.fixed_redirect_url,
    scopesText: auth.oidc_draft.scopes.join('\n'),
    allowedDomainsText: auth.oidc_draft.allowed_email_domains.join('\n'),
    bootstrapAdminEmailsText: auth.oidc_draft.bootstrap_admin_emails.join('\n'),
    sessionTTL: auth.session_policy.session_ttl,
    sessionIdleTTL: auth.session_policy.session_idle_ttl,
  }
}

export function parseListInput(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export function oidcDraftPayloadFromForm(form: OIDCFormState): OIDCDraftMutationBody {
  return {
    issuer_url: form.issuerURL.trim(),
    client_id: form.clientID.trim(),
    client_secret: form.clientSecret.trim(),
    redirect_mode: form.redirectMode,
    fixed_redirect_url: form.fixedRedirectURL.trim(),
    scopes: parseListInput(form.scopesText),
    allowed_email_domains: parseListInput(form.allowedDomainsText),
    bootstrap_admin_emails: parseListInput(form.bootstrapAdminEmailsText),
    session_ttl: form.sessionTTL.trim(),
    session_idle_ttl: form.sessionIdleTTL.trim(),
  }
}
