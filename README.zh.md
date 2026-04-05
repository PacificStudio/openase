<p align="center">
  <img src="docs/imgs/openase.svg" alt="OpenASE" width="80" height="80" />
</p>

<h1 align="center">OpenASE<br><sub>工单驱动的自动化软件工程</sub></h1>

<p align="center">
  <strong>OpenASE</strong> 是一个一站式平台，将工单转化为可运行的代码 — AI Agent 自动认领工单，在你的机器上执行工作流，并以完整的可追溯性交付成果。
</p>

<p align="center">
  <a href="#-从零开始运行"><img src="https://img.shields.io/badge/快速开始-10_min-blue?style=for-the-badge" alt="快速开始"></a>
  <a href="docs/guide/zh/index.md"><img src="https://img.shields.io/badge/指南-中文-ff69b4?style=for-the-badge" alt="中文指南"></a>
  <a href="docs/guide/en/index.md"><img src="https://img.shields.io/badge/Guide-English-ff69b4?style=for-the-badge" alt="English Guide"></a>
  <a href="#-架构"><img src="https://img.shields.io/badge/Architecture-Monolith-blueviolet?style=for-the-badge" alt="架构"></a>
  <a href="#-cli-参考"><img src="https://img.shields.io/badge/CLI-Reference-orange?style=for-the-badge" alt="CLI"></a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Linux-supported-success?logo=linux&logoColor=white" alt="Linux">
  <img src="https://img.shields.io/badge/macOS-supported-success?logo=apple&logoColor=white" alt="macOS">
  <img src="https://img.shields.io/badge/Windows-untested-lightgrey?logo=windows&logoColor=white" alt="Windows">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/SvelteKit-Frontend-FF3E00?logo=svelte&logoColor=white" alt="SvelteKit">
  <img src="https://img.shields.io/badge/Tailwind_CSS-Styling-06B6D4?logo=tailwindcss&logoColor=white" alt="Tailwind">
  <img src="https://img.shields.io/badge/PostgreSQL-Database-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Binary--first-go:embed-2D3748?logo=go&logoColor=white" alt="Embedded">
  <img src="https://img.shields.io/badge/SSE-Real--time-brightgreen" alt="SSE">
</p>

---

## 🖼️ 产品截图

<p align="center">
  内嵌的 Web UI 涵盖工单编排、AI 辅助规划、Skill 编写和实时项目追踪。
</p>

<p align="center">
  <img src="docs/imgs/active-project-board.png" alt="OpenASE 活跃项目看板" width="100%" />
</p>
<p align="center">
  <strong>实时执行</strong><br>
  监控真实项目工作，工单在 Backlog、Todo、In Progress、Review 之间流转。
</p>

<table align="center" width="100%">
<tr>
<td width="50%" align="center" style="vertical-align: top; padding: 12px;">
  <img src="docs/imgs/kanban-board.png" alt="OpenASE 看板" width="100%" />
  <p><strong>工单看板</strong><br>以看板视图管理待办和执行流程。</p>
</td>
<td width="50%" align="center" style="vertical-align: top; padding: 12px;">
  <img src="docs/imgs/kanban-board-with-ai-assistant.png" alt="OpenASE Project AI 助手" width="100%" />
  <p><strong>Project AI</strong><br>在看板旁边拆解工作为工单，检查工作区上下文。</p>
</td>
</tr>
<tr>
<td colspan="2" align="center" style="vertical-align: top; padding: 12px;">
  <img src="docs/imgs/skill-editor-install-codex.png" alt="OpenASE Skill 编辑器" width="100%" />
  <p><strong>Skill 编辑器</strong><br>编辑内置或自定义 Skill，驱动可复用的自动化工作流。</p>
</td>
</tr>
</table>

---

## 🧭 为什么需要 OpenASE

AI 编码 Agent 非常强大 — 但前提是人类保持控制。真正的问题在于**如何**实现这种控制。OpenASE 围绕两种互补的人机交互模式构建：**同步**和**异步**。

### 异步 AI — Ticket Agent

当需求明确、验收标准清晰、**Harness**（约束 Agent 行为的硬边界文档）已就绪时，**Ticket Agent** 全自动执行整个任务。它遵循 Workflow 的指示，更新工单状态，完成工单描述的工作 — 人类无需看护。

Workflow 定义了状态流转，并告诉 Agent 如何在每个阶段推进工单。它是一个**软控制面**：你在 Workflow 中编写自然语言指令来引导 Agent 在每个状态下的行为。以下是两种常见模式：

