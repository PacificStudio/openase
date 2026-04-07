# OpenASE IAM Admin Control Plane Boundaries

This document defines the steady-state boundary between the three IAM-related control planes introduced by ASE-92:

- instance admin under `/admin`
- organization admin under `/orgs/:orgId/admin`
- project settings under `/orgs/:orgId/projects/:projectId/settings`

It is the source of truth for route ownership, permission boundaries, and migration planning for ASE-93 through ASE-96.

Use it together with:

- [`iam-dual-mode-contract.md`](./iam-dual-mode-contract.md)
- [`human-auth-oidc-rbac.md`](./human-auth-oidc-rbac.md)
- [`iam-admin-console-rollout.md`](./iam-admin-console-rollout.md)

## Goals

- Separate instance-scoped IAM operations from org-scoped administration and project-scoped settings.
- Make `/admin` exclusive to `instance_admin` authority.
- Let `org_owner` and `org_admin` handle org membership and org-scoped authorization without granting instance-wide power.
- Keep project settings focused on project-local configuration and project-local authorization.
- Remove ambiguity between `disabled` and `oidc` access semantics.

## Boundary Rules

1. Instance scope owns global identity and security posture. If a change affects every org, every project, or every human login, it belongs under `/admin`.
2. Organization scope owns who belongs to an org and which org-scoped grants they hold. If a change affects one org and its descendant projects, it belongs under `/orgs/:orgId/admin`.
3. Project settings own project-local configuration. If a change only affects one project's runtime, integrations, workflows, or project-scoped bindings, it stays in project settings.
4. Membership and role binding are separate layers.
   - membership answers "who belongs to this org / project, and what lifecycle state are they in?"
   - role binding answers "which permissions are granted at instance / org / project scope?"
5. Invitation is an organization-admin concern because it creates or advances org membership lifecycle. It is not an instance-auth concern.
6. OIDC configuration is an instance-admin concern because it changes the identity system for the whole installation, not for a single org.

## Scope Ownership

| Concern | Owning scope | Primary route | Notes |
|---|---|---|---|
| `auth.mode`, OIDC draft config, enable / disable flow, bootstrap admin policy | Instance | `/admin/auth` | Global singleton for the entire OpenASE installation |
| Cached user directory, upstream identities, user disable / enable | Instance | `/admin/users` | Users are installation-wide identities, not org-owned rows |
| Global session governance, forced user-session revocation | Instance | `/admin/sessions` | Self-service session actions may still exist under `/auth/sessions`, but admin governance is instance-scoped |
| Instance auth audit, break-glass posture, instance-scoped role bindings | Instance | `/admin/security` | Includes the highest-privilege governance controls |
| Org membership lifecycle, seat state, onboarding and offboarding | Organization | `/orgs/:orgId/admin/members` | Membership is the identity relationship to the org |
| Org invitation lifecycle | Organization | `/orgs/:orgId/admin/invitations` | Invitation creates pending org membership, not global identity |
| Org-scoped role bindings such as `org_owner` / `org_admin` | Organization | `/orgs/:orgId/admin/roles` | Authorization is separate from membership lifecycle |
| Project name, description, repo wiring, workflows, agents, notifications | Project | `/orgs/:orgId/projects/:projectId/settings` | Purely project-local configuration |
| Project credentials and outbound integrations | Project | `/orgs/:orgId/projects/:projectId/settings` | "Security" here means project-owned secrets and integrations, not global human auth |
| Project-scoped role bindings such as `project_admin` | Project | `/orgs/:orgId/projects/:projectId/settings` | Keep project access near the project it governs |

## Route Tree

```text
/admin
  /admin/auth
  /admin/users
  /admin/sessions
  /admin/security

/orgs/:orgId/admin
  /orgs/:orgId/admin/members
  /orgs/:orgId/admin/invitations
  /orgs/:orgId/admin/roles

/orgs/:orgId/projects/:projectId/settings
  general
  repositories
  agents
  workflows
  notifications
  security        # project-owned credentials / outbound integrations
  access          # project-scoped role bindings
  archived
```

### Route Intent

