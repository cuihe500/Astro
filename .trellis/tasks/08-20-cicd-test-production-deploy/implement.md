# CI/CD 实施清单

## 前置说明

- 父任务只协调三个子任务，不直接承载应用或基础设施实现。
- 先完成主机控制面，确认固定部署契约后再完成 Astro 仓库工作流；GitHub 治理可与两者并行，但必须在首次自动部署前完成所需 Environment 配置。
- 所有基础设施文件只写入 `/root/base-workspace`，绝不复制至 Astro 仓库。

## 0. 共享契约冻结

- [ ] 记录并确认以下固定接口：
  - 环境：`test`、`production`
  - SSH 命令：`deploy <environment> <api-digest> <web-digest>`
  - 两个 GHCR 仓库：`astro-api`、`astro-web`
  - 测试端口：`18080`、`18081`；生产端口：`18083`、`18084`
  - GitHub Environments：`test`、`production`
  - repository variable：`ASTRO_PRODUCTION_ENABLED=false`
- [ ] 确认应用工作流不携带数据库密码、JWT、OAuth2 secret、kubeconfig 或 GHCR 拉取凭据。

## 1. 子任务：Astro 主机部署控制面

路径：`08-20-astro-host-control-plane`。

- [ ] 按 `/root/base-workspace/AGENTS.md` 创建 `/root/base-workspace/applications/astro/README.md`，并在应用索引中登记 Astro。
- [ ] 创建测试与生产的独立 external Docker 网络、MariaDB 数据库/用户、root-only密码文件与运行时环境文件；数据库操作前做可恢复备份并验证。
- [ ] 创建测试 K8s kubeconfig 的 root-only挂载路径；为生产预留独立路径且保持生产部署禁用。
- [ ] 在 base-workspace 创建测试与生产的固定 Compose 定义，API/Web 均仅绑定对应 loopback 端口，并使用 API `/health` 与 Web HTTP healthcheck。
- [ ] 创建 `astro-deploy` 最小权限账号、GHCR read-only凭据、SSH 强制命令、digest 参数校验与禁止交互式 shell 的 authorized_keys 配置。
- [ ] 实现部署状态、前一版本记录、健康检查失败恢复和受限手动回滚入口；禁止 `down`、清理、数据库删除和任意命令执行。
- [ ] 建立 `astro-test.bytcloud.org` / `astro.bytcloud.org` 的 Cloudflare DNS、单域名 Certbot DNS-01 证书和 OpenResty 站点；先备份，`openresty -t` 后 reload。
- [ ] 配置 `/api/` 与 `/health` 到 API、其他路径到 Web 的反代及 SPA fallback。
- [ ] 配置独立测试 OAuth2 client 和回调；保留生产 provider，分别保存 secret。
- [ ] 记录部署、回滚、日志、备份、DNS/TLS、数据库和测试/生产 K8s 限制；验证 loopback、SNI、JP、KR 和公网路径。

**完成门槛**：使用一个临时/可验证 image digest 可调用固定部署入口，测试服务在不暴露容器公网端口时通过本机和 OpenResty健康检查；生产入口存在但由于生产 K8s gate 保持禁用。

## 2. 子任务：Astro 镜像与发布流水线

路径：`08-20-astro-image-release-pipeline`。依赖第 1 项的 SSH 命令、环境名和端口契约。

- [ ] 使用 `trellis-before-dev` 加载后端规范后，检查现有配置调用点与所有调用者。
- [ ] 最小化扩展 `pkg/config`：为 `ASTRO_RUNTIME_ENV=test|production` 加明确环境变量覆盖和启动前关键配置校验；保留开发默认行为。
- [ ] 为运行时配置严格校验添加最小单元测试；确保不记录/暴露秘密。
- [ ] 添加 API 多阶段 Dockerfile：以最小运行镜像运行编译结果、固定工作目录、提供配置路径、添加 OCI metadata，不携带 kubeconfig/secret。
- [ ] 添加 Web 多阶段 Dockerfile：构建 Vite 产物并由最小静态服务器提供 SPA fallback；同源 API 不编译环境 URL。
- [ ] 添加最小 `.dockerignore`，排除 Git、Trellis、node_modules、构建产物、日志和本地秘密。
- [ ] 添加 GitHub Actions workflow：main/tag/release 触发、Makefile 检查、Buildx `linux/arm64`、GHCR digest 输出、test 环境受限 SSH 部署、release正式生产候选条件和跳过逻辑。
- [ ] workflow 配置环境级并发，避免 tag push/release 发布重叠部署；部署调用只传两个 digest。
- [ ] 更新 Astro 项目发布说明：说明事件语义、Release/prerelease行为、所需 GitHub Environment 键名、生产 enable前置条件；不写主机脚本或秘密。

**完成门槛**：workflow 可通过 YAML/actionlint 等静态检查；本地使用现有 Makefile 校验，镜像可构建/检查 ARM64；仓库不含任何主机控制面或秘密。

## 3. 子任务：GitHub 发布治理

路径：`08-20-astro-github-release-governance`。与第 1、2 项同步所需名称。

