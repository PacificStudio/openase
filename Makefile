WEB_DIR := web
LINT_SCRIPT := ./scripts/ci/lint.sh

.PHONY: web-build web-lint web-format-check web-check web-validate build run lint lint-all lint-depguard

web-build:
	npm --prefix $(WEB_DIR) install
	npm --prefix $(WEB_DIR) run build

web-lint:
	npm --prefix $(WEB_DIR) install
	npm --prefix $(WEB_DIR) run lint

web-format-check:
	npm --prefix $(WEB_DIR) install
	npm --prefix $(WEB_DIR) run format:check

web-check:
	npm --prefix $(WEB_DIR) install
	npm --prefix $(WEB_DIR) run check:svelte

web-validate: web-format-check web-lint web-check

build: web-build
	go build ./cmd/openase

run: web-build
	go run ./cmd/openase serve

lint:
	OPENASE_LINT_NEW_FROM_REV=$${LINT_BASE_REV:-$$(git merge-base origin/main HEAD)} $(LINT_SCRIPT)

lint-all:
	$(LINT_SCRIPT)

lint-depguard:
	$(LINT_SCRIPT) --enable-only=depguard ./...
