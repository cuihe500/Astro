# CI/CD 测试与生产部署

## Goal

建立最小、可追溯的 Astro 发布流程：部署主机不保存源码或执行构建，只从 GHCR 拉取不可变镜像；OpenResty 对外提供 HTTPS 和同源 Web/API 访问。

- `main` 的每次推送及普通 Git tag 推送自动发布测试环境。
- GitHub Release 的 `published` 事件在测试验证后按状态处理生产候选。
- 测试入口为 `https://astro-test.bytcloud.org`；生产入口为 `https://astro.bytcloud.org`。

## Confirmed Facts

- Astro 是 GitHub 单仓库 `cuihe500/Astro`，当前无 Actions workflow、GitHub Environment、Actions secret、分支保护或 ruleset。
- 后端是 Go/Gin，已有 `make test`、`make lint`、`make build` 和 `GET /health`；前端是 React + TypeScript + Vite，已有 `make frontend-check`，生产 API 默认同源 `/api/v1`。
- 当前未有 Dockerfile、应用镜像、CI workflow 或部署实现。后端现在读取 `configs/config.yaml`，且只有 OAuth2 的部分字段支持环境变量覆盖。
- 测试与生产在同一台 Ubuntu 24.04 ARM64 主机运行。Docker/Compose、MariaDB、OpenResty、Certbot 和 JP/KR 公网入口已经可用；业务容器必须只监听 loopback。
- 本机权威基础设施资料和未来主机控制面归属 `/root/base-workspace`。新增 Astro 主机资产必须遵循其应用文档、备份、秘密、OpenResty、Docker 和数据库规范。
- 现有 MariaDB 只监听 `127.0.0.1:3306`；每个环境必须使用独立数据库、数据库用户、秘密文件和 external Docker 网络。禁止新增 MariaDB 容器或复用另一环境的数据/凭据。
- 当前测试 K8s 可供测试环境使用。应用命名空间固定为 `astro-user-{userID}`，所以测试与生产不得共用同一 K8s 集群。
- 本机已有生产 Astro OAuth2 Provider；测试环境需要独立 OAuth2 client、secret 和严格回调地址。
- 两个 Astro 域名目前都没有 DNS 记录、单域名证书、OpenResty 站点或运行中的 Astro 服务。

## Requirements

### R1. 验证与构建

每次发布产物前，CI 必须执行：

- `make test`
- `make lint`
- `make build`
- `make frontend-check`

任一步失败不得推送镜像或部署。镜像必须以 Git commit SHA 绑定，并构建 `linux/arm64` 版本以匹配部署主机。

### R2. 镜像与追溯

CI 必须向私有 GHCR 推送独立的 API 与 Web 镜像。部署只能使用 `ghcr.io/...@sha256:...` 形式的不可变 digest；commit SHA 和正式 Release tag 仅作便于识别的标签，不得以浮动标签部署。

### R3. 测试发布路由

以下事件必须自动构建、推送并部署测试环境：

| 事件 | 测试环境行为 |
|---|---|
| `main` push | 部署对应 commit SHA 镜像 |
| 任意普通 Git tag push | 部署对应 tag 指向 commit 的 SHA 镜像 |
| GitHub Release `published` 且 `prerelease=true` | 部署该 Release tag 指向 commit 的 SHA 镜像 |
| GitHub Release `published` 且 `prerelease=false` | 先部署并验证该 Release 版本的测试环境，再进入生产候选判断 |

测试部署必须幂等；失败时保留并自动恢复上一个已知可用 digest，不删除数据库或其他应用。

### R4. 生产发布路由

只有 GitHub Release `published` 且 `prerelease=false` 才可进入生产部署候选。普通 tag、`main` 和 prerelease Release 永不直接部署生产。

生产 job 必须同时满足：CI 已成功、测试部署成功、GitHub `production` Environment 审批通过、生产部署开关已启用、独立生产 K8s 与专用 kubeconfig 已就绪。生产 K8s 未准备好时，正式 Release 的生产 job 必须显示为“已跳过/生产未启用”，不得把该前置条件误报为发布失败。

### R5. 环境隔离与运行时配置

测试和生产必须分别使用：

- 独立 Compose 项目、API/Web 容器、loopback 端口、Docker external 网络、MariaDB 数据库/用户、运行时配置、K8s kubeconfig、OAuth2 client/secret、JWT secret 和版本状态；
- 测试 API/Web 端口：`127.0.0.1:18080/18081`；生产预留：`127.0.0.1:18083/18084`；
- base-workspace 中 `/root/.secrets/astro/<environment>/` 下的 root-only 秘密文件。

部署运行时不能使用仓库默认 JWT、数据库凭据或开发配置。应用在 `ASTRO_RUNTIME_ENV=test|production` 时必须拒绝缺失、默认或不安全的关键配置。

### R6. 公网入口

