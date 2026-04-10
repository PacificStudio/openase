# OpenASE Human Auth, OIDC, And RBAC

This document describes the current OIDC setup and RBAC behavior shipped in OpenASE.

For the long-term dual-mode IAM contract that defines how `auth.mode=disabled`
and `auth.mode=oidc` coexist, see
[`docs/en/iam-dual-mode-contract.md`](./iam-dual-mode-contract.md).
For the steady-state boundary between instance admin, org admin, and project settings, see
[`docs/en/iam-admin-boundaries.md`](./iam-admin-boundaries.md).

## Summary

OpenASE supports browser-side human authentication with:

- OIDC as the only upstream human identity source
- an OpenASE-managed `openase_session` httpOnly cookie for browser requests
- OpenASE database-backed RBAC as the authorization source of truth
- project chat, project conversations, proposal approval, and audit actors derived from the authenticated human principal

The browser never receives the upstream OIDC access token or refresh token.

## Choosing The Right Mode

Use the auth mode that matches the deployment you actually have:

- Keep `auth.mode=disabled` for personal laptops, local demos, throwaway sandboxes, and any instance where one operator is effectively the whole control plane.
- Prefer `auth.mode=oidc` once the instance is shared with a team, exposed beyond loopback, or expected to keep per-user audit, invitations, memberships, and session isolation.
- If you are a single admin but still want browser login and future multi-user readiness, `oidc + instance_admin` is a valid upgrade path. It is optional, not mandatory.
- `instance_admin` is the highest authorization role _inside_ OIDC mode. It does not replace disabled mode's local bootstrap principal.

Practical guidance:

- Continue using `disabled` when OIDC would add overhead without meaningful security or collaboration value.
- Move to `oidc` when you need real user identity, org membership lifecycle, session inventory, or auditable administrator separation.
- Treat `auth.mode=disabled` on a non-loopback / public host as a temporary emergency-only posture.

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
    session_ttl: 0s
    session_idle_ttl: 0s
