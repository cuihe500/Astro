# GitHub 与 Trellis 工作治理实施清单

## 远程工作项

- [x] 通过 GitHub CLI 只读确认仓库、owner、认证 scopes 和已有 Projects。
- [x] 创建 `Astro Development` Project #6，记录 URL/number/id。
- [x] 创建本治理 Issue #1，加入 Project，并回填当前 Trellis `github_issue` / `github_project` 元数据。
- [x] 配置 Project Status、Work Type、Priority、Start date、Target date、Trellis Task 字段。
- [x] 配置“总览”表格、“工作流”看板并复核 6 个内建 workflow 已启用；API 不支持项已写入 UI 清单。

## 仓库内容

- [x] 新增 Feature/Bug Issue Forms 与 Issue 配置。
- [x] 新增 PR 模板。
- [x] 新增 `docs/development-workflow.md`。
- [x] 更新 `AGENTS.md`，加入不可跳过的 GitHub/Trellis 门禁与例外规则。
- [x] 更新 `.trellis/workflow.md` 的任务创建、planning、start、review 和完成规则。
- [x] 为日常 Trellis 生命周期增加最小 Make 入口，并同步文档命令。
- [x] 生成当前未归档 Trellis 任务待补录清单，不改历史归档任务。

## 验证

- [x] 使用 Makefile 中已有/新增目标验证文档引用、模板结构和 Trellis 元数据。
- [x] 运行 `make workflow-lint`。
- [x] 复核 `git diff`，确认未引入 PAT、同步 Action 或无关字段。
- [x] 通过 GitHub CLI 复核 Project、字段和本治理条目。

## GitHub UI 待办

- [ ] 将“工作流”视图的 `Group by` 设置为 `Status`。
- [ ] 启用仓库级 `Auto-add to project`，过滤条件为 `repo:cuihe500/Astro is:issue`。

## 风险与回滚点

- GitHub CLI 缺少 `project` scope 时停止远程写入并请求授权，不创建重复 Project。
- 视图和 Project 原生 workflow 若无公开写 API，保留最短 UI 操作清单。
- 不删除远程 Project、字段、Issue 或现有 Trellis 任务作为失败回滚手段。
