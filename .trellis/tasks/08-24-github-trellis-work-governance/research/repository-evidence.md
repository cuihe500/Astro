# 仓库治理现状证据

## 已确认

- `.github/` 仅存在 `.github/workflows/release.yml`，没有 Issue Form、Issue config 或 PR 模板。
- `.trellis/config.yaml` 的 lifecycle hooks 仍为注释示例，没有启用项目 hook。
- `.trellis/scripts/task.py` 支持创建时重复传入 `--meta key=value`，也支持 `set-meta` 回填。
- `.trellis/workflow.md` 当前要求先获许可再创建 Trellis 任务，但没有 GitHub Issue/Project 前置门禁。
- `Makefile` 当前没有 Trellis 或 GitHub Project 管理目标。
- 未归档任务共五个；除本任务只有 `governance_bootstrap=true` 外，其余任务的 `meta` 均为空：
  - `08-20-cicd-test-production-deploy`
  - `08-20-astro-github-release-governance`
  - `08-20-astro-image-release-pipeline`
  - `08-20-astro-host-control-plane`
  - `08-24-github-trellis-work-governance`
- 已归档任务不在迁移范围。

## 远程实施结果

- GitHub CLI 已认证为 `cuihe500`，具备 `repo` 与 `project` scopes。
- 当前 remote 对应 `cuihe500/Astro`，Issues 已启用。
- 已创建 `Astro Development` Project：`https://github.com/users/cuihe500/projects/6`。
- 已创建本治理 Issue：`https://github.com/cuihe500/Astro/issues/1`。
- Project 已关联 Astro 仓库并公开；Issue #1 已填写 Assignee、Status、Work Type、Priority、Start date 和 Trellis Task。
- 已创建“总览”表格与“工作流”看板；公开 API 无法设置看板分组，需要在 UI 将 `Group by` 设为 `Status`。
- Project 自动生成的 6 个内建 workflow 均已启用；Issue Forms 会直接加入 Project，仓库级兜底 Auto-add 仍需在 UI 以 `repo:cuihe500/Astro is:issue` 启用。
- GitHub GraphQL schema 没有创建或更新 Project workflow 的 mutation，因此未用 PAT Action 绕行。
- GitHub 保留 `Type` 字段名，因此使用等价的 `Work Type`。