- **全栈编码** — 单个 Agent 处理整个生命周期。设置简单的状态流 `Todo → In Progress → In Review → Merging → Done`。Agent 认领工单、编写代码、创建 PR 并推进状态。`In Review` 阶段的人工审查是可选的 — 你可以保留它作为质量关卡，也可以让另一个 Agent 来审查。

- **混合接力** — 多个专业 Agent 以接力方式协作。设置更丰富的流程 `Design → Backend → Frontend → Testing → In Review → Merging → Done`，为每个阶段分配不同的 Agent 角色（产品经理、后端工程师、前端工程师、测试工程师）。Agent 随着工单推进将工作交接给下一个角色。人工审查检查点在任何阶段都是可选的 — 你可以在需要监督的地方插入，或完全移除以实现全自动。

> 异步和同步 AI 均支持多种 Agent CLI。推荐使用 **Claude Code** 和 **Codex**；**Gemini CLI** 已支持但稳定性较差。

### 同步 AI — Project AI

当你的需求还不够明确、问题空间需要探索、或者还没准备好提交正式工单时 — 与 **Project AI** 对话。

Project AI 是一个同步交互式助手，位于控制面板的侧边栏。用它来分析需求、探索技术方案、撰写 PRD 和文档、初始化仓库，在启动异步执行之前完成所有准备工作。每个对话标签页运行在**独立的工作区** — 标签页之间互相隔离，可以并行运行，方便管理和并排查看。

**Project AI 的能力：**

- 读取工单详情、依赖关系、活动历史和运行状态
- 读取 Workflow/Harness 配置、Skill 代码和机器健康状态
- 检查 git 工作区差异（分支、文件变更、仓库状态）
- 创建和更新工单，发布项目动态
- 直接修改 Workflow/Harness 和 Skill
- 触发工单的 Agent 执行
- 操作 git（提交、分支、推送）
- 控制 Agent 运行时（暂停、恢复、中断）

**多 Agent、隔离工作区 — 统一面板：** 每个 Project AI 标签页生成独立的隔离工作区。你可以在同一面板中并排运行多个 Agent — 每个处理不同的任务、在不同的分支上工作，互不干扰。不再需要在终端之间切换或在不同 IDE 之间跳转。一个统一视图即可启动、监控和交互所有 Agent。

**上下文感知聚焦：** 当你在控制面板中导航时，Project AI 自动在 4 种聚焦模式之间切换 — **Workflow**、**Skill**、**Ticket** 和 **Machine** — 为对话注入相关上下文，始终理解你正在查看的内容。

**最佳实践：** 带着模糊的想法开始，与 Project AI 讨论收敛到最佳方案，让它起草文档并初始化仓库。然后搭建自动化的全栈或混合接力 Workflow，配置工单依赖关系控制并行和阻塞，让系统运行起来。通过精心构建的 Harness 和依赖图，平台可以长时间保持高吞吐量的自主工作，完全无需人工介入。

### Skill — 扩展 Agent 的能力

Skill 是可复用的指令文档，赋予 Agent 超越原始编码的额外能力。每个 Workflow 自动绑定一个**内置 Ticket Skill**，教会 Ticket Agent 如何在平台上更新工单状态 — 这就是 Agent 驱动状态流转而无需硬编码逻辑的方式。

除了内置 Skill，你还可以：

- **绑定更多内置 Skill** 到任何 Workflow，用于常见操作（如 git 规范、PR 创建、代码审查清单）。
- **创建自定义 Skill**，通过 Skill 编辑器编码项目特定的知识和流程。
- **从仓库导入 Skill** — 让 Project AI 从你的仓库拉取 Skill 文件并注册到 OpenASE。

当 Workflow 运行时，其绑定的 Skill 会在运行时**注入到 CLI Agent 的 Skill 目录**（如 `.codex/skills/`、`.claude/skills/`、`.gemini/skills/`），Agent 会原生加载它们。Project AI 可以访问**所有 Skill**，不受 Workflow 绑定限制，是编写、编辑和调试 Skill 的理想场所。

**最佳实践：** 使用 Project AI 交互式地创建、修改或调试你的 Skill。当 Skill 工作正常后，保存到 OpenASE 并绑定到相关 Workflow — 在这些 Workflow 下运行的每个 Ticket Agent 都会自动继承该 Skill。

### 交织

