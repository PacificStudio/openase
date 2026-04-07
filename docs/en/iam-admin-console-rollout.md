# OpenASE IAM Admin Console Rollout Checklist

This checklist documents how to roll out the full IAM admin console safely across both supported auth modes: `disabled` and `oidc`.

During the current transition, some controls may still appear together in Settings -> Security. The target control-plane ownership is defined in [`iam-admin-boundaries.md`](./iam-admin-boundaries.md) and should guide all follow-up implementation.

Use it together with:

- [`human-auth-oidc-rbac.md`](./human-auth-oidc-rbac.md)
- [`iam-dual-mode-contract.md`](./iam-dual-mode-contract.md)
- [`iam-admin-boundaries.md`](./iam-admin-boundaries.md)

## Rollout Goals

- Keep personal or local single-user deployments simple under `auth.mode=disabled`.
- Add a complete operator-facing IAM console for OIDC deployments.
- Make the OIDC switch explicit, testable, and reversible.
- Validate diagnostics, RBAC, sessions, user directory, memberships, and invitations before broader exposure.

## Surfaces Covered By The Admin Console

Settings -> Security now provides:

- active auth mode and configured auth mode
- issuer and bootstrap admin summary
- explicit disabled-mode guidance and public-exposure warnings
- OIDC draft form with save / test / enable actions
- effective access across instance, organization, and project scopes
- role bindings
- session inventory and audit summary
- user directory
- organization members and invitations
- rollout and rollback documentation links

## Deployment Choice Matrix

### Keep `auth.mode=disabled` when:

- the instance is local-only or loopback-bound
- one operator effectively owns the whole deployment
- you do not need separate human identities, invitations, or session isolation
- you want the simplest possible setup and are comfortable with local-admin semantics

### Move to `auth.mode=oidc` when:

- the instance is shared by multiple humans
- the control plane is exposed beyond loopback
- you need user-level audit attribution
- you need organization members, invitations, or per-user session governance
- you need `instance_admin` and lower roles to be granted to real users instead of the disabled-mode local principal

## Pre-Rollout Checklist

Before touching the mode:

1. Confirm the target base URL, redirect URL, and TLS plan.
2. Register the OpenASE callback URL in the OIDC provider.
3. Decide the initial bootstrap admin email list.
4. Decide the initial organization and project role-binding model.
5. Confirm rollback access: the operator must know how to revert `auth.mode` to `disabled`.

## Disabled-Mode Validation

For personal and local deployments, validate that the Security page:

- clearly states that disabled mode is active
- explains that the current operator already has local highest privilege
- does not force OIDC setup
- warns when the instance is not loopback-bound
- allows saving and testing OIDC without interrupting current usage

This ensures that disabled mode remains a first-class product path rather than a degraded migration screen.

## OIDC Draft And Enablement Checklist

1. Open Settings -> Security while still on `auth.mode=disabled`.
2. Fill in:
   - issuer URL
   - client ID
   - client secret
   - redirect URL
   - scopes
   - allowed email domains
   - bootstrap admin emails
3. Click `Save draft`.
   - Expected result: the draft is stored, but the active auth mode remains `disabled`.
4. Click `Test configuration`.
   - Expected result: provider discovery succeeds and returns issuer / authorization / token endpoint diagnostics.
5. Click `Enable OIDC`.
   - Expected result: the configured mode changes to `oidc`, and the UI returns next steps.
6. Restart the service if the UI reports that restart is required.
7. Perform the first OIDC login with a bootstrap admin account.
8. Verify that the bootstrap user receives `instance_admin`.

## Post-Enable Verification

After the first successful OIDC login, verify the following from the product UI:

- issuer and mode summary are correct
- the current user is the expected real OIDC user
- instance / organization / project effective access panels are populated
- role bindings can be inspected and updated
- session inventory shows the current browser session
- user directory lists synchronized users
- organization members and invitations are available
- audit / diagnostics summary reflects the new login path

## Migration Checklist From Basic OIDC+RBAC To The Full Console

Use this when upgrading an existing OIDC-backed instance:

1. Confirm existing OIDC settings still match the provider.
2. Open the Security page and verify the stored issuer, scopes, and redirect URL.
3. Confirm bootstrap admin emails are still correct for break-glass recovery.
4. Review instance, organization, and project bindings for administrators and operators.
5. Validate session inventory and user directory data against expected users.
6. Validate organization memberships and invitations.
7. Review audit / diagnostics summary after a fresh login.
8. Remove or narrow bootstrap admin emails once steady-state RBAC is confirmed.

## Rollback Plan

Rollback must be explicit and fast:

1. If OIDC login or authorization fails during rollout, revert `auth.mode` to `disabled`.
2. Restart the service if required by the deployment model.
3. Confirm that the Security page again shows disabled mode and the local admin principal.
4. Keep the saved OIDC draft so you can retry after fixing the provider or RBAC issue.
5. Record the failure cause before attempting another enablement.

Rollback must never require fabricating a local OIDC user.

## Recommended Validation Matrix

Validate both product modes before wider rollout:

### `disabled`

- Security page renders the auth setup panel
- saving OIDC draft does not change the active mode
- testing OIDC returns discovery diagnostics
- public-host warning appears when appropriate

### `oidc`

- first bootstrap login grants `instance_admin`
- effective access panels match expected bindings
- role-binding CRUD works across instance, org, and project scopes
- session inventory and revoke actions work
- user directory and membership diagnostics load
- organization invites can be sent and managed

## Operational Notes

- `Save draft` is intentionally non-destructive.
- `Enable OIDC` changes the configured mode only after discovery validation succeeds.
- The current implementation may still require a restart before the configured mode becomes active.
- Disabled mode remains a supported long-term operating mode, not merely a migration fallback.
