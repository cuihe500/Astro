# CI/CD 技术设计

## 1. 目标与边界

本设计将 Astro 的应用发布与主机基础设施拆开：

```text
Astro GitHub 仓库
  ├─ 应用源码、Dockerfile、Actions workflow、发布说明
  ├─ GitHub-hosted runner：检查、构建 ARM64 API/Web 镜像、推送 GHCR
  └─ SSH 强制命令：deploy <environment> <api-digest> <web-digest>
                                      │
                                      ▼
/root/base-workspace（基础设施单独所有）
  ├─ applications/astro/：Compose、部署/回滚入口、状态与主机文档
  ├─ /root/.secrets/astro/<environment>/：运行时秘密
  ├─ OpenResty：域名 → 同环境 loopback API/Web
  └─ MariaDB / Docker network / K8s kubeconfig
```

Astro 仓库不能写入、传输或管理主机基础设施定义。第一次建立主机控制面是独立子任务；以后由 base-workspace 单独维护。

## 2. 组件和接口

### 2.1 应用镜像

| 镜像 | 运行内容 | 对外端口 | 说明 |
|---|---|---:|---|
| `ghcr.io/cuihe500/astro-api` | 编译后的 Go 服务和只读基础配置 | 容器 `8080` | 使用运行时秘密覆盖环境值；挂载专用 kubeconfig |
| `ghcr.io/cuihe500/astro-web` | Vite `dist` 的静态 HTTP 服务 | 容器 `80` | 同源 API，不在构建时写环境 URL |

两个镜像均构建 `linux/arm64`，带 OCI source/revision 标签。Actions 记录 API 与 Web digest；部署只接收 digest，不能只接收标签。

### 2.2 固定主机部署契约

基础设施子任务建立下列稳定接口，应用 workflow 只依赖该接口：

```text
SSH forced command:
/usr/local/lib/astro/deploy <test|production> \
  ghcr.io/cuihe500/astro-api@sha256:<64 hex> \
  ghcr.io/cuihe500/astro-web@sha256:<64 hex>
```

契约行为：

1. 只接受精确环境和两个符合正则的 GHCR SHA256 digest；拒绝其他参数。
2. 读取该环境的固定 Compose 文件、固定 secret 路径和固定状态路径。
3. 先拉取精确镜像；仅重建目标 Compose 项目；不执行 `down`、`--remove-orphans`、清理或数据库 DDL/DML。
4. 等待 API/Web health，检查 API loopback `/health` 和 OpenResty 的本机 SNI。
5. 成功后原子写入当前版本 state；失败时重新部署 state 中的上一对 digest，并返回失败。
6. 不从 Git 获取文件、不接受 shell、不输出秘密。

`astro-deploy` SSH key 的 `authorized_keys` 使用 `command="..."`、禁用转发/Tty/agent forwarding 的单条强制命令；账号没有 root shell。需要 root 的初始 DNS、证书、OpenResty、数据库、网络与 sudo 配置由基础设施子任务一次性完成。

### 2.3 环境资源表

| 资源 | 测试 | 生产 |
|---|---|---|
| 域名 | `astro-test.bytcloud.org` | `astro.bytcloud.org` |
| API/Web loopback | `18080` / `18081` | `18083` / `18084` |
| Compose 项目 | `astro-test` | `astro-production` |
| Docker external network | `app-astro-test-deps` | `app-astro-production-deps` |
| MariaDB | `astro_test` / 专用用户 | `astro_production` / 专用用户 |
| K8s | 当前测试集群 | 独立生产集群，未就绪 |
| OAuth2 | 测试 client 和回调 | 已有生产 client，独立 secret |
| 秘密根目录 | `/root/.secrets/astro/test/` | `/root/.secrets/astro/production/` |
| GitHub Environment | `test` | `production` |

两个 Compose 项目、数据库、用户、网络、JWT/OAuth secret、kubeconfig、状态文件必须完全独立。测试 K8s 和生产 K8s 不能共用，因为应用命名空间含有可能重复的本地用户 ID。