- `/admin/auth`
  - Owns auth-mode summary, OIDC draft fields, provider test, enable / rollback guidance.
  - Never appears inside org admin or project settings because auth mode is global.
- `/admin/users`
  - Owns cached human directory, identity inspection, user disable / enable, and per-user auth diagnostics.
- `/admin/sessions`
  - Owns instance-wide session governance, including revoke-all-for-user and suspicious-session investigation.
- `/admin/security`
  - Owns instance auth audit, break-glass posture, and instance-scoped authorization controls.
- `/orgs/:orgId/admin/members`
  - Owns the org seat ledger: active, invited, suspended, removed, ownership transfer state.
- `/orgs/:orgId/admin/invitations`
  - Owns create / resend / revoke invite flows and invite history for that org.
- `/orgs/:orgId/admin/roles`
  - Owns org-scoped grants and org-admin escalation rules.
- `/orgs/:orgId/projects/:projectId/settings`
  - Keeps project-local configuration only.
  - Must not own instance auth, user directory, org memberships, or org invitations.

## Why OIDC Lives In `/admin`

OIDC belongs under `/admin` for structural reasons, not because it is "security-flavored UI":

- `auth.mode` is a single installation-wide switch.
- OIDC issuer, client, redirect, allowed domains, and bootstrap admins affect every org and project.
- The configuration must remain reachable in `disabled` mode before any real org memberships exist.
- A broken OIDC rollout needs instance-level rollback and break-glass recovery, not org-local delegation.

Therefore `/admin/auth` is instance-only and guarded by `instance_admin` authority.

## Why Invitations Live In Org Admin

Invitations belong under org admin because they are about org membership lifecycle:

- an invite decides who may join a specific org
- an invite can be valid for one org and irrelevant to another
- day-to-day onboarding should be delegated to `org_owner` / `org_admin` without giving them instance-wide auth control
- accepting an invite materializes org membership first; any resulting authorization grant is a separate org or project binding concern

Therefore `/orgs/:orgId/admin/invitations` is org-scoped, not `/admin/invitations`.

## Membership vs Role Binding

Membership and authorization must not collapse into one table or one UI concept.

### Membership

Membership records answer:

- does this subject belong to the org?
- is the membership `invited`, `active`, `suspended`, or `removed`?
- who invited them, when, and what is the invite state?

Membership is the source of truth for org visibility, seat lifecycle, and onboarding/offboarding history.

### Role Binding

Role bindings answer:

- which instance, org, or project permissions are granted?
- is the grant direct or group-backed?
- when does the grant expire?

Role bindings are the source of truth for authorization.

### Layering Rule

An invite or member-management flow may create both artifacts over time, but they stay conceptually separate:

1. invite creates a pending org-membership relationship
2. acceptance activates membership
3. admin workflows may then attach or adjust org-scoped or project-scoped role bindings

No authorization check should infer durable admin rights only from the existence of a membership row.

## Access Semantics By Auth Mode

### `auth.mode=disabled`

Disabled mode does not make admin routes anonymous. Instead, the browser operates as the canonical local subject `local_instance_admin:default`.

Rules:

- `/admin/*` is reachable without OIDC login because the active principal already has instance-admin-equivalent authority.
- `/orgs/:orgId/admin/*` and project settings are also reachable to that same local subject.
- OpenASE must not pretend that disabled mode has real `org_owner` or `org_admin` humans. The route still evaluates with local instance authority.
- If the installation later switches to OIDC, those routes begin enforcing real human membership and role bindings.

### `auth.mode=oidc`

OIDC mode requires a real human session for normal control-plane routes.

Rules:

- `/admin/*` requires an authenticated human whose effective instance roles include `instance_admin`.
- `org_owner` and `org_admin` do not get `/admin` access unless they also hold `instance_admin`.
- `/orgs/:orgId/admin/*` requires an authenticated human with org-admin authority for that org, or inherited instance-admin authority.
- Project settings require project-local authority; instance-admin and inherited org authority may satisfy that check where appropriate.
- Unauthenticated access redirects to login or returns an auth challenge according to the browser/API entrypoint.

## Permission Matrix

Legend:

