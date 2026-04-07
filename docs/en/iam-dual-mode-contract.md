# OpenASE IAM Dual-Mode Auth Contract And Domain Model

This document defines the long-term IAM contract introduced by ASE-77. It is the source of truth for how OpenASE preserves the existing single-user `auth.mode=disabled` product while expanding the OIDC-backed multi-user model.

## Goals

- Keep `auth.mode=disabled` as a first-class, long-lived personal deployment mode.
- Define `auth.mode=oidc` as the multi-user browser auth mode backed by real `User` and `UserIdentity` records.
- Split IAM into explicit layers: Identity, Membership, Authorization, Session, and Audit.
- Replace weak string assembly at domain boundaries with typed contracts and parsers.

## Core Decisions

1. `disabled` is not a weakened OIDC mode. It is a stable single-user product mode with no OIDC dependency and no browser login requirement.
2. `disabled` mode uses a server-defined local bootstrap principal, `local_instance_admin:default`, whose effective permissions equal `instance_admin` without pretending to be a real OIDC user.
3. `oidc` mode uses real human subjects and browser sessions. `instance_admin` remains the highest instance role, but it is an authorization role inside OIDC mode, not a replacement for disabled mode.
4. Switching from `disabled` to `oidc` is an explicit admin workflow: configure draft OIDC settings, test, and then enable. Failure must preserve or restore the disabled-mode experience.

## Auth Mode Contract

| Concern | `auth.mode=disabled` | `auth.mode=oidc` |
|---|---|---|
| Product positioning | Personal or single-user deployment | Team or multi-user deployment |
| OIDC dependency | None | Required |
| Browser login | Never required | Required for normal browser control-plane routes |
| Primary interactive subject | `local_instance_admin:default` | `user:<user-id>` |
| Effective instance authority | Equal to `instance_admin` | Determined by instance/org/project bindings |
| Real `User` / `UserIdentity` rows | Not required and must not be synthesized | Required |
| Multi-user isolation | Not supported | Supported |
| Persistent conversation owner | `local_instance_admin:default` | `user:<user-id>` |
| Interactive audit actor | `local_instance_admin:default` | `user:<user-id>` |
| Conversation-assisted audit actor | `local_instance_admin:default via project-conversation:<conversation-id>` | `user:<user-id> via project-conversation:<conversation-id>` |

### Disabled Mode Principal Semantics

Disabled mode must keep the current no-login experience intact while still exposing every admin capability needed by a personal deployment.

- Canonical subject: `SubjectRef{Kind: local_instance_admin, Key: default}`.
- Human meaning: it represents "the local operator of this OpenASE instance", not an upstream identity.
- Authorization meaning: its effective permissions are equivalent to `instance_admin` so Settings, RBAC administration, and other management actions remain available.
- Modeling rule: it must not require a row in `users` or `user_identities`.
- Session rule: a browser session may exist for CSRF, device, and revoke tracking, but the session points to the local subject rather than inventing a fake user.

### OIDC Mode Principal Semantics

OIDC mode is the multi-user path.

- Canonical interactive subject: `SubjectRef{Kind: user, Key: <user-id>}`.
- Browser session ownership, audit records, and conversation ownership derive from the authenticated user.
- `instance_admin` is the highest instance role and can be granted through role bindings or bootstrap admin logic.
- `instance_admin` does not replace disabled mode because it still requires a real OIDC user, OIDC configuration, and browser login.

## Local Bootstrap Admin And Legacy Aliases

The canonical disabled-mode subject is `local_instance_admin:default`.

During migration, OpenASE should keep reading the legacy ownership string `local-user:default` as an alias. Follow-up tickets should normalize stored ownership and audit rows to the typed subject form while preserving backward compatibility for old records.

## Settings, Session, Audit, And Conversation Ownership

### Settings Access

- In `disabled`, `/admin` and other protected writes run as `local_instance_admin:default`.
- In `oidc`, `/admin` runs as the authenticated user and authorization checks depend on the user's effective bindings.

### Session Model

- Reuse `browser_sessions`, but widen the domain model to attach a typed `SubjectRef` and device metadata.
- In `disabled`, the session identifies a browser/device but not a real human directory user.
- In `oidc`, the session binds to a real user and can be revoked independently per device.

### Audit Model

- Audit records should store typed subject data and an optional delegated conversation subject.
- Disabled-mode audit records use `local_instance_admin:default` as the actor.
- OIDC-mode audit records use `user:<user-id>` as the actor.

### Conversation Ownership

- Persistent project conversations are always server-derived.
- Disabled mode uses the stable local subject so browser-local IDs never define ownership.
- OIDC mode uses the authenticated user subject; a project conversation runtime remains a delegated runtime, not the primary human actor.

## Disabled To OIDC Enablement Flow

The mode switch is a configuration workflow, not a login side effect.

### Storage Contract

- Effective runtime mode lives in the config provider as `auth.mode`.
- Non-secret OIDC draft fields live under `auth.oidc.*` in the same config provider.
- Secrets such as `auth.oidc.client_secret` live in the secret provider. For the current file-backed local deployment that means `~/.openase/.env` or equivalent environment-backed secret material, not a value echoed back through normal Settings reads.
- Future secret stores may change the storage backend, but not the contract: secrets stay separate from the non-secret config document.

### Admin Workflow