### 2.4 OpenResty 和 Web 路由

每个环境一个单域名证书和站点配置：80/10080 跳转 HTTPS，443/10443 终止 TLS。

```text
/api/      -> API loopback port
/health    -> API loopback port
/*         -> Web loopback port（前端服务器负责 SPA fallback）
```

应用容器只绑定 `127.0.0.1`。新域名 DNS-only CNAME 指向既有 `entry.bytcloud.org`，并依既有 JP/KR→OpenResty 路径发布。OpenResty 变更遵循 base-workspace：备份、`openresty -t`、reload、SNI/入口/公网验证。

## 3. 运行时配置安全改造

当前服务只能部分覆盖 OAuth2 配置，因此应用子任务必须将配置加载改为：基础非敏感 YAML + 明确环境变量覆盖，且在 `ASTRO_RUNTIME_ENV=test|production` 严格校验。

必须从环境注入并校验：

- `ASTRO_RUNTIME_ENV`
- 数据库 host、port、name、user、password
- JWT secret
- OAuth2 client secret、redirect URL；测试还需要独立 client ID 和端点/启用状态（若与生产不同）
- Kubernetes kubeconfig 的容器内固定挂载路径
- server mode、日志路径/级别

生产/测试校验失败条件：变量缺失、JWT 使用仓库默认值 `astro-secret-key`、空数据库密码、空 kubeconfig、默认本地 redirect URL，或 key 值仍为示例。失败发生在连接数据库/K8s 前。开发本地仍可从 `configs/config.yaml` 使用安全的现有行为。

配置应使用与结构字段一一对应的少量 helper，不引入通用反射配置框架。为配置严格校验增加最小单元测试。

## 4. GitHub Actions 事件和数据流

### 4.1 测试发布 workflow

触发：

```yaml
on:
  push:
    branches: [main]
    tags: ['**']
  release:
    types: [published]
```

处理规则：

1. 对 push，解析 `github.sha`；对 release，解析 release tag 指向的 commit SHA。
2. 运行四项 Makefile 校验；失败停止。
3. 生成并推送 API/Web ARM64 镜像，打 commit-SHA 标签；记录两个 digest。
4. 对 `test` GitHub Environment 发起 SSH 强制命令部署，并传入两个 digest。
5. 正式 release 成功测试后交给生产 job；prerelease 与普通 tag 到此结束。

为防止 release 同时产生 tag push 而重复冲突，测试部署使用以环境为范围的并发组，并只允许最新运行继续。部署入口本身也必须幂等。

### 4.2 生产 workflow/job

生产 job 的最小条件：

```text
release event == published
AND release.prerelease == false
AND test deployment succeeded
AND production Environment approval
AND repository variable ASTRO_PRODUCTION_ENABLED == true
AND production kubeconfig/health preflight passed
```

`ASTRO_PRODUCTION_ENABLED` 初始为 `false` 或缺失；此时 job 用 `if` 显式跳过，显示“生产未启用”，Release workflow 仍成功。生产集群就绪后，基础设施和 GitHub 治理子任务更新该开关与 preflight，才允许审批/部署。

### 4.3 GitHub 权限和配置

构建 job 权限：`contents: read`、`packages: write`。部署 jobs 只读取相应 Environment secret。部署主机的 GHCR token 只含 `read:packages`。

建议 secrets/variables 命名：

| GitHub Environment | 配置 | 用途 |
|---|---|---|
| `test` | `ASTRO_DEPLOY_HOST`、`ASTRO_DEPLOY_PORT`、`ASTRO_DEPLOY_USER`、`ASTRO_DEPLOY_SSH_KEY`、`ASTRO_DEPLOY_KNOWN_HOSTS` | 受限测试部署连接与固定 host key |
| `production` | 同名生产连接变量/secret | 受限生产部署连接；Environment 审批保护 |
| repository variable | `ASTRO_PRODUCTION_ENABLED=false` | 控制生产 job 是否可开始 |