- `RW`: full read/write administration for the surface
- `RO`: read-only visibility
- `Self`: only self-service endpoints outside this surface
- `-`: no access

| Surface | Disabled `local_instance_admin` | OIDC `instance_admin` | OIDC `org_owner` | OIDC `org_admin` | OIDC `project_admin` | OIDC other member / anonymous |
|---|---|---|---|---|---|---|
| `/admin/auth` | RW | RW | - | - | - | - |
| `/admin/users` | RW | RW | - | - | - | - |
| `/admin/sessions` | RW | RW | - | - | - | `Self` via `/auth/sessions` only |
| `/admin/security` | RW | RW | - | - | - | - |
| `/orgs/:orgId/admin/members` | RW | RW | RW | RW | - | - |
| `/orgs/:orgId/admin/invitations` | RW | RW | RW | RW | - | - |
| `/orgs/:orgId/admin/roles` | RW | RW | RW | Limited RW (cannot grant or revoke `org_admin` / `org_owner`) | - | - |
| Project settings: general / repos / workflows / notifications | RW | RW | RW | RW | RW | - |
| Project settings: security (project credentials / integrations) | RW | RW | RW | RW | RW | - |
| Project settings: access (project role bindings) | RW | RW | RW | RW | RW | - |

## Organization Admin Capability Matrix

`instance_admin` always overrides org-local limits. Between org roles, the boundary is:

| Org admin action | `org_owner` | `org_admin` |
|---|---|---|
| View members, invites, roles | RW | RW |
| Create / resend / revoke invitations | RW | RW |
| Activate, suspend, or remove non-owner members | RW | RW |
| Manage project-level administrators inside descendant projects | RW | RW |
| Grant or revoke `org_admin` | RW | - |
| Grant or revoke `org_owner` | RW | - |
| Transfer org ownership | RW | - |
| Remove the last remaining owner | - | - |

This keeps day-to-day org operations delegable to `org_admin` while reserving org-governance and anti-lockout actions to `org_owner`.

## Project Settings: Keep vs Move

### Keep In Project Settings

- project name, description, and metadata
- repository connections and outbound repo credentials
- agents, workflows, scheduled jobs, and notifications
- project-owned SSH keys and outbound integration secrets
- project-scoped role bindings and access review
- archived ticket and other project-local maintenance tools

### Move Out Of Project Settings

- auth mode summary and OIDC configuration
- bootstrap admin management
- instance user directory and identity diagnostics
- global session governance and user-session revocation
- org memberships and org invitations
- org-scoped role bindings
- any break-glass or installation-wide security posture control

### Naming Rule

Project settings may keep a section named "Security", but that label is limited to project-owned secrets, outbound credentials, and project runtime hardening. It must not become the catch-all surface for human IAM governance.

## Migration Map

| Current / transitional surface | Target steady-state surface | Scope owner | Follow-up ticket |
|---|---|---|---|
| Settings -> Security OIDC setup | `/admin/auth` | Instance | ASE-93 |
| Settings -> Security user directory | `/admin/users` | Instance | ASE-94 |
| Settings -> Security session governance | `/admin/sessions` | Instance | ASE-94 |
| Settings -> Security instance auth audit and instance role controls | `/admin/security` | Instance | ASE-93 / ASE-94 |
| Embedded org members UI in shared IAM view | `/orgs/:orgId/admin/members` | Organization | ASE-95 / ASE-96 |
| Embedded org invite actions in shared IAM view | `/orgs/:orgId/admin/invitations` | Organization | ASE-95 / ASE-96 |
| Embedded org role binding management in shared IAM view | `/orgs/:orgId/admin/roles` | Organization | ASE-96 |
| Project-scoped bindings inside shared IAM view | Project settings access section | Project | follow-up under ASE-91 umbrella |

## Non-Goals

This document does not redefine every concrete permission key. It defines the control-plane boundary each later implementation must respect.

If a later ticket introduces a new IAM surface, it must first answer three questions:

1. Does this change affect the whole installation, one org, or one project?
2. Is it membership lifecycle or authorization grant management?
3. Which existing control plane should own it so that `/admin`, org admin, and project settings remain non-overlapping?
