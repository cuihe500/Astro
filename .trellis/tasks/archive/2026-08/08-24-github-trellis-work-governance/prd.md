# 建立 GitHub 与 Trellis 工作治理流程

## 目标

为 Astro 建立一个可执行、可追踪的工作入口：所有需要实施的工作先进入 GitHub Issue 和指定 GitHub Project，随后才允许创建并启动对应的 Trellis 任务；功能、缺陷、PR、Project 状态和 Trellis 生命周期之间保持可核对的一对一关系。

## 背景与事实

- 仓库当前只有发布 workflow，没有 Feature/Bug Issue Form 或 PR 模板。
- `.trellis/config.yaml` 未启用生命周期 hook。
- Trellis 已支持通过 `task.json.meta` 保存外部工作项，不需要新增任务数据结构。
- 当前未归档 Trellis 任务的 `meta` 为空；本任务是治理流程首次引导，因此以 `governance_bootstrap=true` 创建，待 GitHub 工作项建立后补齐正式关联。
- 根目录规范要求开发命令使用 Makefile 已有目标，但当前 Makefile 没有 Trellis 生命周期入口；正式流程必须消除该冲突。
- 既有任务资料指向仓库 `cuihe500/Astro`；实施时仍须通过 GitHub CLI 只读检查确认仓库 owner、认证状态和是否已有同名 Project，避免重复创建。

## 需求

### R1 GitHub Project

- 在 `cuihe500` 名下复用或创建唯一 Project：`Astro Development`。
- 以 Project 内建 Title、Assignees、Labels、Repository、Linked pull requests 为基础，并配置：
  - Status：Triage、Backlog、Ready、In Progress、In Review、Blocked、Done、Cancelled。
  - Work Type：Feature、Bug、Maintenance（GitHub 保留 `Type` 名称）。
  - Priority：P0、P1、P2、P3。
  - Start date：实际开始日期。
  - Target date：可选目标日期。
  - Trellis Task：进入实施后填写仓库内任务目录。
- 提供“总览”表格视图和“工作流”看板视图；若 GitHub API/CLI 不支持视图配置，完成可自动配置部分并给出最短的 UI 操作清单。
- 使用 GitHub Projects 原生工作流自动加入本仓库新 Issue，并在 Issue 关闭后进入 Done；若原生工作流无法由 API 配置，记录需要由仓库所有者完成的 UI 步骤，不引入 PAT Action 替代。

### R2 GitHub 工作项模板

- Feature Issue Form 必须收集用户价值、范围、非范围、验收标准、依赖和影响面。
- Bug Issue Form 必须收集影响、期望/实际行为、复现步骤、环境、证据和回归验收标准。
- 禁止空白公开 Issue；安全漏洞引导至 GitHub Security Advisory，禁止公开敏感信息。
- PR 模板必须要求关联 Issue、Trellis 任务、验证结果及治理检查；完整交付使用关闭关键字，部分交付只引用 Issue。

### R3 仓库流程文档

- 新增一份中文、面向人和 AI 的权威工作治理文档，说明事实源、字段、状态、Feature/Bug 流程、父子任务、需求变更、紧急修复、安全漏洞和完成定义。
- 明确唯一映射：`1 个可独立验收的 Trellis 任务 = 1 个 GitHub Issue = 1 个 Project 条目`。
- Issue 保存业务需求与验收标准；Project 保存治理状态；Trellis 保存技术规划与执行上下文；PR 保存代码审查与交付记录。

### R4 AI 与 Trellis 门禁

- 在根目录 `AGENTS.md` 增加不可跳过的工作治理规则。
- 在 `.trellis/workflow.md` 的 no_task、planning、启动和收尾阶段加入 GitHub 工作项门禁，使规则能够随工作流状态持续注入 AI 上下文。
- 创建 Trellis 任务时必须保存：
  - `meta.github_issue`：当前仓库 Issue 完整 URL。
  - `meta.github_project`：指定 Project 完整 URL。
- `task.py start` 前必须确认 Issue 已加入 Project，并填写 Work Type、Priority、Assignee、Status、Start date 和 Trellis Task。
- Bug 必须使用 Bug Issue；Feature 必须使用 Feature Issue；Maintenance 同样必须有 Issue 和 Project 条目。
- 未获得 GitHub 权限或无法验证关联时，AI 必须停止实施并向用户报告，不能先写代码后补录。
- GitHub 管理操作和 Trellis 生命周期操作应通过文档明确允许的 Make 目标执行；一次性 GitHub Project 初始化保留明确的引导例外。

### R5 迁移与例外

- 为本治理任务创建对应 GitHub Issue、加入 Project，并回填 Trellis 元数据。
- 盘点当前未归档 Trellis 任务，输出待补录清单；不擅自为历史任务创建或关闭 Issue。
- 已归档历史任务不追溯迁移。
- 紧急修复最迟在 PR 审核或合并前补齐工作项，不能以紧急为由永久跳过。
- 安全漏洞不得创建公开 Bug Issue；使用 Security Advisory 或受限私有工作项，并只在 Trellis 中保存不泄密的引用。

## 验收标准

- [x] GitHub 中存在唯一的 `Astro Development` Project，字段和状态与本 PRD 一致。
- [x] 本治理任务有对应 Issue、Project 条目和非空 `task.json.meta.github_issue/github_project`。
- [x] 仓库包含可用的 Feature/Bug Issue Form、Issue 配置和 PR 模板。
- [x] 仓库包含一份中文工作治理文档，覆盖完整生命周期与例外路径。
- [x] `AGENTS.md` 和 `.trellis/workflow.md` 都明确禁止无 GitHub 工作项的 Trellis 实施。
- [x] Makefile 提供完成日常 Trellis 生命周期所需的最小入口，文档示例不再要求 AI 绕过 Makefile。
- [x] Project 原生自动化已启用；API 无法完成的 UI 配置项被明确列出并标记为待所有者操作。
- [x] 当前未归档且缺少关联的 Trellis 任务形成清单，历史归档任务未被改动。
- [x] 运行仓库现有相关 Make 校验目标通过，Issue Form 完成结构与 Project 关联校验。

## 非目标

- 不开发 GitHub 与 Trellis 双向同步服务。
- 不引入 PAT 驱动的 GitHub Action。
- 不为历史归档任务补建 Issue。
- 不增加估点、迭代、复杂度评分等当前未需要的 Project 字段。
- 不自动修改或关闭现有未归档任务对应的远程工作项。

## 阻塞问题

无。Project owner/title 和首次引导策略将在最终规划审批时一并确认。
