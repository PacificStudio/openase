WEB_DIR := web

.PHONY: web-build build run

web-build:
	npm --prefix $(WEB_DIR) install
	npm --prefix $(WEB_DIR) run build

build: web-build
	go build ./cmd/openase

run: web-build
	go run ./cmd/openase serve
