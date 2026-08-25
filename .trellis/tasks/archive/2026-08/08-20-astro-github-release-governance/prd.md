# 配置 Astro GitHub 发布治理

## Goal

在 GitHub 设置最小发布治理：测试/生产 Environments、生产启用开关和 `main` 分支保护，以配合 Astro 的镜像发布 workflow。

## Requirements

- 创建 `test`、`production` Environments；生产要求人工审批。
- 分别设置部署连接信息；运行时数据库/JWT/OAuth2/kubeconfig/GHCR拉取凭据不得放入 GitHub。
- 设置 `ASTRO_PRODUCTION_ENABLED=false`，直到独立生产 K8s 与专用 kubeconfig 就绪。
- `main` 普通协作者要求 PR + CI、禁止 force push；owner `cuihe500` 可直接 push/绕过检查，但仍触发 CI/CD。
- 不因生产未启用令正式 Release workflow 失败；应显示 production skipped。

## Acceptance Criteria

- [ ] GitHub API/UI 显示两个 Environment，production 有 reviewer 保护。
- [ ] 部署 secrets/variables 仅含 host、port、user、SSH private key、固定 SSH host key 等连接信息，并按环境隔离。
- [ ] repository variable 为 `ASTRO_PRODUCTION_ENABLED=false`。
- [ ] main 保护符合普通协作者与 owner bypass 规则，force push 被禁止。
- [ ] Release workflow 条件在未启用生产时明确 skipped。

## Out of Scope

- 修改主机部署入口、Compose、DNS、OpenResty、数据库或实际生产 K8s。
- 将 database/JWT/OAuth2/kubeconfig/GHCR拉取凭据上传 GitHub。
