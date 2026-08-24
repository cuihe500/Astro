## 目标

建立 Astro 的统一工作治理入口，使所有可实施工作都能从 GitHub Issue、GitHub Project、Trellis 任务追踪到 Pull Request 和最终验收。

## 范围

- 创建并配置 `Astro Development` GitHub Project。
- 建立 Feature、Bug Issue Forms 和 Pull Request 模板。
- 新增中文开发工作流文档。
- 将 GitHub 工作项关联加入 `AGENTS.md` 与 Trellis 生命周期门禁。
- 提供符合仓库命令规范的最小 Make 入口。
- 盘点未归档且尚未关联 GitHub 的 Trellis 任务。

## 非范围

- 不开发 GitHub 与 Trellis 双向同步服务。
- 不引入 PAT 驱动的 GitHub Action。
- 不为已归档历史任务补建 Issue。
- 不自动为现有未归档任务创建或关闭远程工作项。

## 验收标准

- [ ] `Astro Development` Project 包含约定的状态、类型、优先级、日期和 Trellis Task 字段。
- [ ] 仓库包含 Feature/Bug Issue Forms、Issue 配置和 PR 模板。
- [ ] 仓库包含完整的中文工作治理文档。
- [ ] AI 与 Trellis 流程明确禁止无 GitHub Issue/Project 关联的实施。
- [ ] 本 Issue、Project 条目和 `.trellis/tasks/08-24-github-trellis-work-governance` 相互关联。
- [ ] 当前未归档且缺少 GitHub 关联的 Trellis 任务形成待补录清单。

## Trellis

`.trellis/tasks/08-24-github-trellis-work-governance`
