# OpenASE 技术文档 — 索引

本目录存放**技术契约、灰度指南、面向开发者的参考文档**。面向终端用户的产品文档(工单、工作流、机器等)请见 [`../guide/zh/`](../guide/zh/index.md)。

## 入门

| 文档 | 何时阅读 |
|------|---------|
| [源码构建与启动](./source-build-and-run.md) | 从源码构建、首次初始化、运行模式、托管用户服务、校验 |
| [配置参考](./configuration.md) | 环境变量、配置文件查找顺序、认证相关设置 |
| [CLI 参考](./cli-reference.md) | `openase` 资源命令、原始 API 逃生口、实时流、Agent 平台环境变量 |
| [开发指南](./development.md) | 仓库结构、构建/lint/测试/openapi 命令 |

## 身份与访问控制(IAM)

这三份文档是分层的,建议从上往下读:

| 文档 | 作用 |
|------|------|
| [IAM 双模式契约](./iam-dual-mode-contract.md) | `disabled` 与 `oidc` 两种鉴权模式的产品级契约,先读这个建立心智模型 |
| [IAM 管理边界](./iam-admin-boundaries.md) | 设置→安全 与 管理控制台 各自负责哪些 IAM 控件 |
| [OIDC 与 RBAC 配置](./human-auth-oidc-rbac.md) | 分步配置 OIDC 提供方(Auth0、Entra ID)、运维检查清单与 RBAC |

## 远程运行时与传输

| 文档 | 作用 |
|------|------|
| [WebSocket 运行时契约](./websocket-runtime-contract.md) | `ws_listener` 与 `ws_reverse` 共享的线协议、消息模型、版本与错误码 |
| [Remote Runtime v1 灰度](./remote-websocket-rollout.md) | 拓扑选型、迁移、守护进程安装、SSH 辅助诊断、运维指引 |

## Agent CLI 适配

| 文档 | 作用 |
|------|------|
| [Claude Code 流式协议](./claude-code-stream-protocol.md) | OpenASE 如何消费 Claude Code 流式输出 |
| [Gemini CLI 适配](./gemini-cli-adaptation-guide.md) | Gemini CLI 适配器集成说明 |
| [供应商推理等级矩阵](./provider-reasoning-effort-matrix.md) | 各 Provider 支持的推理强度参数 |

## 运维

| 文档 | 作用 |
|------|------|
| [可观测性清单](./observability-checklist.md) | 指标、日志、活动流、Webhook 摄入的卫生清单 |
| [桌面版 v1](./desktop-v1.md) | Electron 桌面外壳流程、打包、测试分层、桌面版 PostgreSQL v1 策略 |