在实践中，这两种模式不断交织：

- **需求变更** → 已有的 Harness 过期，需要同步打磨。
- **项目迭代** → 必须编写新的 Harness，防止 Agent 偏离方向。
- **技术债积累** → 定时任务触发 Agent 自动清理。

同步和异步工作的交织需要一个**统一的家** — 而不是分散在终端、IDE 和聊天窗口中。OpenASE 将一切整合在一起：**多 Agent CLI 支持**（Claude Code、Codex、Gemini CLI）、**多机器调度**，以及单一控制面板，让你无需在工具之间切换。

### 组织与项目管理

OpenASE 开箱即用支持**多组织（Org）**管理。每个 Org 包含自己的项目集，每个项目拥有独立的工单、工作流、Skill、机器和 Agent 配置。跨 Org 的**团队协作**（共享项目、跨 Org 角色权限）目前正在开发中（**WIP**）。

### ⚠️ 安全提醒

为了最大化无人值守 Workflow 的执行效率，OpenASE 默认以**宽松权限标志**启动 CLI Agent（例如 Claude Code 的 `--dangerously-skip-permissions`、Codex 的 `--full-auto`）。这意味着 Agent 可以在宿主机上读写和执行任意命令，无需逐次确认。

**请注意风险：**
- 仅在你信任 Agent 访问范围的机器上运行 OpenASE。
- 根据需要限制操作系统级权限（用户账户、文件系统边界）。
- **本项目不是为公网部署设计的。** 它面向本地开发、私有网络和受信任的环境。
- 默认情况下，OpenASE 以 **Demo 模式**（认证关闭）运行 — 无需登录，网络上的任何人都可以访问控制面板。这对于快速本地体验很方便，但**在局域网或多用户部署场景下，请务必配置 HTTPS + OIDC 认证**以保护你的数据和 Agent 访问权限。参见 [OIDC & RBAC 指南](docs/guide/zh/settings.md)了解配置方法。

### 愿景

OpenASE 的目标是成为一个**全生命周期软件工程平台**：从工单到部署代码的端到端迭代、跨角色的团队协作、多仓库编排 — 一切由上述工单优先、Agent 原生的模型驱动。

---

## ✨ 核心特性

<table align="center" width="100%">
<tr>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>📋 工单驱动编排</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Core-06B6D4?style=for-the-badge&logo=target&logoColor=white" alt="Core" />
</div>

<p align="center"><strong>看板 & 列表视图</strong></p>
<p align="center"><strong>父子 & 依赖追踪</strong></p>
<p align="center"><strong>自定义状态 & 优先级</strong></p>
<p align="center"><strong>仓库范围绑定</strong></p>

</td>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>🤖 多 Agent 支持</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Agents-10B981?style=for-the-badge&logo=robot&logoColor=white" alt="Agents" />
</div>

<p align="center"><strong>Claude Code / Codex / Gemini CLI</strong></p>
<p align="center"><strong>实时流式输出（SSE）</strong></p>
<p align="center"><strong>Agent 生命周期管理</strong></p>
<p align="center"><strong>并发执行控制</strong></p>

</td>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>⚡ 工作流引擎</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Automation-8B5CF6?style=for-the-badge&logo=lightning&logoColor=white" alt="Automation" />
</div>

<p align="center"><strong>Markdown Harness 文档</strong></p>
<p align="center"><strong>Skill 绑定 & 生命周期钩子</strong></p>
<p align="center"><strong>定时 Cron 任务</strong></p>
<p align="center"><strong>内置角色模板</strong></p>

</td>
</tr>
<tr>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>🖥️ 机器管理</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Infra-F59E0B?style=for-the-badge&logo=server&logoColor=white" alt="Infra" />
</div>

<p align="center"><strong>本地 / 直连 / 反向连接机器</strong></p>
<p align="center"><strong>WebSocket 执行 + SSH 辅助兼容</strong></p>
<p align="center"><strong>健康监控 & 探针</strong></p>
<p align="center"><strong>CPU / 内存 / 磁盘指标</strong></p>
<p align="center"><strong>连接诊断</strong></p>

</td>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>🔐 认证 & 安全</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Security-EC4899?style=for-the-badge&logo=shield&logoColor=white" alt="Security" />
</div>

<p align="center"><strong>OIDC 浏览器登录（Auth0、Entra ID）</strong></p>
<p align="center"><strong>Agent 平台 Token 认证</strong></p>
<p align="center"><strong>Org & 项目 RBAC</strong></p>
<p align="center"><strong>GitHub 凭证管理</strong></p>

