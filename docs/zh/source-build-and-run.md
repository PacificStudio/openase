# OpenASE 源码构建与启动指南

本指南覆盖了从源码构建 OpenASE、运行首次设置以及在开发环境或单机部署上启动平台的完整流程。

关于远程 WebSocket 机器部署、守护进程安装和传输故障排查，请参见 [`docs/en/remote-websocket-rollout.md`](../en/remote-websocket-rollout.md)。

## 环境要求

- Go `1.26.1`，已添加到 `PATH`
- Node.js `22 LTS` 或 `24 LTS`，以及 `corepack pnpm`；如果你要运行 `make build-web` 或修改 `web/` 下的文件，这是必需的
- PostgreSQL（可从运行 OpenASE 的机器访问），或安装 Docker 让 setup 自动启动本地 PostgreSQL
- `git`
- 可选：`codex`、`claude` 或 `gemini` 已添加到 `PATH`，setup 会自动检测并注册 Agent Provider

避免使用 `23.x` 这类奇数版本的非 LTS Node。当前前端依赖集合包含 `engines` 约束，可能导致 `pnpm` 直接拒绝 `v23.11.1` 这样的版本。为了稳定构建，优先使用 `22.x` 线上的 `22.12+`，或受支持的 `24.x` 版本。

如果 `go` 不在 `PATH` 中，本项目通常使用以下路径之一：

```bash
export PATH=$PWD/.tooling/go/bin:$HOME/.local/go1.26.1/bin:$PATH
```

## 1. 克隆仓库

```bash
git clone https://github.com/PacificStudio/openase.git
cd openase
```

如果这台机器已经配置好 GitHub SSH key，也可以使用等价的 SSH 克隆方式：

```bash
git clone git@github.com:PacificStudio/openase.git
cd openase
```

## 2. 构建二进制文件

从仓库根目录一次性构建嵌入式前端和 Go 二进制文件：

```bash
make build-web
```

等价的显式命令为：

```bash
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run build
go build -o ./bin/openase ./cmd/openase
```

`make build-web` 只负责重新构建前端资源，然后编译 Go 二进制文件。它**不会**执行 `make openapi-generate`，也不会自动刷新 `api/openapi.json` 和 `web/src/lib/api/generated/openapi.d.ts`。

`make build` 仅基于 `internal/webui/static/` 中的当前内容编译 Go 二进制文件。在全新检出的仓库中，这意味着只有跟踪的占位文件，因此根 UI 会返回 503 构建提示，直到你重新生成 `web/`。

如果你想单独刷新嵌入式前端而不使用 `make build-web`，请运行：

```bash
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run build
go build -o ./bin/openase ./cmd/openase
```

如果你修改了后端 API 契约，或者希望在构建前刷新已提交的 OpenAPI 产物，请先单独运行：

```bash
make openapi-generate
```

前端构建和 Go 构建是一个发布单元。`vite build` 会刷新 `internal/webui/static/` 下的文件，但已构建或正在运行的 `openase` 二进制文件会继续提供旧的嵌入包，直到你重新构建二进制文件。如果浏览器堆栈跟踪提到 `internal/webui/static/_app/immutable/` 下不存在的 chunk 名称，首先假定是旧二进制文件或缓存的不可变资源，重新构建 `./cmd/openase`，然后强制刷新页面。

## API 契约生成

OpenASE 将后端导出的 OpenAPI 契约和前端生成的 TypeScript 契约纳入版本控制。

从仓库根目录重新生成两个产物：

```bash
make openapi-generate
```

显式命令为：

```bash
go run ./cmd/openase openapi generate --output api/openapi.json
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run api:generate
```

验证已提交的产物是否最新：

```bash
make openapi-check
```

CI 运行相同的 diff 检查，当 `api/openapi.json` 或 `web/src/lib/api/generated/openapi.d.ts` 过期时会使 PR 失败。

服务管理命令（如 `up`、`down`、`restart` 和 `logs`）请使用编译后的二进制文件。这些命令会安装或控制托管的用户服务，应指向稳定的可执行路径，而非临时的 `go run` 构建产物。

## 3. 准备 PostgreSQL

你可以跳过本节，让 `openase setup` 自动启动一个 Docker 容器运行的 PostgreSQL。如果你已有 PostgreSQL 并希望手动配置，最小可用 DSN 如下：

