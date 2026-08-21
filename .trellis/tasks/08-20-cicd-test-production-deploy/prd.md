# CI/CD 测试与生产部署

## Goal

建立最小、可追溯的 Astro 发布流程：部署主机不保存源码或执行构建，只从 GHCR 拉取不可变镜像；OpenResty 对外提供 HTTPS 和同源 Web/API 访问。

- `main` 的每次推送及普通 Git tag 推送自动发布测试环境。
- GitHub Release 的 `published` 事件在测试验证后按状态处理生产候选。
- 测试入口为 `https://astro-test.bytcloud.org`；生产入口为 `https://astro.bytcloud.org`。

## Confirmed Facts

- Astro 是 GitHub 单仓库 `cuihe500/Astro`。初始目标提交为 `d465a2d2f08df2a9c1f1fb3d55cc643afb9694db`；首轮 CI 发现 workflow 的 ShellCheck `SC2029` 后，以最小修复提交 `17f6d72e1de0754cfc083f7a12d74e9dfc036de4` 完成真实测试发布，本地与远端 `main` 一致。
- 仓库已包含安全运行时配置、API/Web ARM64 Dockerfile、`release.yml` 和发布文档；GitHub Run `32462154081` 已通过全部 Makefile 校验、双镜像构建和测试部署，生产 job 按条件跳过。
- GitHub `test` Environment 已配置三项受限连接变量和两项 SSH secret；当前仍无 `production` Environment、Release、分支保护或 ruleset。
- 测试与生产在同一台 Ubuntu 24.04 ARM64 主机运行。Docker/Compose、MariaDB、OpenResty、Certbot 和 JP/KR 公网入口已经可用；业务容器必须只监听 loopback。
- `/root/base-workspace/applications/astro/` 已建立测试/生产隔离的 Compose、数据库/用户、网络、秘密、测试 kubeconfig、受限部署入口、状态和回滚控制面；GitHub 测试部署公钥已安装到 forced-command 入口。
- `astro-test.bytcloud.org` 与 `astro.bytcloud.org` 已有独立 DNS、证书和 OpenResty 站点；测试 Web 与 `/health` 当前返回成功，生产环境保持禁用并返回 `503`。
- 当前测试 K8s 与独立测试 OAuth2 client 已就绪。应用命名空间固定为 `astro-user-{userID}`，所以测试与生产不得共用同一 K8s 集群。

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

### R8. 首次端到端测试发布

本轮最初以完整 commit SHA `d465a2d2f08df2a9c1f1fb3d55cc643afb9694db` 作为测试版本标识；首轮 CI 发现 workflow 静态检查问题后，按约定以最小修复提交 `17f6d72e1de0754cfc083f7a12d74e9dfc036de4` 重新触发真实 `main push` 发布链路，不额外创建 Git tag 或 GitHub Release。

首次发布必须完成：GitHub CI 校验、API/Web ARM64 GHCR 镜像推送、digest 受限部署、容器与 loopback 健康检查、OpenResty 公网验证，以及测试域名下的前端、`/health`、OAuth2 登录入口和核心应用生命周期烟测。测试数据使用明确命名的临时账号/应用，应用在验证后删除；不得触及生产环境。

如果目标提交因应用或流水线缺陷无法部署，只允许做实现目标所需的最小修复，并以新的 commit SHA 重新发布；最终结果必须明确记录实际部署 SHA 与两个镜像 digest。

## Delivery Map and Acceptance Criteria

本请求拆分为三个可独立验证的子任务；父任务只负责跨任务契约、顺序和最终集成检查。

| 子任务 | 责任边界 | 依赖与完成条件 |
|---|---|---|
| `08-20-astro-host-control-plane` | `/root/base-workspace` 内一次性建立测试/生产运行控制面：数据库/网络、Compose、秘密路径、受限部署入口、OpenResty、DNS/TLS、回滚与主机文档 | 先于自动测试部署；发布稳定的环境名、端口、部署入口与 digest 参数契约 |
| `08-20-astro-image-release-pipeline` | Astro 仓库内的运行时配置、API/Web 镜像定义、测试/构建/GHCR workflow、只传 digest 的受限部署调用及发布文档 | 依赖主机控制面给出的契约；可构建 ARM64 镜像并安全部署测试环境 |
| `08-20-astro-github-release-governance` | GitHub Environments、部署 secrets/variables、`main` 分支保护和正式 Release 审批/生产启用策略 | 与镜像 workflow 所需名称一致；生产 K8s 未启用时 job 明确跳过 |

父任务最终验收：

- [x] API 与 Web 可构建为私有、可追溯的 `linux/arm64` GHCR 镜像，且部署引用 digest。
- [ ] `main`、普通 tag、prerelease Release 和正式 Release 的测试发布行为符合 R3；正式 Release 的生产候选行为符合 R4。
- [x] 测试环境可通过 `astro-test.bytcloud.org` 同源访问前端、API、OAuth2 回调和 `/health`，不直接公开容器端口。
- [ ] 生产环境的首次控制面和端口/域名/秘密隔离已经建立，但在独立生产 K8s 就绪前，Release production job 明确跳过。
- [ ] CI 失败、镜像拉取失败或运行时健康检查失败时，当前可用版本不被破坏，且有可执行的回滚入口。
- [x] Astro 仓库不包含 Compose、OpenResty、主机脚本、运行状态、kubeconfig、数据库密码、JWT secret、OAuth2 secret 或 GHCR 拉取凭据。
- [x] `/root/base-workspace/applications/astro/` 的文档和控制面完成首次建立，并遵循该项目的备份、秘密和验证规范。
- [ ] `main` 分支保护和 GitHub Environments 按 R7 配置完成。
- [x] `d465a2d2f08df2a9c1f1fb3d55cc643afb9694db`（或为完成本次发布产生并明确记录的最小修复提交）通过 `main push` 完成 CI、GHCR、digest 部署和公网验证。
- [x] `https://astro-test.bytcloud.org` 可加载 Web，`/health` 返回成功，OAuth2 登录入口可达，临时用户可完成应用创建、查询、启动、停止、重启、日志和删除流程。

## Out of Scope

- Kubernetes GitOps、Argo CD、Flux、Helm、多节点高可用、自动扩缩容、集中日志或完整监控平台。
- 生产 K8s 集群、生产 kubeconfig 的创建和生产生产部署的启用；本任务只实现安全跳过机制及启用前置条件。
- 自动数据库备份、破坏性数据库迁移、数据库删除或跨环境数据复制。
- 修改 Astro 的业务 API、Kubernetes 应用编排语义或前端业务功能；仅允许为安全运行时配置、镜像化和静态托管所必需的最小改动。
- 将宿主机基础设施定义复制回 Astro 仓库，或让 CI 管理 OpenResty/DNS/证书/Compose。
- 本轮不创建 Git tag、prerelease 或正式 Release；这些事件分支在后续独立演练，不阻塞首次域名可见效果。

## Open Questions

无。本轮首次测试发布已经完成；父任务剩余验收项继续由对应子任务跟踪。
