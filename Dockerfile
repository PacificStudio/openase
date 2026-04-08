FROM node:22-alpine AS web-builder
WORKDIR /src

COPY web/package.json web/pnpm-lock.yaml ./web/
RUN corepack enable && corepack pnpm --dir /src/web install --frozen-lockfile --reporter=append-only

COPY web ./web
RUN mkdir -p /src/internal/webui/static
RUN corepack pnpm --dir /src/web run build

FROM golang:1.26.1-alpine AS go-builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY ent ./ent
COPY internal ./internal
COPY deploy ./deploy
COPY --from=web-builder /src/internal/webui/static ./internal/webui/static
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/openase ./cmd/openase

ARG CODEX_VERSION=0.118.0
ARG CLAUDE_CODE_VERSION=2.1.96
ARG GEMINI_CLI_VERSION=0.36.0

FROM node:22-alpine AS runtime
ARG CODEX_VERSION
ARG CLAUDE_CODE_VERSION
ARG GEMINI_CLI_VERSION
RUN apk add --no-cache ca-certificates tzdata wget git bash ripgrep coreutils \
    && npm install -g \
        "@openai/codex@${CODEX_VERSION}" \
        "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" \
        "@google/gemini-cli@${GEMINI_CLI_VERSION}" \
    && npm cache clean --force \
    && addgroup -S openase \
    && adduser -S -D -h /var/lib/openase -s /sbin/nologin -G openase openase \
    && mkdir -p /var/lib/openase /app

COPY --from=go-builder /out/openase /usr/local/bin/openase
COPY --from=go-builder /src/deploy/coolify/entrypoint.sh /usr/local/bin/openase-entrypoint
RUN chmod 755 /usr/local/bin/openase /usr/local/bin/openase-entrypoint \
    && chown -R openase:openase /var/lib/openase /app

USER openase
WORKDIR /app
ENV HOME=/var/lib/openase
EXPOSE 40023
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=5 CMD wget -q -O /dev/null "http://127.0.0.1:${OPENASE_SERVER_PORT:-40023}/healthz" || exit 1
ENTRYPOINT ["/usr/local/bin/openase-entrypoint"]
