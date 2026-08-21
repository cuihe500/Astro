# 发布运行时与主机部署规范

> 本文定义 Astro 应用镜像、GitHub Actions 与独立主机控制面之间的可执行契约。主机 Compose、OpenResty、部署脚本、状态和秘密只属于 `/root/base-workspace`，不得复制到 Astro Git 仓库。

## 场景：Digest-only 测试发布

### 1. Scope / Trigger

- 触发：新增或修改运行时环境变量、API/Web 镜像、GitHub 发布 workflow、部署 SSH 调用、测试/生产发布条件时。
- 目标：GitHub-hosted Actions 只构建并传递不可变镜像 digest；主机只拉取镜像并执行固定部署入口，不接收源码、Compose 路径、任意 shell 或秘密。
- 主机权威控制面：`/root/base-workspace/applications/astro/README.md`。仓库仅记录本规范、应用代码、Dockerfile、workflow 与应用发布说明。

### 2. Signatures

唯一远程部署命令：

```text
deploy <test|production> \
  ghcr.io/cuihe500/astro-api@sha256:<64 lowercase hex> \
  ghcr.io/cuihe500/astro-web@sha256:<64 lowercase hex>
```

- 环境只能是 `test` 或 `production`。
- API digest 正则：`^ghcr\.io/cuihe500/astro-api@sha256:[0-9a-f]{64}$`。
- Web digest 正则：`^ghcr\.io/cuihe500/astro-web@sha256:[0-9a-f]{64}$`。
- CI 通过 `astro-deploy` 的 SSH forced command 调用该签名；不能传入 `latest`、tag、URL、Compose 文件路径、额外参数或 shell 片段。
- 手动回滚只允许主机侧 `rollback <test|production>` 重放 state 中的 `previous` digest，不能接收任意镜像引用。

### 3. Contracts

#### 环境隔离

| 项目 | test | production |
|---|---|---|
| Compose 项目 | `astro-test` | `astro-production` |
| API/Web loopback | `127.0.0.1:18080/18081` | `127.0.0.1:18083/18084` |
| MariaDB | `astro_test` / `astro_test` 用户 | `astro_production` / `astro_production` 用户 |
| MariaDB 依赖网络 | `app-astro-test-deps` | `app-astro-production-deps` |
| Runtime secrets | `/root/.secrets/astro/test/` | `/root/.secrets/astro/production/` |
| State | `state/test.json` | `state/production.json` |

测试 API 还必须连接现有 Docker `kind` 网络；其 kubeconfig 使用 `astro-test-control-plane:6443`，以便从容器访问本地 Kind API。Web 只能连接测试 MariaDB 依赖网络。生产不得连接测试 `kind` 网络。

#### 运行时环境变量

后续 API 镜像必须在 `ASTRO_RUNTIME_ENV=test|production` 时使用并严格校验以下宿主机环境变量，而不能回退到仓库开发默认值：

```text
ASTRO_RUNTIME_ENV
ASTRO_SERVER_PORT
ASTRO_SERVER_MODE
ASTRO_DATABASE_HOST
ASTRO_DATABASE_PORT
ASTRO_DATABASE_USER
ASTRO_DATABASE_PASSWORD
ASTRO_DATABASE_DBNAME
ASTRO_DATABASE_CHARSET
ASTRO_JWT_SECRET
ASTRO_KUBERNETES_KUBECONFIG
ASTRO_LOG_LEVEL
ASTRO_LOG_FILE
ASTRO_OAUTH2_AUTHENTIK_CLIENT_ID
ASTRO_OAUTH2_AUTHENTIK_CLIENT_SECRET
ASTRO_OAUTH2_AUTHENTIK_REDIRECT_URL
```

- API 容器的 kubeconfig 固定只读挂载到 `/run/secrets/astro-kubeconfig`。
- API 镜像入口只接受 `ASTRO_RUNTIME_ENV=test|production`。入口以 root 读取 `/run/secrets/astro-kubeconfig`，复制为 `/run/astro/kubeconfig`（`astro:astro 0400`），覆盖容器内 `ASTRO_KUBERNETES_KUBECONFIG` 后立即通过 `su-exec` 以固定 UID/GID `10001` 执行业务进程；不得对 bind mount 执行 `chmod`/`chown`，宿主机 `/root/.secrets` 始终保持 `root:root 600`。本地开发使用 `make run`，不把缺少运行时 gate 的 API 镜像当作开发入口。
- API 和 Web 镜像都必须提供 Compose healthcheck 需要的 `wget` 或 `curl`；API `/health` 必须返回成功，Web `/` 必须可访问。
- Gin 和 Web Nginx 访问日志只能记录 URL path：Go 使用 `Request.URL.Path`，Nginx 使用 `$uri`；禁止记录 `RequestURI`、`RawQuery`、Gin 默认 `LogFormatterParams.Path` 或 Nginx `$request`，避免 OAuth2 callback 的 `code` / `state` 进入日志。
- Host state 只保存 `{current, previous}` 两对 API/Web digest 和时间，不保存 tag、密码、Token 或 kubeconfig 内容。

#### 生产 gate

生产在以下条件全部满足前必须拒绝部署，而不是尝试 pull 或启动容器：

```text
/root/.secrets/astro/production/enabled == true
AND /root/.secrets/astro/production/kubeconfig is non-empty
AND GitHub production Environment 已审批
```