不在 Actions secret 保存数据库、JWT、OAuth2、kubeconfig 或 GHCR 拉取凭据；它们仅驻留主机 `/root/.secrets/astro/<environment>/`。

## 5. 发布、回滚和故障边界

- 成功部署把 `{api_digest, web_digest, commit_sha, release_tag?, deployed_at}` 写入环境 state，保留前一成功版本。
- 健康检查失败先恢复前一 state 中的 digest；若不存在前一版本，停止本次目标容器而不触碰数据库。
- 手动回滚由基础设施侧的固定命令指定已记录的版本 state，不允许任意仓库引用。
- GitHub workflow 失败仅代表本次新版本未部署成功，不应隐式修改数据库、DNS、证书或其他项目。
- 自动迁移来自现有 `AutoMigrate`，仅在应用启动时作用于该环境的独立数据库；首次前应由主机控制面做逻辑备份，发布流程不执行数据库删除/恢复。

## 6. GitHub 分支治理

`main` 使用 GitHub 分支保护或 ruleset：普通协作者必须 PR、通过 CI 状态检查且禁止 force push。owner `cuihe500` 有绕过 PR/检查的 direct push 权限；该旁路只影响 GitHub 合并门禁，不影响 `push` 后的构建、测试和测试部署。

## 7. 风险、权衡与延后项

| 风险 | 处理 | 延后 |
|---|---|---|
| 主机与生产共处 | 两套完全隔离的运行资源 | 独立生产主机/HA |
| Release/tag 触发重叠 | 环境并发组 + 幂等部署 | 复杂 release 编排平台 |
| 生产 K8s 未就绪 | 生产 job 明确跳过 | 生产集群建设 |
| host SSH 权限扩大 | 单账号、强制命令、digest 白名单 | 更复杂的 mTLS/零信任部署 |
| 镜像架构错误 | CI 强制 `linux/arm64` | 多架构发布 |
| 运行时秘密缺失 | 启动前严格验证 | 外部 Secret Manager |
| 前端深链接 404 | Web 容器 SPA fallback | SSR/边缘路由 |

## 8. 兼容性与回滚

- 本地开发继续通过 `make run`、`make frontend-run` 运行；不要求 Docker 或生产 secret。
- 后端 API 路径、响应格式和前端请求接口保持不变。
- OAuth2 生产回调保持现有 `astro.bytcloud.org` URL；测试新增独立回调，不改变生产 provider。
- 如果 CI/CD 需要撤销，停用 workflow/部署 key 后不会影响已运行的主机 Compose 版本；基础设施回滚在 base-workspace 文档中独立执行。

## 9. 首次测试发布批次

首次真实发布初始目标为 `d465a2d2f08df2a9c1f1fb3d55cc643afb9694db`，首轮 CI 暴露 workflow 静态检查问题后，以最小修复提交 `17f6d72e1de0754cfc083f7a12d74e9dfc036de4` 完成部署。发布使用已有 `main push` 路径，commit SHA 即测试版本号。发布前只补齐 `test` Environment、forced-command 专用部署 key 和固定 host key；生产开关保持关闭，不为首次可见效果增加 tag、Release 或生产治理依赖。

执行顺序为：配置受限连接 → 推送目标提交 → GitHub CI → 推送两个 ARM64 镜像 → 传递两个 digest → 测试 Compose 更新与健康检查 → 公网 Web/API/OAuth2 烟测 → 临时应用全生命周期验证。任一步失败先保留 Actions 与主机诊断证据；主机部署入口按已有 state 恢复上一成功 digest，禁止通过删除数据库或放宽 SSH 权限绕过失败。

首个成功版本没有可供恢复的历史版本，因此本批次只验证失败输入不会破坏成功版本；自动恢复上一 digest 的真实演练延后到存在第二个可部署版本时进行。