- [ ] 创建 `test` Environment 并设置受限部署连接信息；创建 `production` Environment 并设置对应连接信息和 required reviewer 审批。
- [ ] 设定 repository variable `ASTRO_PRODUCTION_ENABLED=false`；明确生产 K8s 就绪后才由受权操作启用。
- [ ] 配置 `main` 分支保护/规则集：普通协作者 PR + CI required checks、禁止 force push；`cuihe500` 可绕过 PR/检查而直接 push。
- [ ] 复核 owner direct push 仍触发 workflow，正式 release 的 prerelease false 条件进入 production job，未启用时显示 skipped。
- [ ] 在 GitHub 配置完成后，核对权限最小化：workflow `packages: write`，主机 token `read:packages`，没有把运行时秘密放入 GitHub。

**完成门槛**：GitHub UI/API 可证明 Environments、分支保护和生产开关符合设计；不改变生产部署启用状态。

## 4. 集成和验证

- [x] 用一个真实 main commit 完成：CI 检查 → ARM64镜像推送 → digest 测试部署 → API/Web health → `astro-test.bytcloud.org` 浏览器/OAuth2 烟测。
- [ ] 推一个非 Release tag，确认它只更新测试环境。
- [ ] 创建 prerelease GitHub Release，确认它只更新测试环境。
- [ ] 创建/模拟正式 GitHub Release，确认测试通过后 production job 因 `ASTRO_PRODUCTION_ENABLED=false` 明确 skipped，而非 workflow 失败。
- [ ] 使测试部署健康检查失败一次，确认部署入口保留/恢复上一可用版本且数据库未被删除。
- [x] 检查 `git status`、镜像 metadata、base-workspace 文档、OpenResty配置、Docker 端口绑定、数据库网络、GHCR包可见性和 GitHub Actions日志。
- [x] 最终通过 `make test`、`make lint`、`make build`、`make frontend-check`，并按 base-workspace规则完成运行时验证。

### 4.1 首次测试发布：`d465a2d` → `17f6d72`

- [x] 复核目标 SHA、远端差异和工作区；只推送已提交的 `main`，不携带本地未提交文件。
- [x] 生成并安装 forced-command 专用部署 key；创建 GitHub `test` Environment，只录入固定 host/port/user、私钥和 host key，不上传应用运行时秘密。
- [x] 保持 `ASTRO_PRODUCTION_ENABLED=false` 或缺失；不启动生产 Compose，不创建 tag/Release。
- [x] 首轮 `d465a2d2f08df2a9c1f1fb3d55cc643afb9694db` 在 workflow 静态检查失败；最小修复后以 `17f6d72e1de0754cfc083f7a12d74e9dfc036de4` 完成 `CI 校验`、ARM64 镜像发布和测试部署，记录运行 URL、实际 SHA 与 API/Web digest。
- [x] 验证测试 API/Web 容器健康、只监听 `127.0.0.1:18080/18081`，并验证本机 SNI、JP/KR 入口及公网 `https://astro-test.bytcloud.org`。
- [x] 通过测试域名验证 Web、`/health`、OAuth2 登录入口和核心应用创建/查询/启动/停止/重启/日志/删除；清理临时应用并记录无法通过公开 API 清理的测试数据。
- [x] 对成功版本提交非法环境/tag/digest 请求，确认受限入口拒绝且现有服务、state 和数据库不受影响；第二个有效版本出现前不伪造“上一 digest 自动恢复”结论。
- [x] 将真实结果同步到对应子任务清单和 `/root/base-workspace/applications/astro/README.md`，不得记录 secret 正文。

## 验证命令

Astro 仓库只使用现有 Makefile 目标：

```bash
make test
make lint
make build
make frontend-check
```

基础设施子任务遵循其权威文档，至少包括：

```bash
docker compose config --quiet
docker compose ps
/usr/local/openresty/bin/openresty -t
systemctl is-active openresty
```

实际 DNS、证书、OpenResty、Compose、MariaDB和部署脚本命令由 `/root/base-workspace/applications/astro/README.md` 定义，不复制到 Astro 仓库。

## 风险点与回滚点

| 风险点 | 回滚/保护措施 |
|---|---|
| 配置严格校验阻止启动 | 保留开发环境行为；先添加配置测试；生产/测试只用 explicit secret 文件 |
| 新镜像无法启动 | host state 回到前一 digest；不修改数据库 |
| OpenResty/DNS/证书变更影响其他站点 | base-workspace 先备份、语法检查、只改独立站点文件、平滑 reload |
| release/tag 事件重复 | test 并发组 + 幂等 digest 部署 |
| 生产 K8s 尚未准备 | `ASTRO_PRODUCTION_ENABLED=false`，job 显式 skipped |
| CI SSH 权限扩大 | 强制命令、无 shell、只允许 environment+digest 参数 |

## 任务启动前复核

- [x] 父 PRD、design 和 implement 已无未决问题。
- [x] 三个子任务均已拥有独立 PRD、design、implement 和 JSONL context。
- [x] 第 1 子任务作为下一个执行目标；第 2 子任务在其部署契约明确后执行；第 3 子任务可并行配置但不得提前启用生产。
- [x] 用户已审阅本实施清单并在后续消息中明确批准开始实施。
