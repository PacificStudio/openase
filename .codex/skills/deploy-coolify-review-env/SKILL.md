---
name: deploy-coolify-review-env
description:
  Create or update a branch-scoped Coolify review environment with one command,
  and delete it with one command. Use when a ticket must deploy a preview
  before human review, while keeping the workflow simple and fast.
---

# Deploy Coolify Review Env

## Goal

- Deploy the current ticket branch into a deterministic Coolify preview
  environment with a single command.
- Delete the same preview environment with a single command.
- Keep all configuration env-driven so the skill stays lightweight and reusable.
- Default to a single fixed Coolify project/environment pair so agents can call
  the scripts directly without reconstructing deployment coordinates each run.

## Required Runtime Environment

Set these environment variables either in the skill-local `.env` file or on the
agent machine / provider runtime:

- `COOLIFY_BASE_URL`
- `COOLIFY_API_TOKEN`
- `COOLIFY_PROJECT_UUID`
- `COOLIFY_SERVER_UUID`
- `COOLIFY_DESTINATION_UUID`
- `COOLIFY_GIT_REPOSITORY`
- `COOLIFY_PORTS_EXPOSES`
- `COOLIFY_ENVIRONMENT_NAME`
- `COOLIFY_TEMPLATE_APPLICATION_UUID`

### Repository Access Mode

Set `COOLIFY_REPOSITORY_MODE` to one of:

- `public` (default)
- `private-github-app`
- `private-deploy-key`

If `COOLIFY_REPOSITORY_MODE=private-github-app`, also set:

- `COOLIFY_GITHUB_APP_UUID`

If `COOLIFY_REPOSITORY_MODE=private-deploy-key`, also set:

- `COOLIFY_PRIVATE_KEY_UUID`

## Optional Runtime Environment

- `COOLIFY_BUILD_PACK` (default `dockerfile`)
- `COOLIFY_BASE_DIRECTORY`
- `COOLIFY_PUBLISH_DIRECTORY`
- `COOLIFY_DOCKERFILE_LOCATION`
- `COOLIFY_INSTALL_COMMAND`
- `COOLIFY_BUILD_COMMAND`
- `COOLIFY_START_COMMAND`
- `COOLIFY_HEALTH_CHECK_ENABLED`
- `COOLIFY_HEALTH_CHECK_PATH`
- `COOLIFY_HEALTH_CHECK_PORT`
- `COOLIFY_AUTOGENERATE_DOMAIN`
- `COOLIFY_DOMAIN_TEMPLATE` where `{{name}}` becomes the generated app name
- `COOLIFY_DESCRIPTION_PREFIX` (default `OpenASE review env`)
- `COOLIFY_ENV_PREFIX` (default `review`)
- `COOLIFY_NAME_MAX_LENGTH` (default `48`)

## Secret Storage

- The scripts automatically load `.codex/skills/deploy-coolify-review-env/.env`
  when the skill is injected into a runtime.
- Explicit runtime environment variables override values from the skill-local
  `.env`.
- Start from `.env.example`, then fill the real secrets into `.env`.

## Naming Model

- If `COOLIFY_ENVIRONMENT_NAME` is set, the scripts always deploy into that
  fixed Coolify environment.
- Otherwise the scripts derive the review environment name from the branch:
  - `review-<slugged-branch>`
- The application name defaults to the same value.
- You can override with `--env-name` or `--app-name` if needed.

## Current Default Wiring

This skill is currently wired for:

- Coolify project: `My first project`
- Coolify environment: `review`
- Template application for runtime env + file mount sync: `openase-main`
- Review domain template: `http://{{name}}.100.90.170.69.sslip.io`

Application names still remain branch-scoped so each review deployment gets its
own temporary application under the same environment.

## Commands

Deploy or update a review environment:

```sh
.codex/skills/deploy-coolify-review-env/scripts/deploy_review_env.sh --branch feature/my-change
```

Delete the same review environment:

```sh
.codex/skills/deploy-coolify-review-env/scripts/delete_review_env.sh --branch feature/my-change
```

Delete ticket-scoped apps inside the shared `review` environment by ticket identifier:

```sh
.codex/skills/deploy-coolify-review-env/scripts/delete_review_env.sh --env-name review --ticket-identifier ASE-184
```

## Notes

- The deploy command is idempotent:
  - if the app does not exist, it creates it
  - if the app already exists, it updates the branch and triggers a new deploy
- Before deployment, the skill syncs runtime envs and template file storages
  such as the production `codex-config.toml` / `codex` wrapper mounts
- The delete command is also idempotent:
  - if the app or environment is already missing, it exits successfully
- Use `--wait-seconds` on deploy when a workflow should wait for a terminal
  deployment state before moving a ticket to human review.