</td>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>📡 可观测性</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Realtime-06B6D4?style=for-the-badge&logo=eye&logoColor=white" alt="可观测性" />
</div>

<p align="center"><strong>实时活动事件流</strong></p>
<p align="center"><strong>Agent 运行步骤追踪</strong></p>
<p align="center"><strong>GitHub Webhook 接入</strong></p>
<p align="center"><strong>项目动态线程</strong></p>

</td>
</tr>
</table>

---

## 🤔 什么是 OpenASE？

OpenASE 是一个**单 Go 二进制文件**，将 API 服务器、工作流编排器和内嵌的 Web UI 打包在一起。它遵循**工单驱动**模型：每项工作都是一个工单，每个工单都有工作流，AI Agent 根据状态触发器自动认领并执行工单。

```
你创建一个工单  →  编排器检测到认领状态
    →  Agent 认领工单  →  在机器上执行工作流
    →  活动流记录每一步  →  工单完成
```

**运行时无需 Node.js** — SvelteKit 前端通过 `go:embed` 编译嵌入到 Go 二进制文件中。

---

## 📊 状态 & 路线图

### 功能完成度

| 模块 | 状态 | 说明 |
|------|------|------|
| **工单** | ✅ 稳定 | CRUD、看板/列表视图、评论、依赖、父子关系、归档 |
| **Agent** | ✅ 稳定 | 注册、运行监控、流式输出、生命周期管理 |
| **工作流** | ✅ 稳定 | Harness 编辑、状态/Skill 绑定、钩子、版本历史、影响分析 |
| **Skill** | ✅ 稳定 | 内置 & 自定义 Skill、工作流绑定、启用/禁用 |
| **活动** | ✅ 稳定 | 实时 SSE 事件流、过滤、搜索 |
| **动态** | ✅ 稳定 | 线程、评论、修订历史 |
| **设置** | ✅ 稳定 | 状态管理、仓库、通知、安全、归档工单 |
| **定时任务** | ✅ 稳定 | 基于 Cron 的工单创建、手动触发、启用/禁用 |
| **机器（本地）** | ✅ 稳定 | 本地机器注册、健康探针、资源指标 |
| **CLI** | ✅ 稳定 | 双层合约、资源命令、原始 API、实时流 |
| **安装向导** | ✅ 稳定 | 交互式终端安装、Docker PostgreSQL、托管用户服务（Linux `systemd --user`、macOS `launchd`） |
| **机器（远程）** | 🚧 WIP | 远程运行时正在收敛到 WebSocket 执行，支持直连和反向连接；SSH 保留为辅助和兼容路径 |
| **OIDC 认证** | 🚧 WIP | 浏览器登录、会话管理、RBAC |

### 路线图

| 优先级 | 项目 | 描述 |
|--------|------|------|
| 🔴 高 | **远程机器执行** | 完成直连和反向连接机器的 WebSocket 执行，然后移除运行时路径中的旧版 SSH 兼容 |
| 🟡 中 | **Windows 支持** | WSL2 之外的原生服务管理和 Shell 脚本支持 |
| 🟡 中 | **通知渠道** | Slack、邮件和 Webhook 通知投递 |
| 🟡 中 | **iOS & Android App** | 移动端控制面板，随时随地监控和管理项目 |
| 🟡 中 | **桌面一体化应用** | 独立桌面应用，打包完整的 OpenASE 体验 |
| 🟡 中 | **Kubernetes 运行时** | 在 Kubernetes 集群上运行 Agent 工作负载，实现弹性伸缩 |
| 🟢 远期 | **多组织协作** | 跨组织项目共享和权限 |
| 🟢 远期 | **插件生态** | 第三方插件支持，自定义工具和集成 |
| 🟢 远期 | **指标仪表板** | Agent 性能指标、工单吞吐量分析 |

---

## 🚀 从零开始运行

本节介绍在**全新机器**上所需的一切 — 从安装系统依赖到打开 Web UI。

### 平台支持

| 平台 | 状态 | 说明 |
|------|------|------|
| **Linux** (x86_64, arm64) | ✅ 完全支持 | 主要开发和部署平台 |
| **macOS** (Apple Silicon, Intel) | ✅ 支持 | `setup`、`up/down/restart/logs` 和托管用户服务使用 `launchd`，配置文件位于 `~/Library/LaunchAgents/com.openase.plist` |
| **Windows** | ⚠️ 未测试 | 原生服务管理和 Shell 脚本尚未验证。建议使用 WSL2 作为替代方案 |

