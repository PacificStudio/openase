SHELL := /bin/sh

WEB_DIR := web
GO ?= $(shell if [ -x "$(CURDIR)/.tooling/go/bin/go" ]; then printf '%s' "$(CURDIR)/.tooling/go/bin/go"; elif command -v go >/dev/null 2>&1; then command -v go; else printf '%s' "go"; fi)
GOFMT ?= $(shell if [ -x "$(CURDIR)/.tooling/go/bin/gofmt" ]; then printf '%s' "$(CURDIR)/.tooling/go/bin/gofmt"; elif command -v gofmt >/dev/null 2>&1; then command -v gofmt; else printf '%s' "gofmt"; fi)
PNPM ?= corepack pnpm
LINT_SCRIPT := ./scripts/ci/lint.sh
OPENASE_MAIN := ./cmd/openase
OPENASE_BIN := ./bin/openase
VERSION ?= dev

.DEFAULT_GOAL := help

.PHONY: help format fmt-check test test-backend-coverage check hooks-install hooks-run openapi-generate openapi-check openapi-check-ci frontend-api-audit-check web-install web-lint web-format-check web-check web-validate web-build build build-web run doctor lint lint-all lint-depguard lint-architecture

help:
	@printf '%s\n' \
		'Available targets:' \
		'  make format        Format tracked Go files with gofmt' \
		'  make fmt-check     Fail if tracked Go files need gofmt' \
		'  make test          Run the Go test suite' \
		'  make test-backend-coverage Run full backend tests plus domain/core 100% coverage gate (set OPENASE_ENABLE_FULL_BACKEND_COVERAGE=true for optional overall 75%+ metric)' \
		'  make check         Run Go formatting and enforced backend coverage checks' \
		'  make hooks-install Install Git hooks via lefthook' \
		'  make hooks-run     Run the pre-commit hook against all files' \
		'  make openapi-generate Regenerate api/openapi.json and frontend generated API types' \
		'  make openapi-check Regenerate OpenAPI artifacts and fail if git diff is non-empty' \
		'  make openapi-check-ci Regenerate OpenAPI artifacts without a full web install and fail if git diff is non-empty' \
		'  make frontend-api-audit-check Fail when backend APIs are unbound, contract-only, or wrapped-but-unused in the frontend' \
		'  make web-install   Install frontend dependencies with pnpm install --frozen-lockfile' \
		'  make web-lint      Run frontend ESLint checks' \
		'  make web-format-check Verify frontend formatting with Prettier' \
		'  make web-check     Run the Svelte type checks' \
		'  make web-validate  Run frontend format, lint, and type checks' \
		'  make web-build     Rebuild embedded frontend assets' \
		'  make build         Build openase from the current embedded frontend output' \
		'  make build-web     Rebuild frontend assets, then build openase' \
		'  make run           Run the API server with the current embedded frontend output' \
		'  make doctor        Run local environment diagnostics' \
		'  make lint          Run lint on changes since merge-base with origin/main' \
		'  make lint-all      Run the full lint suite' \
		'  make lint-depguard Run only depguard lint checks' \
		'  make lint-architecture Run repository architecture guard checks'

format:
	@files="$$(git ls-files '*.go')"; \
	if [ -z "$$files" ]; then \
		exit 0; \
	fi; \
	$(GOFMT) -w $$files

fmt-check:
	@set -e; \
	files="$$(git ls-files '*.go')"; \
	if [ -z "$$files" ]; then \
		exit 0; \
	fi; \
	diff="$$($(GOFMT) -l $$files)"; \
	if [ -n "$$diff" ]; then \
		printf 'Run `make format` for:\n%s\n' "$$diff"; \
		exit 1; \
	fi

test:
	./scripts/ci/with_clean_openase_test_env.sh $(GO) test ./...

test-backend-coverage:
	./scripts/ci/backend_coverage.sh

check: fmt-check test-backend-coverage