本次首次部署由基础设施侧在 `/root/base-workspace` 一次性建立：Cloudflare DNS、每域名单独的 Certbot DNS-01 证书、两个 OpenResty 站点、应用路由和验证记录。

OpenResty 必须将 `/api/`（以及 API 健康路径）反代至同环境 API，其余路径反代至同环境 Web；Web 路由必须支持 SPA fallback。应用容器不得直接暴露公网端口。

### R7. 责任边界、权限和文档

- Astro 仓库只保存应用源码、镜像构建定义、GitHub Actions 和应用发布说明；不得保存、生成或同步 Compose、OpenResty、主机部署/回滚脚本、主机状态或秘密。
- 本次首次部署可在 `/root/base-workspace/applications/astro/` 建立上述基础设施控制面；以后所有 Compose、OpenResty、主机部署和回滚维护只在 base-workspace 独立进行，Astro CI 不上传或覆盖它们。
- CI 使用 GitHub-hosted Actions、私有 GHCR、专用 `astro-deploy` SSH 身份与强制命令入口。CI 只能传入白名单环境名和 API/Web image digest，不得执行任意 SSH 命令、拉取 Git 源码或使用 `latest`。
- GitHub `test` / `production` Environments 必须分离部署凭据。`production` 必须启用人工审批。
- `main` 必须启用分支保护：普通协作者要求 PR 和 CI 状态检查、禁止 force push；owner `cuihe500` 可绕过 PR/检查直接推送，但其推送仍必须触发同一测试流水线。
- Astro 仓库文档说明 CI、Release 和 GitHub 配置；主机部署、数据库、OpenResty、DNS、证书、秘密、回滚和验证细节只记录在 base-workspace 的 Astro 应用文档中。

## Delivery Map and Acceptance Criteria

本请求拆分为三个可独立验证的子任务；父任务只负责跨任务契约、顺序和最终集成检查。

| 子任务 | 责任边界 | 依赖与完成条件 |
|---|---|---|
| `08-20-astro-host-control-plane` | `/root/base-workspace` 内一次性建立测试/生产运行控制面：数据库/网络、Compose、秘密路径、受限部署入口、OpenResty、DNS/TLS、回滚与主机文档 | 先于自动测试部署；发布稳定的环境名、端口、部署入口与 digest 参数契约 |
| `08-20-astro-image-release-pipeline` | Astro 仓库内的运行时配置、API/Web 镜像定义、测试/构建/GHCR workflow、只传 digest 的受限部署调用及发布文档 | 依赖主机控制面给出的契约；可构建 ARM64 镜像并安全部署测试环境 |
| `08-20-astro-github-release-governance` | GitHub Environments、部署 secrets/variables、`main` 分支保护和正式 Release 审批/生产启用策略 | 与镜像 workflow 所需名称一致；生产 K8s 未启用时 job 明确跳过 |

父任务最终验收：

- [ ] API 与 Web 可构建为私有、可追溯的 `linux/arm64` GHCR 镜像，且部署引用 digest。
- [ ] `main`、普通 tag、prerelease Release 和正式 Release 的测试发布行为符合 R3；正式 Release 的生产候选行为符合 R4。
- [ ] 测试环境可通过 `astro-test.bytcloud.org` 同源访问前端、API、OAuth2 回调和 `/health`，不直接公开容器端口。
- [ ] 生产环境的首次控制面和端口/域名/秘密隔离已经建立，但在独立生产 K8s 就绪前，Release production job 明确跳过。
- [ ] CI 失败、镜像拉取失败或运行时健康检查失败时，当前可用版本不被破坏，且有可执行的回滚入口。
- [ ] Astro 仓库不包含 Compose、OpenResty、主机脚本、运行状态、kubeconfig、数据库密码、JWT secret、OAuth2 secret 或 GHCR 拉取凭据。
- [ ] `/root/base-workspace/applications/astro/` 的文档和控制面完成首次建立，并遵循该项目的备份、秘密和验证规范。
- [ ] `main` 分支保护和 GitHub Environments 按 R7 配置完成。

## Out of Scope

- Kubernetes GitOps、Argo CD、Flux、Helm、多节点高可用、自动扩缩容、集中日志或完整监控平台。
- 生产 K8s 集群、生产 kubeconfig 的创建和生产生产部署的启用；本任务只实现安全跳过机制及启用前置条件。
- 自动数据库备份、破坏性数据库迁移、数据库删除或跨环境数据复制。
- 修改 Astro 的业务 API、Kubernetes 应用编排语义或前端业务功能；仅允许为安全运行时配置、镜像化和静态托管所必需的最小改动。
- 将宿主机基础设施定义复制回 Astro 仓库，或让 CI 管理 OpenResty/DNS/证书/Compose。

## Open Questions

无。规划已收敛，等待设计与实施清单审阅后的明确实施批准。
