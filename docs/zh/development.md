# OpenASE 开发指南

本页说明仓库结构、构建命令、质量门禁与测试,面向贡献者。终端用户安装与运行模式参见 [`source-build-and-run.md`](./source-build-and-run.md)。

## 仓库结构

```
openase/
├── cmd/openase/              # CLI 入口
├── internal/
│   ├── app/                  # 应用组装(serve / orchestrate / all-in-one)
│   ├── httpapi/              # HTTP API、SSE、Webhook、内嵌 UI
│   ├── orchestrator/         # 调度、健康检查、重试
│   ├── workflow/             # 工作流服务、Harness、钩子、Skill
│   ├── agentplatform/        # Agent Token 认证
│   ├── setup/                # 首次安装
│   ├── builtin/              # 内置角色 & Skill 模板
│   └── webui/static/         # 内嵌前端输出
├── web/                      # SvelteKit 控制面板源码
├── docs/
│   └── guide/                # 用户指南(按模块组织)
├── config.example.yaml
├── Makefile
└── go.mod
```

## 构建命令

```bash
make hooks-install        # 设置 git hooks(lefthook)
make check                # 运行格式化 + 后端覆盖率检查
make build-web            # 构建前端资产 + Go 二进制(不刷新 OpenAPI 产物)
make build                # 仅构建 Go 二进制(使用已有前端)
make run                  # 以开发模式运行 API 服务器
make doctor               # 运行本地环境诊断
```

## 前端质量门禁

```bash
make web-format-check     # Prettier 格式化
make web-lint             # ESLint 检查
make web-check            # Svelte 类型检查
make web-validate         # 以上全部
```

## OpenAPI 契约

```bash
make openapi-generate     # 重新生成 api/openapi.json + TS 类型
make openapi-check        # 验证已提交的产物是否最新
```

## 测试

```bash
make test                        # Go 测试套件
make test-backend-coverage       # 完整后端测试 + 覆盖率门禁
make lint                        # 对 origin/main 以来的变更进行 Lint
make lint-all                    # 完整 Lint 套件
```
