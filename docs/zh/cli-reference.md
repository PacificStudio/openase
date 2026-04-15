# OpenASE CLI 参考

OpenASE 遵循 **GitHub 风格的双层 CLI 契约**:高层资源命令用于常用操作,加上一个原始 API 逃生口用于尚未被封装的场景。

## 资源命令

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

## 原始 API 逃生口

```bash
openase api GET  /api/v1/projects/$PID/tickets --query status_name=Todo
openase api PATCH /api/v1/tickets/$TID --field status_id=$SID
```

## 实时流

```bash
openase watch tickets $PROJECT_ID
```

## 输出格式化

```bash
--jq '<expr>'              # JQ 过滤器
--json field1,field2       # 选择字段
--template '{{...}}'       # Go 模板
```

`--kebab-case` 和 `--snake_case` 两种风格的参数名均可使用。

## Agent 平台环境变量

Agent Worker 从工作区包装器继承以下环境变量:

| 变量 | 用途 |
|------|------|
| `OPENASE_API_URL` | 平台 API 端点 |
| `OPENASE_AGENT_TOKEN` | Agent 认证 Token |
| `OPENASE_PROJECT_ID` | 当前项目上下文 |
| `OPENASE_TICKET_ID` | 当前工单上下文 |
