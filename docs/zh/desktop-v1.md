# OpenASE Desktop v1 指南

OpenASE Desktop v1 是围绕现有本地 OpenASE 运行时的桌面壳。它**不会**替换当前的 Go 后端、SvelteKit 前端或 PostgreSQL 数据模型。桌面宿主负责：

- 以动态端口在 `127.0.0.1` 上启动 `openase all-in-one`
- 在 UI 打开前轮询 `/healthz` 与 `/api/v1/healthz`
- 在启动失败、超时、异常退出时展示错误页，并提供日志/数据目录入口
- 从 Electron 菜单暴露日志、重启、本地数据目录能力
- 将 Electron 宿主和 OpenASE 二进制一起打包为 macOS / Linux 产物

## 范围与非目标

Desktop v1 有意保持当前产品形态不变：

- PostgreSQL 仍然是必需依赖。
- 现有 Web UI 仍然是主界面。
- `openase all-in-one` 仍然是本地服务入口。
- Desktop v1 不在本阶段引入 SQLite、自动更新、深链或生产可用的内置 PostgreSQL。

## 运行模型

### 生产态

1. 桌面宿主从打包资源中解析 OpenASE 二进制。
2. 宿主启动 `openase all-in-one --host 127.0.0.1 --port <dynamic-port> --config <resolved-config>`。
3. 宿主等待两个 readiness 端点都通过。
4. BrowserWindow 打开 `http://127.0.0.1:<dynamic-port>`。

### 开发态

1. 先启动 `web/` 的 Vite dev server。
2. Electron 仍然本地拉起 Go 服务。
3. BrowserWindow 指向 Vite dev server。
4. Vite 继续把 `/api/*` 和 SSE 转发到本机 Go 服务。

## 目录与路径约定

Desktop v1 会把桌面宿主自身状态和现有 OpenASE 数据分开。

| 用途 | 路径 |
| --- | --- |
| 桌面宿主源码 | `desktop/` |
| 桌面宿主用户数据目录 | Electron `app.getPath("userData")` 或 `OPENASE_DESKTOP_USER_DATA_DIR` |
| 桌面宿主日志目录 | Electron `app.getPath("logs")` 或 `OPENASE_DESKTOP_LOGS_DIR` |
| OpenASE 主目录 / 数据目录 | `~/.openase` 或 `OPENASE_DESKTOP_OPENASE_HOME` |
| OpenASE 配置路径 | `~/.openase/config.yaml` 或 `OPENASE_DESKTOP_OPENASE_CONFIG` |
| OpenASE stdout 日志 | `~/.openase/logs/desktop-service.log` |
| OpenASE stderr 日志 | `~/.openase/logs/desktop-service.stderr.log` |

兼容规则：Desktop v1 复用现有 `~/.openase` 结构，不会再复制一份 workspace、数据库配置或运行数据。

## PostgreSQL v1 策略

Desktop v1 支持和 CLI / 源码运行相同的 PostgreSQL 准备方式：

- 手工准备 PostgreSQL
- 通过 `openase setup` 准备 Docker PostgreSQL

你需要明确知道：

- 桌面宿主启动前必须先有有效配置文件
- 该配置文件必须指向可访问的 PostgreSQL
- 桌面宿主本身不会在启动时替你改写数据库配置

推荐首次准备方式：

```bash
make build-web
./bin/openase setup
```

`setup` 写出 `~/.openase/config.yaml` 后，桌面宿主即可直接复用。

### managed local PostgreSQL 后续方向

Desktop v1 只预留扩展路径，不在本 ticket 内交付生产可用的 managed local PostgreSQL。

后续建议路线：

1. 基于现有 `internal/testutil/pgtest` / `embedded-postgres` 资产抽出桌面侧 provider 边界
2. 为桌面宿主增加显式的 `managed-local-postgres` 模式，由宿主负责本地 PostgreSQL 生命周期
3. 保持现有 OpenASE 应用 schema 不变，只替换数据库 provisioning 路径
4. 保留 manual / docker 作为明确可选模式

