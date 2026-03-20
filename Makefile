SHELL := /bin/sh

WEB_DIR := web
GO ?= $(shell if [ -x "$(CURDIR)/.tooling/go/bin/go" ]; then printf '%s' "$(CURDIR)/.tooling/go/bin/go"; elif command -v go >/dev/null 2>&1; then command -v go; else printf '%s' "go"; fi)
GOFMT ?= $(shell if [ -x "$(CURDIR)/.tooling/go/bin/gofmt" ]; then printf '%s' "$(CURDIR)/.tooling/go/bin/gofmt"; elif command -v gofmt >/dev/null 2>&1; then command -v gofmt; else printf '%s' "gofmt"; fi)
NPM ?= npm
LINT_SCRIPT := ./scripts/ci/lint.sh
OPENASE_MAIN := ./cmd/openase
OPENASE_BIN := ./bin/openase

.DEFAULT_GOAL := help

.PHONY: help format fmt-check test check hooks-install hooks-run web-install web-lint web-format-check web-check web-validate web-build build build-web run doctor lint lint-all lint-depguard

help:
	@printf '%s\n' \
		'Available targets:' \
		'  make format        Format tracked Go files with gofmt' \
		'  make fmt-check     Fail if tracked Go files need gofmt' \
		'  make test          Run the Go test suite' \
		'  make check         Run Go formatting and test checks' \
		'  make hooks-install Install Git hooks via lefthook' \
		'  make hooks-run     Run the pre-commit hook against all files' \
		'  make web-install   Install frontend dependencies with npm ci' \
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
		'  make lint-depguard Run only depguard lint checks'

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
	$(GO) test ./...

check: fmt-check test

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

web-install:
	$(NPM) --prefix $(WEB_DIR) ci

web-lint: web-install
	$(NPM) --prefix $(WEB_DIR) run lint

web-format-check: web-install
	$(NPM) --prefix $(WEB_DIR) run format:check

web-check: web-install
	$(NPM) --prefix $(WEB_DIR) run check

web-validate: web-format-check web-lint web-check

web-build: web-install
	$(NPM) --prefix $(WEB_DIR) run build

build:
	@mkdir -p $(dir $(OPENASE_BIN))
	$(GO) build -o $(OPENASE_BIN) $(OPENASE_MAIN)

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
