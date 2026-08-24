# GitHub 与 Trellis 工作治理设计

## 边界

使用现有系统承担各自职责，不新增同步服务：

1. GitHub Issue：需求与验收事实源。
2. GitHub Project：状态、类型、优先级、负责人和日期。
3. Trellis：技术规划、执行上下文和仓库内任务状态。
4. Pull Request：变更审查、验证证据和 Issue 关闭关系。

## 数据关联

- Trellis `task.json.meta.github_issue` 保存 Issue 完整 URL。
- Trellis `task.json.meta.github_project` 保存 Project 完整 URL。
- Project 的 `Trellis Task` 保存 `.trellis/tasks/<task>` 路径。
- PR 正文保存 `Fixes #<issue>` 或 `Refs #<issue>`，并填写 Trellis 路径。

不保存重复的 Project item node ID；当前没有程序需要消费它，URL 和 Project 字段足以人工及 AI 核对。

## 生命周期

```text
Issue 创建 → Project/Triage → 评审/Ready
→ Trellis planning → Project/In Progress + Start date
→ 实施与验证 → PR/Project In Review
→ 合并与验收 → Issue closed + Project Done → Trellis archive
```

Blocked 和 Cancelled 是显式分支。需求变更先修改 Issue，再同步 Trellis PRD；不得让 PRD 成为另一份业务事实源。

## 强制执行面

- `AGENTS.md`：仓库级、跨会话的 AI 规则。
- `.trellis/workflow.md`：按 no_task/planning/in_progress 状态反复注入门禁。
- Issue/PR 模板：在 GitHub 输入边界收集必需信息。
- Makefile：为 Trellis 生命周期提供项目允许的命令入口。

首版不增加网络校验脚本。GitHub 权限、Project 归属和字段由 AI 在执行前通过 GitHub CLI 检查；Project 原生自动化负责常规加入与关闭状态。出现实际漂移后再增加 CI 审计。

## GitHub Project 初始化

- 先只读确认当前仓库、认证 scopes 和同名 Project。
- 同名且属于目标 owner 时复用，否则创建。
- 使用 GitHub CLI/API 创建字段并更新原生 Status 选项；工作类型字段使用 `Work Type`，因为 GitHub 保留 `Type` 名称。
- GitHub API 不支持的视图和内建 workflow 配置通过 UI 完成，不使用额外 Action 绕行。

## 兼容与迁移

- 当前治理任务属于一次性 bootstrap；远程 Issue 建立后立即补齐 meta。
- 现有未归档任务只生成待补录清单，避免猜测 Issue 含义或远程状态。
- 已归档任务保持不变。
- Trellis 上游更新可能提示本地 workflow 冲突；项目治理规则以当前仓库文件为准，并在更新时人工保留。

## 回滚

- 仓库模板和文档可通过单个工作提交回滚。
- GitHub Project 字段删除可能丢失值，实施时只新增或改名；若初始化失败，保留 Project 并报告未完成项，不自动删除远程数据。
