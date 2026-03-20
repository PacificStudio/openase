SHELL := /bin/sh

WEB_DIR := web
GO ?= $(shell if [ -x "$(CURDIR)/.tooling/go/bin/go" ]; then printf '%s' "$(CURDIR)/.tooling/go/bin/go"; elif command -v go >/dev/null 2>&1; then command -v go; else printf '%s' "go"; fi)
GOFMT ?= $(shell if [ -x "$(CURDIR)/.tooling/go/bin/gofmt" ]; then printf '%s' "$(CURDIR)/.tooling/go/bin/gofmt"; elif command -v gofmt >/dev/null 2>&1; then command -v gofmt; else printf '%s' "gofmt"; fi)
NPM ?= npm
OPENASE_MAIN := ./cmd/openase

.DEFAULT_GOAL := help

.PHONY: help format fmt-check test check hooks-install hooks-run web-install web-check web-build build build-web run doctor

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
		'  make web-check     Run the Svelte type checks' \
		'  make web-build     Rebuild embedded frontend assets' \
		'  make build         Build openase from committed embedded assets' \
		'  make build-web     Rebuild frontend assets, then build openase' \
		'  make run           Run the API server with committed embedded assets' \
		'  make doctor        Run local environment diagnostics'

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
	@hook=.git/hooks/pre-commit; \
	tmp=$$(mktemp .git/hooks/pre-commit.XXXXXX); \
	{ \
		printf '#!/bin/sh\n'; \
		printf 'export PATH="%s"\n' "$$PATH"; \
		printf 'export LEFTHOOK_BIN="%s/scripts/lefthook.sh"\n' "$(CURDIR)"; \
		printf 'export OPENASE_GO="%s"\n' "$(GO)"; \
		printf 'export OPENASE_GOFMT="%s"\n' "$(GOFMT)"; \
		sed '/^export PATH=/d;/^export LEFTHOOK_BIN=/d;/^export OPENASE_GO=/d;/^export OPENASE_GOFMT=/d;/^PATH=/d;/^LEFTHOOK_BIN=/d;/^OPENASE_GO=/d;/^OPENASE_GOFMT=/d' "$$hook" | sed '1d'; \
	} > "$$tmp"; \
	mv "$$tmp" "$$hook"; \
	chmod +x "$$hook"

hooks-run:
	$(GO) tool lefthook run pre-commit --all-files --no-auto-install

web-install:
	$(NPM) --prefix $(WEB_DIR) ci

web-check: web-install
	$(NPM) --prefix $(WEB_DIR) run check

web-build: web-install
	$(NPM) --prefix $(WEB_DIR) run build

build:
	$(GO) build $(OPENASE_MAIN)

build-web: web-build build

run:
	$(GO) run $(OPENASE_MAIN) serve

doctor:
	$(GO) run $(OPENASE_MAIN) doctor