### 第 0 步：系统前置条件

<details>
<summary><strong>安装 Go 1.26+</strong></summary>

```bash
# 下载（根据需要调整版本和系统架构）
wget https://go.dev/dl/go1.26.1.linux-amd64.tar.gz

# 解压到 /usr/local（需要 sudo）
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.1.linux-amd64.tar.gz

# 添加到 PATH — 追加到 ~/.bashrc 或 ~/.zshrc
export PATH=$PATH:/usr/local/go/bin

# 验证
go version   # go1.26.1 linux/amd64
```

替代方案：如果使用项目本地工具链：

```bash
export PATH=$PWD/.tooling/go/bin:$HOME/.local/go1.26.1/bin:$PATH
```

</details>

<details>
<summary><strong>安装 Node.js 22 LTS 或 24 LTS & pnpm（仅构建时需要）</strong></summary>

Node.js 仅在构建前端时需要。**运行时不需要**。

使用 LTS 版本：推荐 **Node 22 LTS**，**Node 24 LTS** 也可以。避免使用 **Node 23** 等非 LTS 奇数版本。

```bash
# 方式 A：通过 nvm（推荐）
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
source ~/.bashrc
nvm install 22
nvm use 22

# 方式 B：通过包管理器（Ubuntu/Debian）
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt-get install -y nodejs

# 启用 corepack 以使用 pnpm
corepack enable

# 验证
node --version   # v22.x.x
pnpm --version   # 10.x.x（通过 corepack）
```

</details>

<details>
<summary><strong>安装 PostgreSQL</strong></summary>

你有两个选择 — 让 OpenASE setup 自动启动 Docker 版 PostgreSQL，或者自行安装。

如果你的用户没有 Docker 权限，必须在运行 `openase setup` 之前自行准备 PostgreSQL。

**方式 A：Docker（推荐用于本地开发）**

```bash
# 安装 Docker（如果没有）
sudo apt-get update && sudo apt-get install -y docker.io
sudo usermod -aG docker $USER
newgrp docker   # 或重新登录

# OpenASE setup 会自动为你创建容器
```

**方式 B：自行安装 PostgreSQL**

macOS 本地开发推荐使用 Homebrew 或 Postgres.app 管理的原生 PostgreSQL：

```bash
# macOS 通过 Homebrew
brew install postgresql@16
brew services start postgresql@16

# 验证
psql postgres://localhost:5432/postgres?sslmode=disable -c "SELECT 1;"
```

Linux 上使用发行版包安装：

```bash
# Ubuntu/Debian
sudo apt-get install -y postgresql postgresql-client

# 创建数据库和用户
sudo -u postgres psql -c "CREATE USER openase WITH PASSWORD 'openase';"
sudo -u postgres psql -c "CREATE DATABASE openase OWNER openase;"

# 验证
psql postgres://openase:openase@localhost:5432/openase?sslmode=disable -c "SELECT 1;"
```

</details>

<details>
<summary><strong>安装 Git 及其他工具</strong></summary>

```bash
# Ubuntu/Debian
sudo apt-get install -y git make curl wget

# 验证
git --version
make --version
```

</details>

<details>
<summary><strong>（可选）安装 AI Agent CLI</strong></summary>

OpenASE setup 会自动检测 `PATH` 上的这些工具：

| Agent | 安装方式 |
|-------|---------|
| **Claude Code** | `npm install -g @anthropic-ai/claude-code` |
| **Codex** | `npm install -g @openai/codex` |
| **Gemini CLI** | `npm install -g @anthropic-ai/gemini-cli` |

这些也可以稍后安装 — setup 会自动发现已安装的 Provider。

</details>

### 第 1 步：克隆 & 构建

```bash
git clone https://github.com/PacificStudio/openase.git
cd openase

# 一条命令构建前端 + Go 二进制
make build-web
```

底层执行的是：

```bash
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run build
go build -o ./bin/openase ./cmd/openase
```

`make build-web` 重建内嵌前端然后编译 Go 二进制。它**不会**刷新已提交的 OpenAPI 产物。如果你修改了后端 API 结构或想刷新 `api/openapi.json` 和 `web/src/lib/api/generated/openapi.d.ts`，请先单独运行 `make openapi-generate`。

验证构建：

