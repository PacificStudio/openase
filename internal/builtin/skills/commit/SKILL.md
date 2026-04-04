---
name: "commit"
description: "Write concise Conventional Commit messages that match the actual scope of the change."
---

# Conventional Commit

在提交前整理变更范围，并使用 Conventional Commit 风格：

- `feat(scope): ...` 用于可见能力新增
- `fix(scope): ...` 用于缺陷修复
- `refactor(scope): ...` 用于无行为变化的重构
- `test(scope): ...` 用于测试补充

避免把多个不相关改动塞进同一个提交说明。标题写结果，不写过程。