```yaml
database:
  dsn: postgres://openase:openase@localhost:5432/openase?sslmode=disable
```

setup 不提供第三种“用户态本地数据库”方案。如果当前用户无法访问 Docker，而机器上也没有现成的 PostgreSQL，那么你需要先自行准备 PostgreSQL，再在 setup 里选择手动连接路径。

如果你更喜欢手动管理配置而非使用 setup，从示例配置开始：

```bash
cp config.example.yaml ./config.yaml
```

大多数命令的配置查找顺序为：

1. `--config <path>`
2. `./config.yaml`、`./config.yml`、`./config.json`、`./config.toml`
3. `~/.openase/config.yaml`、`~/.openase/config.yml`、`~/.openase/config.json`、`~/.openase/config.toml`
4. `OPENASE_*` 环境变量和内置默认值

常用环境变量覆盖：

```bash
export OPENASE_DATABASE_DSN=postgres://openase:openase@localhost:5432/openase?sslmode=disable
export OPENASE_SERVER_PORT=19836
export OPENASE_ORCHESTRATOR_TICK_INTERVAL=2s
export OPENASE_LOG_FORMAT=json
```

### Docker PostgreSQL 示例（非默认端口）

本地开发时，一个简单的 Docker PostgreSQL 实例（端口 `15432`）如下：

```bash
docker run -d \
  --name openase-local-pg \
  --restart unless-stopped \
  -e POSTGRES_DB=openase_local \
  -e POSTGRES_USER=openase \
  -e POSTGRES_PASSWORD=change-me \
  -p 127.0.0.1:15432:5432 \
  -v openase_local_pgdata:/var/lib/postgresql/data \
  postgres:16-alpine
```

然后将 OpenASE 指向它：

```bash
export OPENASE_DATABASE_DSN='postgres://openase:change-me@127.0.0.1:15432/openase_local?sslmode=disable'
```

如果 `docker` 报 `permission denied while trying to connect to the docker API` 错误，当前登录会话可能尚未获取 `docker` 组权限。一个实用的解决方法是：

```bash
sg docker -c 'docker ps'
```

## 4. 运行首次设置

启动交互式终端设置：

```bash
./bin/openase setup
```

默认流程在终端内完成，不会打开浏览器。它将引导你完成：

- 选择数据库来源：
  - 自动启动一个本地 Docker PostgreSQL
  - 手动输入已有 PostgreSQL 的连接字段：`host`、`port`、数据库名、用户名、密码和 `sslmode`
- 验证所选数据库连接
- 检查本地 CLI 可用性和版本探测（`git`、`codex`、`claude` 及其他内置 Provider CLI）
- 选择浏览器认证模式：
  - `disabled`（禁用）
  - `oidc`，包含 issuer URL、client ID、client secret、redirect URL、scopes 和 bootstrap admin 的在线提示
- 选择 OpenASE 设置后的运行方式：
  - 仅配置
  - 在支持的机器上安装/更新当前用户的 `systemd --user` 服务
- 写入 `~/.openase/config.yaml`
- 写入 `~/.openase/.env`（含生成的平台认证令牌）
- 创建 `~/.openase/logs/` 和 `~/.openase/workspaces/`
- 初始化默认的本地组织、项目、工单状态和检测到的 Provider 种子数据

当你选择 Docker PostgreSQL 时，setup 使用可预测的默认值：

- 容器：`openase-local-postgres`
- 卷：`openase-local-postgres-data`
- 数据库：`openase`
- 用户：`openase`
- 主机端口：`127.0.0.1:15432`

Setup 自动生成 PostgreSQL 密码，验证容器连接，并在成功后打印复用/停止/删除命令。

如果当前用户没有 Docker 权限，setup 不会切换到别的本地数据库模式。这种情况下，请先准备好 PostgreSQL，再选择手动连接路径。

如果你在设置期间选择 OIDC 模式，流程会指向 [`docs/en/human-auth-oidc-rbac.md`](../en/human-auth-oidc-rbac.md)，适用于 Auth0 或 Azure Entra ID 等标准 OIDC 提供商。

## 5. 启动 OpenASE

### 单进程模式

这是本地开发的默认路径：

```bash
./bin/openase all-in-one --config ~/.openase/config.yaml
```

默认情况下，setup 生成的配置监听 `127.0.0.1:19836`，控制面板可在以下地址访问：

```text
http://127.0.0.1:19836
```