```bash
./bin/openase version
```

### 第 2 步：运行首次安装

```bash
./bin/openase setup
```

交互式终端安装会引导你完成：

1. **数据库** — 自动启动 Docker PostgreSQL，或输入已有的 PostgreSQL 连接（`host`、`port`、`database`、`user`、`password`、`sslmode`）
2. **CLI 检测** — 检查 PATH 上的 `git`、`claude`、`codex`、`gemini`
3. **认证模式** — `disabled`（本地开发）或 `oidc`（浏览器登录）
4. **服务模式** — 仅配置，或安装托管用户服务（Linux `systemd --user`、macOS `launchd`）
5. **种子数据** — 创建 Org、项目、工单状态和检测到的 Provider

安装会在 `~/.openase/` 下创建以下内容：

```
~/.openase/
├── config.yaml       # 运行时配置
├── .env              # 平台认证 Token
├── logs/             # 服务日志
└── workspaces/       # Agent 工作区
```

> **Docker PostgreSQL 说明：** 选择 Docker 时，setup 使用可预测的默认值 — 容器 `openase-local-postgres`，端口 `127.0.0.1:15432`，数据库 `openase`。密码自动生成。如果你的账户没有 Docker 权限，setup 不会提供其他本地数据库回退方案；请先准备好 PostgreSQL，然后选择手动连接路径。

### 第 3 步：启动

```bash
# 一体化模式：API 服务器 + 编排器在单个进程中
./bin/openase all-in-one --config ~/.openase/config.yaml
```

控制面板现在可以访问：

```
http://127.0.0.1:19836
```

> **提示：** 运行 `./bin/openase doctor --config ~/.openase/config.yaml` 诊断任何问题。

### 第 4 步：验证

```bash
# 健康检查
curl -fsS http://127.0.0.1:19836/healthz
curl -fsS http://127.0.0.1:19836/api/v1/healthz

# 或使用内置诊断
./bin/openase doctor --config ~/.openase/config.yaml
```

在浏览器中打开 `http://127.0.0.1:19836` — 你应该能看到 OpenASE 控制面板。

### 接下来？

平台运行后，请参照用户指南 — 快速开始（[EN](docs/guide/en/startup.md) | [中文](docs/guide/zh/startup.md)）进行：

1. 配置工单状态并关联仓库
2. 注册机器和 AI Agent
3. 创建你的第一个工作流和工单
4. 观察 Agent 自动执行

---

## 🔧 其他运行模式

### 托管用户服务

安装向导可以自动安装托管用户服务，你也可以手动管理：

```bash
./bin/openase up      --config ~/.openase/config.yaml   # 安装 & 启动
./bin/openase logs    --lines 100                        # 查看日志
./bin/openase restart                                    # 重启
./bin/openase down                                       # 停止
```

托管服务仅运行 OpenASE 本身（`openase all-in-one --config ...`），不管理 PostgreSQL。如果你指向了已有的 PostgreSQL 实例，请保持该数据库单独运行。如果 setup 创建了 Docker PostgreSQL 容器，该容器仍然是独立于 OpenASE 的服务边界。

| 平台 | 管理器 | 服务定义 | 检查 | 重启 | 停止 | 日志 |
|------|--------|---------|------|------|------|------|
| Linux | `systemd --user` | `~/.config/systemd/user/openase.service` | `systemctl --user status openase` | `systemctl --user restart openase` | `systemctl --user stop openase` | `journalctl --user -u openase -n 200 -f` |
| macOS | `launchd` | `~/Library/LaunchAgents/com.openase.plist` | `launchctl print gui/$(id -u)/com.openase` | `launchctl kickstart -k <target>` | `launchctl bootout <target>` | `tail -n 200 -f ~/.openase/logs/openase.stdout.log ~/.openase/logs/openase.stderr.log` |

对于 Linux 长期运行的服务器，`systemd --user` 可能还需要启用 lingering，以便用户服务在注销后仍然运行：

```bash
loginctl enable-linger "$USER"
```

### 分离进程模式

将 API 服务器和编排器作为独立进程运行：

```bash
./bin/openase serve       --config ~/.openase/config.yaml
./bin/openase orchestrate --config ~/.openase/config.yaml
```

### 纯环境变量模式

如果你更喜欢用环境变量而非配置文件：