hooks-install:
	$(GO) tool lefthook install
	@for hook in .git/hooks/*; do \
		case "$$hook" in \
			*.sample|*.old) continue ;; \
		esac; \
		[ -f "$$hook" ] || continue; \
		grep -q 'call_lefthook' "$$hook" || continue; \
		tmp=$$(mktemp "$$hook.XXXXXX"); \
		{ \
			printf '#!/bin/sh\n'; \
			printf 'export PATH="%s"\n' "$$PATH"; \
			printf 'export LEFTHOOK_BIN="%s/scripts/lefthook.sh"\n' "$(CURDIR)"; \
			printf 'export OPENASE_GO="%s"\n' "$(GO)"; \
			printf 'export OPENASE_GOFMT="%s"\n' "$(GOFMT)"; \
			sed -E '/^(export )?(PATH|LEFTHOOK_BIN|OPENASE_GO|OPENASE_GOFMT)=/d' "$$hook" | sed '1d'; \
		} > "$$tmp"; \
		mv "$$tmp" "$$hook"; \
		chmod +x "$$hook"; \
	done

hooks-run:
	$(GO) tool lefthook run pre-commit --all-files --no-auto-install

openapi-generate: web-install
	$(GO) run $(OPENASE_MAIN) openapi generate --output api/openapi.json
	$(PNPM) --dir $(WEB_DIR) run api:generate

openapi-check: openapi-generate
	git diff --exit-code -- api/openapi.json web/src/lib/api/generated/openapi.d.ts

openapi-check-ci:
	$(GO) run $(OPENASE_MAIN) openapi generate --output api/openapi.json
	@tmp_dir="$$(mktemp -d)"; \
	trap 'rm -rf "$$tmp_dir"' EXIT; \
	printf '{ "private": true }\n' > "$$tmp_dir/package.json"; \
	$(PNPM) --dir "$$tmp_dir" add --save-dev --ignore-scripts --reporter=append-only openapi-typescript@7.13.0 prettier@3.8.1; \
	"$$tmp_dir/node_modules/.bin/openapi-typescript" "$(CURDIR)/api/openapi.json" -o "$(CURDIR)/web/src/lib/api/generated/openapi.d.ts"; \
	node "$$tmp_dir/node_modules/prettier/bin/prettier.cjs" \
		--log-level warn \
		--no-config \
		--semi false \
		--single-quote true \
		--trailing-comma all \
		--print-width 100 \
		--tab-width 2 \
		--use-tabs false \
		--write "$(CURDIR)/web/src/lib/api/generated/openapi.d.ts"
	git diff --exit-code -- api/openapi.json web/src/lib/api/generated/openapi.d.ts

frontend-api-audit-check:
	python3 ./scripts/dev/audit_frontend_api_usage.py \
		--summary-only \
		--fail-on backend_only \
		--fail-on contract_only \
		--fail-on wrapped_but_unused

web-install:
	$(PNPM) --dir $(WEB_DIR) install --frozen-lockfile --reporter=append-only

web-lint: web-install
	$(PNPM) --dir $(WEB_DIR) run lint

web-format-check: web-install
	$(PNPM) --dir $(WEB_DIR) run format:check

web-check: web-install
	$(PNPM) --dir $(WEB_DIR) run check

web-validate: web-format-check web-lint web-check

web-build: web-install
	$(PNPM) --dir $(WEB_DIR) run build

build:
	@mkdir -p $(dir $(OPENASE_BIN))
	$(GO) build -ldflags '-X main.version=$(VERSION)' -o $(OPENASE_BIN) $(OPENASE_MAIN)

build-web: web-build build

run:
	$(GO) run $(OPENASE_MAIN) serve

doctor:
	$(GO) run $(OPENASE_MAIN) doctor

lint:
	OPENASE_LINT_NEW_FROM_REV=$${LINT_BASE_REV:-$$(git merge-base origin/main HEAD)} $(LINT_SCRIPT)

lint-all:
	$(LINT_SCRIPT)

lint-depguard:
	$(LINT_SCRIPT) --enable-only=depguard ./...

lint-architecture:
	python3 ./scripts/ci/architecture_guard.py
