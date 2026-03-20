WEB_DIR := web

.PHONY: web-build web-lint web-format-check web-check web-validate build run

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