```bash
export OPENASE_DATABASE_DSN=postgres://openase:openase@localhost:5432/openase?sslmode=disable
export OPENASE_SERVER_PORT=19836
export OPENASE_ORCHESTRATOR_TICK_INTERVAL=2s

./bin/openase all-in-one
```

或从 `~/.openase/.env` 加载：

```bash
set -a && source ~/.openase/.env && set +a
./bin/openase all-in-one
```

---

## ⚙️ 配置

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `OPENASE_SERVER_PORT` | `19836` | HTTP 服务器端口 |
| `OPENASE_DATABASE_DSN` | — | PostgreSQL 连接字符串（**必需**） |
| `OPENASE_ORCHESTRATOR_TICK_INTERVAL` | `5s` | 编排器轮询间隔 |
| `OPENASE_LOG_FORMAT` | `text` | 日志格式（`text` 或 `json`） |
| `OPENASE_LOG_LEVEL` | `info` | 日志级别 |

### 配置文件查找顺序

1. `--config <path>` 命令行参数
2. `./config.yaml`（或 `.yml`、`.json`、`.toml`）
3. `~/.openase/config.yaml`
4. `OPENASE_*` 环境变量 + 内置默认值

### 认证

| 模式 | 说明 | 使用场景 |
|------|------|---------|
| `disabled` | 无需认证 | 本地开发 |
| `oidc` | 通过 OIDC 提供商浏览器登录 | 生产环境、团队使用 |

OIDC 支持标准提供商：Auth0、Azure Entra ID 以及任何 OpenID Connect 兼容的 IdP。参见 OIDC & RBAC 指南（[EN](docs/en/human-auth-oidc-rbac.md) | [中文](docs/zh/human-auth-oidc-rbac.md)）了解配置方法。

---

## 🏗️ 架构

### 产品形态

| 原则 | 说明 |
|------|------|
| **All-Go 单体** | API 服务器、编排器、安装流程和内嵌 UI 在一个二进制文件中 |
| **二进制优先** | Web UI 通过 `go:embed` 嵌入 — 运行时无需 Node.js |
| **工单驱动** | 工单、工作流、状态和活动是核心运作模型 |
| **多 Agent** | 基于适配器支持 Claude Code、Codex 和 Gemini CLI |
| **Git 感知** | Workflow Harness 和 Skill 在运行时感知仓库上下文 |

### 仓库结构

```
openase/
├── cmd/openase/              # CLI 入口
├── internal/
│   ├── app/                  # 应用组装（serve / orchestrate / all-in-one）
│   ├── httpapi/              # HTTP API、SSE、Webhook、内嵌 UI
│   ├── orchestrator/         # 调度、健康检查、重试
│   ├── workflow/             # 工作流服务、Harness、钩子、Skill
│   ├── agentplatform/        # Agent Token 认证
│   ├── setup/                # 首次安装
│   ├── builtin/              # 内置角色 & Skill 模板
│   └── webui/static/         # 内嵌前端输出
├── web/                      # SvelteKit 控制面板源码
├── docs/
│   └── guide/                # 用户指南（按模块组织）
├── config.example.yaml
├── Makefile
└── go.mod
```

### 系统流程

```mermaid
flowchart LR
    U[用户 / 定时任务] --> T[工单]
    T --> O[编排器]
    O -->|状态触发| A[Agent]
    A --> W[工作流 + Harness]
    W --> S[Skill]
    A -->|在机器上执行| M[机器]
    M --> R[结果]
    R --> E[活动流]
    E --> UI[Web 控制面板]
```

---

## 🖥️ 控制面板

内嵌的 Web UI 提供完整的项目管理体验：

| 模块 | 功能 |
|------|------|
| **[工单](docs/guide/zh/tickets.md)** | 看板、列表视图、过滤、评论、依赖、仓库范围 |
| **[Agent](docs/guide/zh/agents.md)** | 注册、实时运行监控、暂停/恢复/退役生命周期 |
| **[机器](docs/guide/zh/machines.md)** | SSH/本地/云注册、健康探针、资源指标 |
| **[工作流](docs/guide/zh/workflows.md)** | Harness 编辑、状态绑定、Skill 绑定、版本历史、影响分析 |
| **[Skill](docs/guide/zh/skills.md)** | 内置 & 自定义 Skill 管理、工作流绑定 |
| **[定时任务](docs/guide/zh/scheduled-jobs.md)** | 基于 Cron 的工单创建、手动触发、启用/禁用 |
| **[活动](docs/guide/zh/activity.md)** | 实时事件流、类型过滤、关键词搜索 |
| **[动态](docs/guide/zh/updates.md)** | 团队进展线程、评论、修订历史 |
| **[设置](docs/guide/zh/settings.md)** | 状态管理、仓库、通知、安全、归档工单 |

