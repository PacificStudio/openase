# 模块关系图

## 各模块如何协作

```
Settings ──→ 定义状态、关联仓库
  │
  ├─→ Machines ──→ 注册执行环境
  │
  ├─→ Agents ──→ 注册 AI 执行者
  │
  ├─→ Skills ──→ 创建可复用技能包
  │     │
  │     ▼
  ├─→ Workflows ──→ 定义执行模板（绑定 Agent + Skills + 状态触发）
  │     │
  │     ▼
  ├─→ Tickets ──→ 创建工单（关联 Workflow）
  │     │                    │
  │     │         ┌──────────┘
  │     ▼         ▼
  │   Scheduled Jobs ──→ 按计划自动创建工单
  │
  ├─→ Activity ──→ 自动记录所有事件
  │
  └─→ Updates ──→ 人工发布项目进展
```

## 典型工作流

以下是一个完整的工作循环：

### 1. 基础设施层（一次性配置）

```
Settings → 配置状态和仓库
Machines → 注册执行环境
Agents   → 注册 AI Provider
```

### 2. 模板层（按需创建）

```
Skills    → 定义可复用的技能包
Workflows → 创建执行模板，绑定 Agent、Skills、状态触发条件
```

### 3. 执行层（日常使用）

```
Tickets        → 手动创建工单，触发 Agent 执行
Scheduled Jobs → 自动定时创建工单
```

### 4. 观测层（持续监控）

```
Activity → 实时查看系统事件
Updates  → 人工记录项目进展
```

## 数据流向

```
                    ┌─────────────┐
                    │  Scheduled  │
                    │    Jobs     │
                    └──────┬──────┘
                           │ 自动创建
                           ▼
┌──────────┐      ┌─────────────┐      ┌─────────────┐
│   User   │─────→│   Ticket    │─────→│  Workflow    │
└──────────┘ 创建  └──────┬──────┘ 关联  └──────┬──────┘
                           │                     │ 包含
                           │ 领取                ▼
                           ▼            ┌─────────────┐
                    ┌─────────────┐     │   Skills    │
                    │   Agent     │◄────┘
                    └──────┬──────┘ 调用
                           │
                           │ 在...上执行
                           ▼
                    ┌─────────────┐
                    │  Machine    │
                    └──────┬──────┘
                           │
                           │ 产生事件
                           ▼
                    ┌─────────────┐
                    │  Activity   │
                    └─────────────┘
```

## Remote Runtime v1 边界

远程机器执行现在只有一条正式 runtime 平面：

- 本地机器继续使用 `local_process`
- 远程机器一律通过 websocket 执行
- `ws_listener` 表示控制面直接拨号到机器公布的 listener
- `ws_reverse` 表示 `openase machine-agent run` 保持反向 websocket 会话，并通过该通道承载 runtime 消息
- SSH bootstrap 与 SSH diagnostics 保持在执行平面之外，只作为 helper 操作存在

这层拆分在运维上很重要：

- 机器拓扑决定由谁主动发起连接
- websocket runtime contract 决定命令、进程和产物如何传输
- SSH helper 只负责引导和修复，工单执行不会回退到 SSH
