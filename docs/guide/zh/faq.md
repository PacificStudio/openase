# 常见问题（FAQ）

## Agent 相关

### Agent 创建了但不执行工单？

检查以下几项：

1. 是否有可用的 [Machine](./machines.md) 且状态正常
2. [Workflow](./workflows.md) 是否正确绑定了"领取状态"
3. 工单的当前状态是否在 Workflow 的领取状态列表中
4. Agent 是否处于"活跃"状态（非暂停或退役）

### Agent 执行失败了怎么办？

1. 在 [Activity](./activity.md) 中查看错误事件
2. 进入 Agent Run 详情查看输出日志
3. 检查目标 [Machine](./machines.md) 是否在线且资源充足
4. 检查 [Workflow](./workflows.md) 的 Harness 指令是否正确

## 工单相关

### 工单长时间停留在"进行中"？

1. 查看 [Activity](./activity.md) 日志了解最新事件
2. 检查 Agent Run 详情查看是否有报错
3. 确认目标 Machine 是否在线且资源充足
4. 检查 Workflow 是否设置了 `TimeoutMinutes`

### 如何让不同类型的工单由不同的 Agent 处理？

1. 创建多个 [Workflow](./workflows.md)，分别绑定不同的 Agent
2. 创建工单时选择对应的 Workflow
3. 例如：`代码开发` Workflow 绑定 Claude Code，`测试` Workflow 绑定另一个 Agent

### 工单误归档了怎么恢复？

进入 [Settings](./settings.md) → 归档工单，找到目标工单并点击恢复。

## 工作流相关

### 如何写好一个 Harness？

参考 [Workflows 文档](./workflows.md) 中的 Harness 编写建议。核心原则：

- 明确角色定位
- 提供具体步骤
- 设置验收标准
- 使用模板变量注入工单上下文

### 退役工作流会影响现有工单吗？

会。退役前请先查看影响分析，了解哪些工单和定时任务在使用该工作流。可以使用"替换引用"功能批量迁移到新工作流。

## 机器相关

### 机器显示"离线"怎么办？

1. 确认机器是否开机且网络正常
2. 检查 SSH 服务是否运行
3. 确认 SSH 端口和用户名是否正确
4. 在 Machines 页面点击"测试连接"获取诊断信息

## 定时任务相关

### 定时任务没有按时创建工单？

1. 确认任务处于"已启用"状态
2. 检查 Cron 表达式是否正确（查看"下次运行时间"）
3. 确认绑定的工作流仍然存在且未退役

### 如何测试定时任务？

使用"手动触发"功能立即执行一次，无需等待下次调度时间。