你可以在启动时覆盖绑定地址或调度间隔：

```bash
./bin/openase all-in-one --config ~/.openase/config.yaml --host 0.0.0.0 --port 40023 --tick-interval 2s
```

### 仅环境变量模式

如果你想将本地配置放在 `~/.openase/.env` 中而非配置文件，在启动 OpenASE 前将变量导出到当前 shell：

```bash
set -a
source ~/.openase/.env
set +a

./bin/openase all-in-one
```

重要：CLI 读取 `OPENASE_*` 环境变量，但不会自动 source `~/.openase/.env`。如果你跳过 `source` 步骤，进程将使用默认值启动或因缺少 `OPENASE_DATABASE_DSN` 等必需值而失败。

### 分进程模式

当你希望将 API 服务器和编排器作为独立进程运行时：

```bash
./bin/openase serve --config ~/.openase/config.yaml
./bin/openase orchestrate --config ~/.openase/config.yaml
```

## 6. 托管用户服务

Setup 在你选择服务运行模式时可以内联安装托管服务。你也可以之后手动应用或刷新：

```bash
./bin/openase up --config ~/.openase/config.yaml
./bin/openase logs --lines 100
./bin/openase restart
./bin/openase down
```

`up` 仅在找不到配置文件时运行 setup。否则它安装或更新一个执行以下命令的托管服务：

```text
openase all-in-one --config <resolved-config-path>
```

在支持的平台上，这使用仓库的用户服务抽象。服务读取 `~/.openase/.env` 并将日志写入 `~/.openase/logs/`。

长期运行时还需要注意：

- 托管的 `systemd --user` 单元只负责运行 OpenASE 本身，不负责托管 PostgreSQL。
- 如果你连接的是现成 PostgreSQL，需要你自己保证数据库长期运行。
- 如果 setup 创建了 Docker PostgreSQL 容器，它依然和 `openase.service` 是分离的服务边界。
- 如果希望 OpenASE 在用户退出登录后仍持续运行，通常还需要为该用户启用 linger：

```bash
loginctl enable-linger "$USER"
```

## 7. 验证安装

构建后或文档驱动的启动变更后，推荐的验证序列：

```bash
./bin/openase version
./bin/openase doctor --config ~/.openase/config.yaml
./bin/openase serve --help
./bin/openase orchestrate --help
./bin/openase all-in-one --help
```

`doctor` 检查配置加载、本地 CLI 可用性、PostgreSQL 可达性以及 setup 生成的 `~/.openase/` 目录结构。

对于运行中的实例，还可以直接验证 HTTP 健康端点：

```bash
curl -fsS http://127.0.0.1:19836/healthz
curl -fsS http://127.0.0.1:19836/api/v1/healthz
```

## 8. 常见运维说明

- `make build-web` 是刷新嵌入式 UI 后再编译 Go 二进制文件的安全源码构建路径，但它不会执行 `make openapi-generate`。
- 当后端 API 契约有变更，或者你需要刷新已提交的 OpenAPI / TypeScript 产物时，请单独运行 `make openapi-generate`。
- 如果你修改了 Svelte 应用，在编译前请重新构建 `web/`，否则二进制文件仍会嵌入旧的前端输出。
- `make build` 仅基于 `internal/webui/static/` 的当前内容编译 Go 二进制文件；在仅有跟踪占位文件的情况下，根 UI 会返回 503 引导响应，直到你重新构建 `web/`。
- 如果 Docker 设置失败，请检查 Docker 是否已安装、守护进程是否在运行、所选端口是否空闲、容器名是否未被占用。
- 如果 Docker 未安装，或当前用户无法访问 Docker daemon，setup 不会提供另一种本地数据库兜底方案；请改用你自己准备好的 PostgreSQL。
- `up` 应从你打算保留的已编译二进制文件路径运行，因为托管服务会保存安装时的可执行路径。
- `serve`、`orchestrate` 和 `all-in-one` 都接受 `--config`，`serve` / `all-in-one` 还接受 host 和 port 覆盖。
- 如果 `all-in-one` 报 `bind: address already in use` 错误，使用 `lsof -nP -iTCP:<port> -sTCP:LISTEN` 检查当前监听者。
- 从本地 `.env` 运行时，保持文件权限严格，例如 `chmod 600 ~/.openase/.env`。
- 本地日志文件通常保存在 `~/.openase/logs/` 下。
