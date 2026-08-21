# CI/CD 设计研究记录

## 目的

为 Astro 的测试/生产发布规划验证 GitHub Actions 事件语义、现有主机基础设施约束和最小安全部署方式。

## 已验证的仓库与主机事实

- GitHub 仓库为公开仓库 `cuihe500/Astro`；当前无 Actions workflow、Actions secret、Actions variable、GitHub Environment、自托管 runner、分支保护或 ruleset。
- GitHub Actions 已启用，默认 workflow token 权限为 `contents: read`；发布 GHCR 镜像的 workflow 必须显式声明 `packages: write`。
- 当前 `gh` token 缺少 `read:packages`，因此无法枚举现存 GHCR 包；实现不得假定包已存在。首次 workflow 推送应建立/关联包，部署主机另配置仅 `read:packages` 的拉取凭据。
- 本机为 Ubuntu 24.04 ARM64；Docker 29.7、Docker Compose 5.4 可用，默认 Buildx builder 仅显示 `linux/arm64`。GitHub-hosted Actions 构建应显式产生 `linux/arm64` 镜像，避免主机拉到不兼容架构。
- 本机运行 OpenResty：每个站点单独配置 `80/443/10080/10443`，并分别使用单域名 Certbot DNS-01 证书；应用端口均绑定 `127.0.0.1`。OpenResty 必须先 `openresty -t` 再 reload。
- 当前 MariaDB 容器监听 `127.0.0.1:3306`。现有应用通过显式 external Docker 网络和 `mariadb` 别名接入，且每个应用使用独立数据库、用户、密码文件。
- `/root/base-workspace/AGENTS.md` 规定新增应用必须在 `/root/base-workspace/applications/<slug>/` 建立 README；所有运行时秘密只可位于 `/root/.secrets/<component>/`；变更备份只可位于 `/root/base-workspace/backup/<scope>/<UTC timestamp>/`。
- 已确认测试端口 `127.0.0.1:18080/18081` 和生产预留端口 `127.0.0.1:18083/18084` 当前未被占用。
- 后端已有 `GET /health`。前端生产 API 默认同源 `/api/v1`，因此 OpenResty 路由 `/api/` 到 API、其余路径到 Web 容器可避免 CORS 和前端运行时配置复杂度。
- 后端当前只允许 OAuth2 secret 与 redirect URL 的环境覆盖；部署前必须补齐数据库、JWT、服务模式、日志和 kubeconfig 的安全运行时配置机制，不能让生产沿用仓库的开发默认配置。
- 当前 K8s 命名空间固定为 `astro-user-{用户ID}`。测试/生产若共用集群将产生跨环境资源冲突，因此测试仅接入当前测试集群；生产部署在独立生产集群及专用 kubeconfig 就绪前明确跳过。

## GitHub Actions 与 Release 事件结论

来源：GitHub Docs 的 workflow event、release REST API、Packages 文档；检索日期 2026-08-20。

- `release` 事件可通过 `types: [published]` 触发 workflow；它与普通 Git tag 的 `push` 事件不同。
- Release 负载包含 `prerelease` 状态。因此生产 job 应以 `release.published && !release.prerelease` 为条件；draft 不会作为 `published` 发布。
- 普通 tag push 和 `main` push 通过同一测试发布路径处理。Release 引用的 tag 也可能触发 push workflow，因此测试部署必须可以幂等；建议测试部署 workflow 使用并发组取消旧运行，只部署最新 commit SHA，release workflow 另走生产判定。
- 发布容器镜像应使用 `GITHUB_TOKEN` 与 `permissions: {contents: read, packages: write}`；Dockerfile 添加 OCI source label 以关联仓库。
- GitHub Environment 可为测试/生产分别保存 secrets；生产 Environment 应要求人工审批。生产 kubeconfig 尚未配置时，在 workflow 条件中明确跳过 production job，而不是使 Release workflow 失败。

## 分支保护结论

- 当前 `main` 无保护，仓库唯一协作者及 owner 是 `cuihe500`。
- 目标策略：对普通协作者要求 PR 和 CI 状态检查，禁止 force push；允许仓库 owner `cuihe500` 绕过 PR/检查直接 push。该绕过只绕过合并规则，不绕过后续测试 CI/CD workflow。
- GitHub branch protection 的管理员包含设置必须关闭，或采用规则集为 `cuihe500` 配置旁路；实现阶段需在 GitHub UI/API 复核实际可用的单用户 bypass 表达能力。

## 设计约束

1. 不将完整源码、依赖目录或构建产物同步到部署主机；仅同步可审计的运行控制面到 `/root/base-workspace/applications/astro/`。
2. 部署脚本仅接收白名单环境名和 SHA256 digest 形式镜像引用，不能接收任意命令、任意 tag 或任意 Compose 文件路径。
3. `astro-deploy` 仅有运行控制面所需最小权限；秘密、OpenResty 证书、数据库管理和系统级配置由 root 的一次性初始化步骤完成，不授予 CI 任意 root shell。
4. 自动部署不得执行 `docker system prune`、`docker compose down`、`--remove-orphans` 或数据库删除操作。
5. 部署成功以容器 health、回环 `/health`、OpenResty 本机 SNI 和公网/入口验证为准；失败时保留已知可用版本，恢复先前 state 记录的 digest。

## 待实施前需要准备的外部条件

- GitHub Environments `test`、`production` 和相应 SSH、GHCR、部署启用变量。
- `astro-deploy` SSH 用户及其受限 authorized_keys 命令入口。
- 测试 K8s kubeconfig 与生产 K8s kubeconfig（后者可后置，但必须通过可用性 gate 才启用生产）。
- 独立测试/生产 MariaDB 数据库、用户、密码和 Docker external 网络。
- 测试/生产 OAuth2 Provider / Client 及准确回调地址。
- Cloudflare DNS 记录、每域名单独证书、OpenResty 两个站点。

## 相关文件

- `.trellis/tasks/08-20-cicd-test-production-deploy/prd.md`
- `/root/base-workspace/AGENTS.md`
- `/root/base-workspace/openresty/README.md`
- `/root/base-workspace/docker/README.md`
- `/root/base-workspace/databases/README.md`
- `/root/base-workspace/applications/README.md`
- `.trellis/spec/backend/auth-guidelines.md`