当前生产开关为 `false`。正式 Release workflow 必须将此情形显示为 skipped/“生产未启用”，不能报作测试发布失败。

### 4. Validation & Error Matrix

| 条件 | 必须行为 |
|---|---|
| 参数数目、环境名或任一 digest 不匹配 | forced command 和 deploy 都拒绝；不运行 Docker 命令 |
| SSH 原始命令包含 shell、tag、URL、额外参数 | 拒绝；不得获得 shell、PTY、端口/agent/X11 转发 |
| production 未启用或 kubeconfig 缺失 | 在 pull 前拒绝 production |
| runtime env 缺失、开发默认 JWT、空数据库密码、无效 OAuth redirect 或不可读 kubeconfig | API 在连接 MariaDB/Kubernetes 前启动失败 |
| API 镜像缺少/使用其他 `ASTRO_RUNTIME_ENV`，或固定 kubeconfig mount 不可读/为空 | 镜像入口在业务进程启动前失败；不得降级到开发 YAML |
| OAuth2 callback 带 `code` / `state` query | API/Web 访问日志只出现 callback path，不出现 query 名称或值 |
| 镜像 pull 失败 | 先前 state 存在时恢复它；首次 state 为空时仅停止目标 API/Web 服务，不执行 `compose down` |
| `up --wait`、loopback `/health`、SNI `/health` 任一失败 | 恢复前一成功 digest；state 不写入失败版本 |
| state 损坏或 digest 不合法 | 拒绝，不猜测回滚目标 |
| 生产 K8s 尚未就绪 | Release production job skipped；不能伪造 kubeconfig 或启动生产容器 |

部署入口禁止 Git 拉取、`docker compose down`、`--remove-orphans`、Docker prune 和任何数据库 DDL/DML。

### 5. Good / Base / Bad Cases

- **Good**：workflow 将两个精确 ARM64 GHCR digest 传给 `deploy test`；主机拉取后仅更新 `api` 与 `web`，入口复制 root-only kubeconfig 并降权，API 通过 Kind 测试集群和独立 MariaDB 工作，随后 state 原子更新。
- **Base**：第一个测试镜像无法拉取。入口拒绝并只停止本次目标服务，`state/test.json` 仍为 `{current:null,previous:null}`，其他容器、数据库和网络不变。
- **Bad**：workflow 传 `astro-api:latest`、使用任意 SSH 命令、生产复用测试 kubeconfig，或为让非 root 进程读取 secret 而修改宿主机文件权限。它们会破坏不可变发布、环境隔离或 root-only secret 边界，禁止。

### 6. Tests Required

发布相关改动至少验证：

1. `make test`、`make lint`、`make build`、`make frontend-check` 在构建镜像前通过。
2. API 配置测试覆盖 test/production 的完整合法环境，以及缺变量、默认 JWT、空密码、localhost redirect、空/不可读 kubeconfig 的拒绝路径。
3. Docker 构建验证 `linux/arm64`、OCI source/revision 标签、无秘密进入 build context，且 API/Web healthcheck 依赖存在；API 入口必须拒绝缺失 runtime env，并断言降权后 UID 为 `10001`、复制后的 kubeconfig 为 `0400`。
4. workflow/主机契约测试：非法环境、tag、额外参数和不可拉取 digest 都被拒绝；首次失败不写 state、不影响数据库。
5. 首次有效测试 digest 进行集成验证：容器 health、`127.0.0.1:18080/health`、`127.0.0.1:18081/`、本机 SNI、JP/KR 与公网测试域名、OAuth 浏览器回调。
6. 有成功 state 后使用故障 digest 演练自动回滚，并断言前一 digest 重新 healthy、state 没有记录故障 digest。
7. 生产开关为 false 时，正式 Release 的 production job 与主机 deploy 均明确拒绝/跳过。
8. 使用带测试 `code/state` 的 callback URL 请求 API/Web，断言 Gin 与 Nginx 访问日志均不含 query 名称和值。

### 7. Wrong vs Correct

#### Wrong

```yaml
# workflow：可变 tag、且允许任意远程命令。
run: ssh "$HOST" "docker compose pull && docker compose up -d"
```

#### Correct

```yaml
# workflow：只调用固定签名；主机验证 digest 后决定 Compose 路径和动作。
run: >-
  ssh "$ASTRO_DEPLOY_USER@$ASTRO_DEPLOY_HOST"
  "deploy test ${{ needs.build.outputs.api_digest }} ${{ needs.build.outputs.web_digest }}"
```

**原因**：不可变 digest 能将运行版本与构建产物精确对应；强制命令把 CI 权限限制到一个可审计的部署动作，并避免源码、秘密和主机控制面流入 workflow。

#### Wrong：记录完整请求

```go
// Gin 默认 Path 已拼接 RawQuery，会泄露 OAuth2 code/state。
fmt.Fprint(out, params.Path)
```

```nginx
access_log /var/log/nginx/access.log combined; # $request 含 query
```

#### Correct：只记录 path

```go
fmt.Fprint(out, params.Request.URL.Path)
```

```nginx
log_format astro_without_query '"$request_method $uri $server_protocol" $status';
```

**原因**：OAuth2 callback 的 query 是一次性凭据；访问日志只需 path、方法和状态即可诊断路由，不应复制认证材料。
