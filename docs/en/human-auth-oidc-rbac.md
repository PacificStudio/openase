# OpenASE Human Auth, OIDC, And RBAC

This document describes the control-plane human authentication model shipped in OpenASE.

## Summary

OpenASE supports browser-side human authentication with:

- OIDC as the only upstream human identity source
- an OpenASE-managed `openase_session` httpOnly cookie for browser requests
- OpenASE database-backed RBAC as the authorization source of truth
- project chat, project conversations, proposal approval, and audit actors derived from the authenticated human principal

The browser never receives the upstream OIDC access token or refresh token.

## Configuration

Enable human auth by setting `auth.mode=oidc` and providing the OIDC settings below:

```yaml
auth:
  mode: oidc
  oidc:
    issuer_url: https://idp.example.com/realms/openase
    client_id: openase
    client_secret: ${OIDC_CLIENT_SECRET}
    redirect_url: http://127.0.0.1:19836/api/v1/auth/oidc/callback
    scopes: ["openid", "profile", "email", "groups"]
    email_claim: email
    name_claim: name
    username_claim: preferred_username
    groups_claim: groups
    allowed_email_domains: []
    bootstrap_admin_emails: []
    session_ttl: 8h
    session_idle_ttl: 30m
```

Field notes:

- `issuer_url`: OIDC discovery base URL.
- `client_id` / `client_secret`: OAuth client credentials used for the authorization-code flow.
- `redirect_url`: must match the IdP redirect registration and the OpenASE serve URL.
- `scopes`: defaults to `openid`, `profile`, `email`, `groups`.
- `allowed_email_domains`: optional allowlist applied after ID token verification.
- `bootstrap_admin_emails`: optional email allowlist that receives the first-login `instance_admin` binding automatically.
- `session_ttl`: absolute browser session lifetime.
- `session_idle_ttl`: sliding idle timeout. It must not exceed `session_ttl`.

OpenASE also supports equivalent `OPENASE_AUTH_*` environment variables through the normal config loader.

## Browser Flow

OpenASE uses Authorization Code + PKCE:

1. The browser starts at `GET /auth/oidc/start`.
2. OpenASE stores signed login flow state in a short-lived cookie.
3. The IdP redirects back to `GET /api/v1/auth/oidc/callback`.
4. OpenASE performs discovery, exchanges the code server-side, verifies the ID token, synchronizes the local user cache, and creates a browser session.
5. The browser receives only the OpenASE `openase_session` cookie and uses same-origin requests after login.

Anonymous access is intentionally limited to setup, health checks, and auth entrypoints. Under `auth.mode=oidc`, the regular `/api/v1/...` browser control-plane routes require a valid human session.

## Session Model

Browser sessions are stored in the `browser_sessions` table with:

- absolute expiry
- idle expiry
- revoke state
- CSRF secret
- user-agent hash
- IP prefix binding

OpenASE refreshes the idle deadline on active use. Disabling a user in the OpenASE database invalidates subsequent session use immediately because authorization is reloaded from the database during session authentication.

Relevant routes:

- `GET /auth/session`: returns auth mode, current user, CSRF token, effective instance roles, and permissions.
- `POST /auth/logout`: revokes the current session and clears the session cookie.
- `GET /api/v1/auth/me/permissions`: evaluates effective roles and permissions for instance, org, or project scope.

## CSRF Protection

Mutating browser requests use same-origin cookie sessions and are protected by:

- same-site cookies
- Origin or Referer validation
- per-session CSRF token checks on protected writes

Frontend code should obtain the CSRF token from `GET /auth/session` and send it through the normal API client for same-origin mutating requests.

## User Cache And Identity Sync

Successful OIDC login upserts:

- `users`
- `user_identities`
- `user_group_memberships`
- `browser_sessions`

OpenASE stores the local authorization cache and group memberships in its own database so that:

- profile changes can be refreshed on later logins
- groups can back OpenASE role bindings
- disabled users can be blocked by OpenASE regardless of stale upstream browser state

## RBAC Model

Role bindings are stored in the database and evaluated by scope:

- `instance`
- `organization`
- `project`

Subjects can be:

- `user`
- `group`

Current built-in roles:

- `instance_admin`
- `org_owner`
- `org_admin`
- `org_member`
- `project_admin`
- `project_operator`
- `project_reviewer`
- `project_member`
- `project_viewer`

RBAC evaluation rules:

- OpenASE database bindings are the source of truth.
- OIDC claims do not directly grant roles.
- Direct user bindings and group bindings union together.
- Organization bindings inherit downward into descendant project scopes.
- Permissions are default deny.

Role bindings can be managed through:

- `GET/POST/DELETE /api/v1/organizations/:orgId/role-bindings`
- `GET/POST/DELETE /api/v1/projects/:projectId/role-bindings`

## Bootstrap Admin

When `bootstrap_admin_emails` contains the email of an authenticating user, OpenASE ensures that user has an `instance_admin` role binding on login.

Use this to avoid locking yourself out on the first OIDC-enabled deployment:

1. Set `auth.mode=oidc`.
2. Add at least one trusted administrator email to `bootstrap_admin_emails`.
3. Deploy OpenASE.
4. Complete the first browser login with that account.
5. Use Settings or the RBAC APIs to create the steady-state org and project bindings you want.

After bootstrap is complete, you can narrow or clear the bootstrap list.

## Chat, Conversations, And Audit Actors

AI session ownership is always derived from a server-defined principal:

- in `auth.mode=oidc`, project chat, project conversations, and other browser-driven AI flows use the authenticated human principal
- in `auth.mode=disabled`, the server issues and reuses an `openase_ai_principal` browser cookie whose value is a stable `browser-session:<uuid>` principal

Browser-local random ids and `X-OpenASE-Chat-User` request headers are no longer authoritative owner inputs.

Audit semantics:

- normal human write actions are recorded as `user:<user-id>`
- approved conversation actions are attributed as `user:<user-id> via project-conversation:<conversation-id>`

This preserves the distinction between the human approver and the project-conversation runtime principal.

## Settings And Diagnostics

The control plane Settings view exposes the human auth state, including:

- current auth mode
- issuer URL
- current authenticated user
- effective roles and permissions
- org/project role binding management

`GET /auth/session` and `GET /api/v1/auth/me/permissions` are the API equivalents for scripting and diagnostics.

## Troubleshooting

Common checks:

- `auth.mode=oidc` but `/login` loops back immediately:
  Verify `issuer_url`, `client_id`, `client_secret`, and `redirect_url`, and confirm the IdP redirect registration matches the absolute callback URL.
- Login succeeds at the IdP but OpenASE rejects access:
  Check `allowed_email_domains`, user disabled state, and whether the user has any OpenASE role bindings for the target scope.
- Login works but project pages return `403 AUTHORIZATION_DENIED`:
  Inspect `GET /api/v1/auth/me/permissions?project_id=<project-id>` to confirm the effective org/project bindings.
- Logout appears to fail:
  Ensure browser requests include the CSRF token from `GET /auth/session` and originate from the same site.
