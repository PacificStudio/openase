WEB_DIR := web
LINT_SCRIPT := ./scripts/ci/lint.sh

.PHONY: web-build build run lint lint-all lint-depguard

web-build:
	npm --prefix $(WEB_DIR) install
	npm --prefix $(WEB_DIR) run build

build: web-build
	go build ./cmd/openase

run: web-build
	go run ./cmd/openase serve

lint:
	OPENASE_LINT_NEW_FROM_REV=$$(git merge-base origin/main HEAD) $(LINT_SCRIPT)

lint-all:
	$(LINT_SCRIPT)

lint-depguard:
	$(LINT_SCRIPT) --enable-only=depguard ./...
