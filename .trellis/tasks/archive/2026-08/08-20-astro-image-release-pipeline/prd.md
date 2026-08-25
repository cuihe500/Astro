# 实现 Astro 镜像与发布流水线

## Goal

仅在 Astro 仓库实现安全运行时配置、API/Web ARM64 镜像定义、GitHub Actions 构建/测试/GHCR 发布/受限部署调用及应用侧发布说明。

## Requirements

- 测试/生产运行时必须通过明确环境变量覆盖关键配置，并拒绝缺失或默认敏感配置；开发本地行为保持可用。
- 添加 API 和 Web 多阶段 Dockerfile 及最小 `.dockerignore`，不加入基础设施 Compose/OpenResty/host 脚本或秘密。
- workflow 在 `main`、任意 tag 和 Release published 事件运行检查、构建 ARM64 镜像并部署测试环境。
- 只有正式非 prerelease Release 能在测试成功后进入生产候选；生产未启用时明确 skipped。
- 部署 SSH 调用只传环境名、API digest、Web digest，依赖 base-workspace 已有强制命令。

## Acceptance Criteria

- [ ] `make test`、`make lint`、`make build`、`make frontend-check` 通过后才发布镜像。
- [ ] API/Web 镜像带 OCI source/revision metadata，构建 `linux/arm64` 并可解析 digest。
- [ ] 运行时 `ASTRO_RUNTIME_ENV=test|production` 缺失/默认 JWT/空密码/开发回调/kubeconfig 时启动前失败，且有最小测试覆盖。
- [ ] workflow 对 main/tag/prerelease/正式 Release 的测试和生产条件符合父任务 PRD。
- [ ] Astro 仓库不包含 Docker Compose、OpenResty、主机部署脚本、secret 或 kubeconfig。
- [ ] 发布说明只描述 CI 和外部契约，不复制 base-workspace 的基础设施文件。

## Out of Scope

- 创建或修改 `/root/base-workspace`、DNS、OpenResty、数据库、证书、SSH 账号或 GitHub Environment 配置。
- 创建生产 K8s 或启用生产部署。

## Dependency

依赖 `08-20-astro-host-control-plane` 给出稳定 SSH 强制命令：

```text
deploy <test|production> <api-digest> <web-digest>
```