1. Start in `auth.mode=disabled` with the local bootstrap admin subject active.
2. Open `/admin` and save draft OIDC configuration. Saving draft config does not change the current auth mode.
3. Run `Test OIDC`.
   - Validate field completeness.
   - Fetch discovery metadata and JWKS.
   - Confirm the redirect URL is syntactically valid for the current OpenASE base URL.
   - Do not create users, identities, sessions, or role bindings.
4. Click `Enable OIDC`.
   - Re-run provider initialization against the current draft config before changing the stored mode.
   - Persist `auth.mode=oidc` only after that validation succeeds.
   - Return explicit next steps when a service restart is still required to activate the new configured mode.
   - If activation fails, keep the previous disabled-mode runtime active and surface the error.
5. If production rollout fails later, explicitly set `auth.mode=disabled` again. The draft OIDC config may remain stored for later retry.

### Failure And Rollback Rules

- Failed test never changes the active mode.
- Failed enable keeps the last known good mode active.
- Disabled mode remains permanently supported as the rollback target.
- No migration step may require creating a fake local OIDC user just to recover admin access.

## Multi-User Boundary

- `disabled` is single-user semantics only.
- Multiple humans sharing a browser or machine while `disabled` is active still operate as the same local bootstrap admin subject.
- Multi-user identity, membership, invitation, and per-user session isolation are OIDC-mode capabilities.
- A personal deployment may still choose OIDC and grant itself `instance_admin`, but that is an optional deployment choice, not the product replacement for disabled mode.

## IAM Layer Model

| Layer | Responsibility | Reused models | New / changed models |
|---|---|---|---|
| Identity | Canonical people and upstream identities | `users`, `user_identities` | No disabled-mode fake user rows; add typed parse boundary around identity references |
| Membership | Human relationship to organizations and projects | Partial reuse from `role_bindings` during migration | `organization_memberships`, `organization_invitations`, membership status tracking |
| Authorization | Roles, permissions, inheritance, delegated checks | `role_bindings` | Parse into scoped role types instead of mixing all roles as weak strings |
| Session | Browser/device state, CSRF, revocation | `browser_sessions` | Add typed subject refs, device kind, optional nullable user binding for disabled mode |
| Audit | Who did what, through which runtime | Existing activity events remain reusable | auth audit event model with typed actor and delegated actor refs |

## Domain Types And Parse Boundaries

ASE-77 introduces the first typed IAM draft in `internal/domain/iam/contracts.go`.

Key draft types:

- `AuthMode` and `AuthModeContract`
- `SubjectKind` and `SubjectRef`
- `InstanceRole`, `OrganizationRole`, `ProjectRole`
- `MembershipStatus` and `InvitationStatus`
- `SessionDevice`

Parse rules:

- raw config values parse into `AuthMode`
- DB role rows parse into scope-specific role types
- owner and audit actor strings parse into `SubjectRef`
- session device labels parse into `SessionDevice`
- business logic should consume parsed domain types, not raw strings assembled ad hoc

## Data Model Reuse And Additions

### Reuse

- `users`: keep for real humans in OIDC mode only.
- `user_identities`: keep for upstream OIDC identity linkage.
- `browser_sessions`: keep as the session table, but evolve it to reference a typed subject and device metadata.
- `role_bindings`: keep as the authorization source of truth while follow-up work hardens scoped role parsing and scope correctness.

### Add

- `organization_memberships`
- `organization_invitations`
- auth audit events table or event stream payloads with typed actor refs
- optional config version / validation metadata for OIDC enablement workflow if the existing settings store cannot represent it cleanly

## State Model

### Membership

- `active` -> `suspended`
- `active` -> `revoked`
- `active` -> `left`
- `suspended` -> `active`
- `suspended` -> `revoked`
- `revoked` and `left` are terminal

### Invitation

- `pending` -> `accepted`
- `pending` -> `expired`
- `pending` -> `revoked`
- terminal states stay terminal

### Session

- browser/device sessions are separate from principal identity
- a disabled-mode session can be revoked without changing the stable local subject
- an OIDC session revocation affects only that user/device session, not the role bindings themselves

## Rollout, Backfill, And Feature Flags

### Rollout Strategy

1. Ship the typed IAM contract and documentation first.
2. Backfill storage readers so legacy owner strings such as `local-user:default` map to `local_instance_admin:default`.
3. Widen session and audit schemas behind feature flags before switching API payloads to typed subject refs.
4. Introduce organization memberships and invitations without changing disabled-mode semantics.

### Backfill

- Backfill `browser_sessions` rows with `subject_kind=user` and `subject_key=<user-id>` for existing OIDC sessions.
- Backfill conversation ownership and audit rows by aliasing legacy disabled-mode principals to `local_instance_admin:default`.
- Keep old string reads compatible until every reader understands typed subject refs.

### Suggested Feature Flags

- `iam_subject_refs`
- `iam_membership_tables`
- `iam_auth_audit`
- `iam_oidc_settings_enablement`

These flags should gate storage and API rollout, not the long-term existence of disabled mode itself.

## Follow-On Ticket Alignment

- ASE-78 consumes the scoped role model and authorization integrity rules.
- ASE-79 expands permission coverage using the scoped role types defined here.
- ASE-80 consumes the session device and audit actor contract.
- ASE-81 consumes the identity and deprovision model.
- ASE-82 consumes the membership and invitation model.
- ASE-83 consumes the settings, diagnostics, enablement, and rollout workflow.