该能力应通过独立 runtime/provider 合同落地，而不是把数据库托管逻辑侵入业务层。

## 命令

以下命令都在仓库根目录运行。

### 安装依赖

```bash
make desktop-install
```

### 安装桌面 Playwright 浏览器

```bash
make desktop-install-browsers
```

### 开发态

首次桌面开发运行前，先构建一次本地 OpenASE 二进制：

```bash
make build
make desktop-install
cd desktop
OPENASE_DESKTOP_OPENASE_BIN=../bin/openase pnpm run dev
```

该脚本会启动：

- `pnpm --dir ../web dev --host 127.0.0.1 --port 4174`
- 带 `OPENASE_DESKTOP_DEV_SERVER_URL=http://127.0.0.1:4174` 的 Electron 宿主

### 桌面单测与集成测试

```bash
make desktop-test
```

### 桌面完整验证

```bash
make desktop-validate
```

该命令会运行：

- 桌面 unit / integration tests
- 桌面 Electron E2E
- 桌面 package smoke

### 构建桌面 bundle

```bash
make desktop-build
```

该命令会把打包态使用的 OpenASE 二进制输出到 `desktop/.bundle/bin/openase`，并把配置模板复制到 `desktop/.bundle/config/`。

### 生成桌面安装包

```bash
make desktop-package
```

当前包类型：

- macOS：`dmg`、`zip`
- Linux：`AppImage`、`deb`

Windows 暂不纳入 v1 范围，因为当前验证、打包和支持矩阵优先覆盖 macOS / Linux。

### Package smoke

```bash
make desktop-package-smoke
```

该 smoke 会构建 unpacked app，并检查打包资源中是否包含：

- OpenASE 二进制
- 配置模板
- desktop bundle manifest

## 测试分层

Desktop v1 的验证不是只靠人工验收，而是保留分层测试。

### 现有仓库基线继续保留

- `make test-backend-coverage`
- `make web-validate`
- `scripts/dev/` 下已有服务黑盒脚本

### 新增桌面层验证

- unit：端口分配、命令拼装、健康检查超时、单实例、目录解析
- integration：服务生命周期、重启动作、窗口/控制器流程、异常退出处理
- E2E：启动 Electron，进入宿主承载页面，验证 API 联通，验证缺失配置时错误页可见
- package smoke：构建 unpacked app，并验证打包资源完整性

## 本地打包 smoke checklist

在 macOS / Linux 上做安装包验收时，建议执行以下 checklist：

1. `make desktop-package`
2. 安装或打开生成的桌面产物
3. 确认应用只允许单实例运行
4. 确认 readiness 期间出现启动中页面
5. 确认主 UI 从 `127.0.0.1:<dynamic-port>` 正常加载
6. 确认菜单可打开日志目录和数据目录
7. 确认 `Restart Local Service` 能干净替换子进程
8. 确认退出应用后没有遗留 OpenASE 孤儿进程
9. 确认二次启动能复用已有 `~/.openase` 配置和数据

## CI 集成

仓库 CI 已新增 `Desktop Checks` job。当 desktop、Go、web、构建脚本或 workflow 改动可能影响桌面行为时，会执行：

- `make desktop-install`
- `make desktop-install-browsers`
- `make desktop-validate`

## 故障排查

### 配置缺失

如果桌面宿主提示 `~/.openase/config.yaml` 不存在，先执行：

```bash
./bin/openase setup
```

### 数据库不可达

Desktop v1 不会隐藏 PostgreSQL 错误。若配置文件存在但服务仍启动失败，请从错误页打开日志目录，检查 DSN 指向的 PostgreSQL 是否可达。

### 开发态启动了错误二进制

开发态执行 `pnpm run dev` 前，设置 `OPENASE_DESKTOP_OPENASE_BIN=../bin/openase`，确保宿主启动的是你刚构建的 repo-local 二进制。
