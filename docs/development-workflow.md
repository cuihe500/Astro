# GitHub 与 Trellis 开发流程

本文是 Astro 工作项治理的权威说明，适用于人和 AI。目标 Project 为 [Astro Development](https://github.com/users/cuihe500/projects/6)。

## 核心关系

每个可独立验收的工作必须满足：

```text
1 个 GitHub Issue = 1 个 Project 条目 = 1 个 Trellis 任务
```

| 载体 | 唯一职责 |
|---|---|
| GitHub Issue | 业务需求、问题证据和验收标准的事实源 |
| GitHub Project | 类型、优先级、负责人、日期和当前状态 |
| Trellis | 技术规划、研究、实现步骤和验证上下文 |
| Pull Request | 代码审查、验证结果和交付关系 |

纯问答、只读调查和不产生仓库或外部状态变更的评审不需要工作项；一旦决定实施代码、文档、配置或流程变更，必须先进入本流程。

## Project 字段

| 字段 | 值或规则 | 开始实施前必填 |
|---|---|:---:|
| Title | 与 Issue 标题一致 | ✅ |
| Status | Triage、Backlog、Ready、In Progress、In Review、Blocked、Done、Cancelled | ✅ |
| Work Type | Feature、Bug、Maintenance | ✅ |
| Priority | P0、P1、P2、P3 | ✅ |
| Assignee | 当前负责人 | ✅ |
| Start date | 实际开始实施的日期 | ✅ |
| Target date | 有明确承诺时填写 |  |
| Trellis Task | `.trellis/tasks/<任务目录>` | ✅ |

Project 已创建“总览”表格和“工作流”看板，6 个内建 workflow 已启用。Issue Form 会通过 `projects: ["cuihe500/6"]` 自动加入 Project，关闭 Issue 后由内建 workflow 转为 `Done`。

以下两项尚未完成，GitHub API 无法配置，仓库所有者需在 Project #6 的 UI 设置一次：

1. 打开“工作流”视图，将 `Group by` 设为 `Status`。
2. 打开 `Workflows → Auto-add to project`，过滤条件设为 `repo:cuihe500/Astro is:issue`，作为 CLI/API 创建 Issue 的兜底。

## 状态定义

| Status | 进入条件 | Trellis 状态 |
|---|---|---|
| Triage | Issue 刚创建，等待分类 | 无任务 |
| Backlog | 有价值，但当前不排期 | 无任务 |
| Ready | 范围和验收标准清楚，优先级与负责人明确 | 可创建 `planning` 任务 |
| In Progress | 规划获批并开始实施 | `in_progress` |
| In Review | PR 已创建，等待审查或验收 | `in_progress` |
| Blocked | 已记录阻塞原因、责任人和下一检查时间 | 保持原状态 |
| Done | PR 已合并、验收通过、Issue 已关闭 | 归档 |
| Cancelled | 明确不再实施，并记录原因 | 记录结果后归档 |

## 标准流程

### 1. 建立工作项

1. Feature 使用“功能需求”表单；Bug 使用“缺陷报告”表单；重构、文档、依赖和运维使用“维护任务”表单。
2. Issue 必须写清模板中的全部必填项，并确认已加入 Astro Development。
3. 设置 Work Type、Priority、Assignee 和 Status。只有达到 `Ready` 才进入 Trellis 规划。
4. AI 在进行远程操作前必须获得用户授权；所有日常 GitHub CLI 操作使用 `make github GITHUB_ARGS='<参数>'`。无法认证、无权限或无法核对 Project 关联时停止，不得先实施后补录。

### 2. 创建并评审 Trellis 规划

获得用户创建任务的同意后，通过唯一 Make 入口创建任务并立即保存关联：

```bash
make trellis TRELLIS_ARGS='create "<标题>" --slug <slug> --meta github_issue=<Issue完整URL> --meta github_project=https://github.com/users/cuihe500/projects/6'
```

完成 `prd.md`；复杂任务同时完成 `design.md`、`implement.md` 和上下文清单。业务范围或验收标准以 Issue 为准，Trellis 不得另建一套相互冲突的需求。

规划获批后，在 Project 填写 Start date 和 Trellis Task，将 Status 改为 `In Progress`，确认所有必填字段后再启动：

```bash
make trellis TRELLIS_ARGS='start <任务目录>'
```

未满足以上条件时，只能补充 Issue 或规划，禁止修改产品代码、文档、配置和部署状态。

### 3. 实施、验证和审查

1. 按 Trellis 规划实施，只使用 Makefile 已提供的命令。
2. 执行与变更匹配的校验；治理文件可运行 `make governance-check TASK=<任务目录>`。
3. 创建 PR 前将 Project Status 改为 `In Review`。
4. PR 必须填写 Issue 和 Trellis 路径。完整交付使用 `Fixes #<编号>`，部分交付使用 `Refs #<编号>`；部分交付不得关闭 Issue 或归档 Trellis 任务。

### 4. 完成或取消

- PR 合并且验收通过后关闭 Issue，确认 Project 为 `Done`，再执行 `make trellis TRELLIS_ARGS='archive <任务目录>'`。
- `task.py finish` 只清除当前会话指针，不代表工作完成；不得用它替代归档。
- 放弃工作时，在 Issue 记录原因并以 `Not planned` 关闭，将 Project 改为 `Cancelled`，执行 `make trellis TRELLIS_ARGS='set-meta <任务目录> outcome cancelled'` 后归档。
- 阻塞时将 Project 改为 `Blocked`，在 Issue 记录原因、责任人和下一检查时间；解除后恢复到原流程状态。

## 特殊场景

### 父子任务

只有能独立验收的交付才建立 Sub-issue、Project 条目和 Trellis 子任务。单纯实现步骤只写入父任务的 `implement.md`，不得为了拆步骤制造工作项。父子关系不代表依赖顺序，依赖必须写入子任务规划。

### 需求变更

先修改 Issue 的范围和验收标准，再同步 Trellis PRD/设计；变更导致独立交付或原 Issue 无法准确表达时，新建 Issue 和 Trellis 任务。不得只修改 Trellis 文档。

### 紧急修复

正常情况下仍须先建立 Bug Issue。只有用户明确授权且生产故障需要立即止损时，才可先实施；最迟在 PR 审查或合并前补齐 Issue、Project 字段和 Trellis 关联。AI 不得自行认定紧急例外。

### 安全漏洞

不得创建公开 Bug Issue，也不得把漏洞细节、Token 或密钥写入 Project、Trellis、日志或 PR。使用 GitHub Security Advisory 或受限私有工作项；`meta.github_issue` 只保存有权限控制且不泄密的引用。无法建立安全关联时停止并联系仓库所有者。

## AI 执行门禁

AI 每次进入实施前必须核对：

- Issue 类型正确、内容可验收且仍是最新事实源。
- Issue 已加入唯一 Project，且状态和必填字段完整。
- Trellis `task.json.meta.github_issue` 与 `meta.github_project` 非空并指向当前工作项。
- Project 的 Trellis Task 与当前目录一致。
- 用户已批准规划，Trellis 状态为 `in_progress`。

任一项无法验证时立即停止实施，报告缺失项并等待处理；不得猜测 URL、编号或字段值。

## 存量待补录

以下未归档任务在恢复实施前必须先建立或确认 Issue、加入 Project 并补齐元数据；本流程不替它们自动创建远程工作项：

- `.trellis/tasks/08-20-cicd-test-production-deploy`
- `.trellis/tasks/08-20-astro-github-release-governance`
- `.trellis/tasks/08-20-astro-image-release-pipeline`
- `.trellis/tasks/08-20-astro-host-control-plane`

已归档历史任务不追溯迁移。
