# OpenASE 配置参考

本页说明 OpenASE 的环境变量、配置文件查找顺序与认证相关设置。源码构建与启动步骤参见 [`source-build-and-run.md`](./source-build-and-run.md)。

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `OPENASE_SERVER_PORT` | `19836` | HTTP 服务器端口 |
| `OPENASE_DATABASE_DSN` | — | PostgreSQL 连接字符串(**必需**) |
| `OPENASE_SECURITY_CIPHER_SEED` | 空 | 可选的共享加密种子,用于 GitHub 凭据存储;当多个环境需要读取同一批加密记录时应显式设置 |
| `OPENASE_ORCHESTRATOR_TICK_INTERVAL` | `5s` | 编排器轮询间隔 |
| `OPENASE_LOG_FORMAT` | `text` | 日志格式(`text` 或 `json`) |
| `OPENASE_LOG_LEVEL` | `info` | 日志级别 |

`OPENASE_SECURITY_CIPHER_SEED` 对应配置文件里的 `security.cipher_seed`。如果未设置,OpenASE 会保持兼容行为,从 `database.dsn` 推导 GitHub 凭据加密种子。

## 配置文件查找顺序

1. `--config <path>` 命令行参数
2. `./config.yaml`(或 `.yml`、`.json`、`.toml`)
3. `~/.openase/config.yaml`
4. `OPENASE_*` 环境变量 + 内置默认值

## 认证

- 全新本地安装使用一次性的 **local bootstrap 链接**进行浏览器授权;不再暴露匿名管理员访问。
- **OIDC** 仍然是共享实例、团队环境和网络化部署的长期浏览器认证路径。
- 如果当前 OIDC 配置导致无法登录,先执行 `openase auth break-glass disable-oidc`,再通过 `openase auth bootstrap create-link --return-to /admin/auth --format text` 重新进入 `/admin/auth` 修复配置。

OIDC 支持标准提供商:Auth0、Azure Entra ID 以及任何 OpenID Connect 兼容的 IdP。

另见:

- IAM 双模式契约 — [English](../en/iam-dual-mode-contract.md) | [中文](./iam-dual-mode-contract.md)
- OIDC & RBAC 指南 — [English](../en/human-auth-oidc-rbac.md) | [中文](./human-auth-oidc-rbac.md)