---

## 💻 CLI 参考

OpenASE 遵循 **GitHub 风格的双层 CLI 合约**：

### 资源命令

```bash
openase ticket list       --status-name Todo --json tickets
openase ticket create     --title "修复登录 bug" --description "..."
openase ticket update     --status_name "In Review"
openase ticket comment    create --body "发现阻塞依赖"
openase ticket detail     $PROJECT_ID $TICKET_ID

openase workflow create   $PROJECT_ID --name "Codex Worker"
openase scheduled-job trigger $JOB_ID
openase project update    --description "最新上下文"
```

### 原始 API 逃生口

```bash
openase api GET  /api/v1/projects/$PID/tickets --query status_name=Todo
openase api PATCH /api/v1/tickets/$TID --field status_id=$SID
```

### 实时流

```bash
openase watch tickets $PROJECT_ID
```

### 输出格式化

```bash
--jq '<expr>'              # JQ 过滤器
--json field1,field2       # 选择字段
--template '{{...}}'       # Go 模板
```

`--kebab-case` 和 `--snake_case` 两种风格的参数名均可使用。

---

## 🔌 Agent 平台

Agent Worker 从工作区包装器继承环境变量：

| 变量 | 用途 |
|------|------|
| `OPENASE_API_URL` | 平台 API 端点 |
| `OPENASE_AGENT_TOKEN` | Agent 认证 Token |
| `OPENASE_PROJECT_ID` | 当前项目上下文 |
| `OPENASE_TICKET_ID` | 当前工单上下文 |

---

## 🛠️ 开发

### 构建命令

```bash
make hooks-install        # 设置 git hooks（lefthook）
make check                # 运行格式化 + 后端覆盖率检查
make build-web            # 构建前端资产 + Go 二进制（不刷新 OpenAPI 产物）
make build                # 仅构建 Go 二进制（使用已有前端）
make run                  # 以开发模式运行 API 服务器
make doctor               # 运行本地环境诊断
```

### 前端质量门禁

```bash
make web-format-check     # Prettier 格式化
make web-lint             # ESLint 检查
make web-check            # Svelte 类型检查
make web-validate         # 以上全部
```

### OpenAPI 合约

```bash
make openapi-generate     # 重新生成 api/openapi.json + TS 类型
make openapi-check        # 验证已提交的产物是否最新
```

### 测试

```bash
make test                        # Go 测试套件
make test-backend-coverage       # 完整后端测试 + 覆盖率门禁
make lint                        # 对 origin/main 以来的变更进行 Lint
make lint-all                    # 完整 Lint 套件
```

---

## 📖 文档

| 文档 | EN | 中文 |
|------|----|----|
| **用户指南** | [English](docs/guide/en/index.md) | [中文](docs/guide/zh/index.md) |
| 快速开始 | [English](docs/guide/en/startup.md) | [中文](docs/guide/zh/startup.md) |
| 模块架构 | [English](docs/guide/en/architecture.md) | [中文](docs/guide/zh/architecture.md) |
| FAQ | [English](docs/guide/en/faq.md) | [中文](docs/guide/zh/faq.md) |
| **源码构建与运行** | [English](docs/en/source-build-and-run.md) | [中文](docs/zh/source-build-and-run.md) |
| OIDC & RBAC | [English](docs/en/human-auth-oidc-rbac.md) | [中文](docs/zh/human-auth-oidc-rbac.md) |
| 可观测性 | [English](docs/en/observability-checklist.md) | [中文](docs/zh/observability-checklist.md) |
| WebSocket 部署 | [English](docs/en/remote-websocket-rollout.md) | [中文](docs/zh/remote-websocket-rollout.md) |
| Gemini CLI 适配 | [English](docs/en/gemini-cli-adaptation-guide.md) | [中文](docs/zh/gemini-cli-adaptation-guide.md) |
| Claude Code 流协议 | [English](docs/en/claude-code-stream-protocol.md) | [中文](docs/zh/claude-code-stream-protocol.md) |

---

## 📄 许可证

参见 [LICENSE](LICENSE)。

---

<p align="center">
  <strong>OpenASE</strong>
  <br>
  <em>创建工单，Agent 完成剩下的一切。</em>
</p>