```

Field notes:

- `issuer_url`: OIDC discovery base URL.
- `client_id` / `client_secret`: OAuth client credentials used for the authorization-code flow.
- `redirect_url`: must match the IdP redirect registration and the OpenASE serve URL.
- `scopes`: defaults to `openid`, `profile`, `email`, `groups`.
- `allowed_email_domains`: optional allowlist applied after ID token verification.
- `bootstrap_admin_emails`: optional email allowlist that receives the first-login `instance_admin` binding automatically.
- `session_ttl`: absolute browser session lifetime. Set `0s` to disable absolute expiry.
- `session_idle_ttl`: sliding idle timeout. Set `0s` to disable idle expiry. When `session_ttl` is positive, `session_idle_ttl` must not exceed it.

OpenASE also supports equivalent `OPENASE_AUTH_*` environment variables through the normal config loader.

## Settings UI And Explicit OIDC Enablement

OpenASE now splits human IAM across four steady-state surfaces:

- `/admin/auth`: instance auth mode, OIDC draft configuration, bootstrap admins, validation, enablement, and rollback guidance
- `/admin`: instance directory, session governance, and break-glass diagnostics
- `/orgs/:orgId/admin/*`: organization members, invitations, and org-scoped role bindings
- Project Settings -> `#access`: project-scoped bindings and effective project access
- Project Settings -> `#security`: project-owned credentials, webhook boundaries, and runtime token posture only

When OpenASE runs in `auth.mode=disabled`, `/admin/auth` intentionally keeps the local admin experience intact while exposing the OIDC rollout workflow:

- the page states that the instance is in disabled / local single-user mode
- the local bootstrap principal remains active and usable
- you can save an OIDC draft without interrupting current disabled-mode usage
- you can test provider discovery before enabling anything
- switching to OIDC requires an explicit `Enable OIDC` action

The disabled-mode setup form supports:

- issuer URL
- client ID
- client secret
- redirect URL
- scopes
- allowed email domains
- bootstrap admin emails

Current product behavior:

1. `Save draft` persists the OIDC draft to the config file and keeps the active runtime mode unchanged.
2. `Test configuration` performs provider discovery and returns actionable endpoint diagnostics and warnings.
3. `Enable OIDC` validates discovery again, writes `auth.mode=oidc`, and returns next steps.
4. The current release still requires a service restart before the new configured mode becomes active.

Project Settings -> Security still exists during the compatibility window, but only as a project-security surface plus migration guidance. It must not continue acting as the instance auth control plane.

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

- device metadata for inventory views
- absolute expiry
- idle expiry
- revoke state
- CSRF secret
- user-agent hash
- IP prefix binding

OpenASE refreshes the idle deadline on active use. Disabling a user in the OpenASE database invalidates subsequent session use immediately because authorization is reloaded from the database during session authentication.

Relevant routes:

- `GET /auth/session`: returns auth mode, current user, CSRF token, effective instance roles, and permissions.
- `GET /auth/sessions`: returns the current user's active browser session inventory, auth audit timeline, and reserved step-up metadata.
- `DELETE /auth/sessions/:id`: revokes one browser session, including the current session when needed.
- `POST /auth/sessions/revoke-all`: revokes every other browser session while preserving the current one.
- `POST /auth/users/:userId/sessions/revoke`: lets an instance-level administrator force-revoke all sessions for a specific user.
- `GET /api/v1/instance/users`: lists the cached OIDC user directory with search and status filters.
- `GET /api/v1/instance/users/:userId`: returns identities, cached groups, active-session count, and recent auth audit for one user.
- `POST /api/v1/instance/users/:userId/status`: performs an auditable user enable or disable transition and can revoke existing browser sessions immediately.
- `POST /auth/logout`: revokes the current session and clears the session cookie.
- `GET /api/v1/auth/me/permissions`: evaluates effective roles and permissions for instance, org, or project scope.

Auth audit events capture:

- login success
- login failure
- logout
- session revocation
- session expiry
- user-disabled-after-login enforcement

## CSRF Protection

Mutating browser requests use same-origin cookie sessions and are protected by:

- same-site cookies
- Origin or Referer validation
- per-session CSRF token checks on protected writes

Frontend code should obtain the CSRF token from `GET /auth/session` and send it through the normal API client for same-origin mutating requests.

When OpenASE sits behind a trusted reverse proxy or a separate local dev frontend, you can allow additional browser origins with `auth.csrf.trusted_origins`. Each entry must be an absolute origin such as `http://localhost:4173` with no path, query, or fragment.

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

Identity synchronization semantics:

- the canonical upstream identity key is `issuer + subject`
- email, display name, avatar URL, and group memberships are mutable cached fields and refresh on later login for the same `issuer + subject`
- email changes must not create a duplicate cached user when the upstream `issuer + subject` stays the same
- sign-in fails closed if a new upstream `issuer + subject` collides with an existing cached email because automatic link, unlink, and merge are not shipped yet

Current user directory boundaries:

- OpenASE currently supports one canonical upstream identity per cached user
- manual link, unlink, and merge operations are not supported yet
- if an administrator or future migration produces multiple identities for one user row, OpenASE treats that as out-of-contract state and does not expose merge automation yet

Group synchronization strategy:

- OpenASE currently ships an OIDC group cache only
- synchronized groups can back RBAC bindings directly
- a separate local group catalog is deferred to later IAM follow-up work

Deprovision lifecycle:

- manual admin disable is supported now through the instance user directory
- disabling a user revokes existing browser sessions immediately and preserves auth audit history
- future automatic deprovision sources such as upstream sync, webhook callbacks, and SCIM remain reserved follow-up integration points

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
- Human permissions are resource/action oriented. Built-in roles now expand into
  concrete keys such as `org.read`, `project.create`, `ticket_comment.update`,
  `workflow.delete`, `harness.update`, `status.read`,
  `security_setting.update`, `notification.read`, and `conversation.create`.
- List and index APIs apply the principal's effective visibility before
  returning organizations, projects, repositories, and other human-facing
  collections.

Human permissions and agent scopes are intentionally related but not shared:

- Human permissions govern browser-authenticated human actions in the control
  plane.
- Agent scopes govern issued runtime tokens such as `projects.update` or
  `tickets.update.self`.
- Similar names do not imply interchangeability; a human permission does not
  mint an agent scope, and an agent scope does not satisfy human RBAC.

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

When `auth.mode=disabled`, persistent project conversations switch to a server-defined local principal: `local-user:default`.

- this keeps conversation ownership stable across frontend dev-server ports and browser-local storage resets
- `localStorage` remains UI-only for tab layout and drafts; it is no longer the source of truth for persistent conversation ownership
- this mode is intentionally a local single-user / shared-instance fallback, not a multi-user isolation model

Audit semantics:

- normal human write actions are recorded as `user:<user-id>`
- approved conversation actions are attributed as `user:<user-id> via project-conversation:<conversation-id>`

This preserves the distinction between the human approver and the project-conversation runtime principal.

## Settings And Diagnostics

The control plane Settings view exposes the human auth state, including:

- current auth mode
- configured auth mode from disk
- issuer URL
- bootstrap admin summary
- disabled-mode local bootstrap principal guidance
- saved OIDC draft fields and explicit save / test / enable actions
- current authenticated user
- session inventory with current-device detection and revoke actions
- auth audit timeline for browser access events
- stable project-conversation owner semantics (`user:<user-id>` under OIDC, `local-user:default` when auth is disabled)
- effective roles and permissions
- the distinction between human permissions and mintable agent scopes
- instance / org / project role binding management
- organization members and invitations
- documentation links for migration, rollback, and rollout planning

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
