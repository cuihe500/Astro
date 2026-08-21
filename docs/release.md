# Astro 镜像发布说明

Astro 使用 GitHub Actions 构建两个私有 `linux/arm64` 镜像，并只把不可变 digest 交给主机侧受限部署入口：

- `ghcr.io/cuihe500/astro-api`
- `ghcr.io/cuihe500/astro-web`

镜像以完整 Git commit SHA 标记；GitHub Release 额外使用 Release tag 标记。标签只用于识别，测试和生产部署始终使用 `@sha256:...` 引用。

## 事件与环境

| GitHub 事件 | CI | 测试环境 | 生产环境 |
|---|---|---|---|
| Pull Request 到 `main` | 执行 | 不发布镜像 | 不部署 |
| `main` push | 执行 | 构建并部署 | 不部署 |
| 任意 Git tag push | 执行 | 构建并部署 | 不部署 |
| prerelease Release published | 执行 | 构建并部署 | 不部署 |
| 正式 Release published | 执行 | 先构建、部署并验证 | 仅在生产开关、审批和主机 gate 全部满足时部署 |

正式 Release 在 repository variable `ASTRO_PRODUCTION_ENABLED` 不为 `true` 时会将生产 job 标记为 skipped，不影响测试发布成功。当前生产 Kubernetes 尚未启用，因此该变量必须保持 `false`。

## GitHub 配置契约

`test` 和 `production` GitHub Environments 分别提供同名配置：

| 类型 | 名称 | 用途 |
|---|---|---|
| Environment variable | `ASTRO_DEPLOY_HOST` | 部署主机地址 |
| Environment variable | `ASTRO_DEPLOY_PORT` | SSH 端口 |
| Environment variable | `ASTRO_DEPLOY_USER` | 受限部署用户 |
| Environment secret | `ASTRO_DEPLOY_SSH_KEY` | forced-command 专用私钥 |
| Environment secret | `ASTRO_DEPLOY_KNOWN_HOSTS` | 与主机及端口匹配的固定 SSH host key 记录 |

`production` Environment 还必须配置人工审批。数据库密码、JWT Secret、OAuth2 Client Secret、kubeconfig 和主机 GHCR 拉取凭据不进入 GitHub；它们只由部署主机的运行时配置提供。

主机侧只接受以下远程命令，workflow 不上传源码、Compose 或任意 shell：

```text
deploy <test|production> \
  ghcr.io/cuihe500/astro-api@sha256:<64 位小写十六进制> \
  ghcr.io/cuihe500/astro-web@sha256:<64 位小写十六进制>
```

## API 运行时配置

未设置 `ASTRO_RUNTIME_ENV` 时保留本地开发 YAML 行为。设置为 `test` 或 `production` 时，API 在连接 MariaDB 或 Kubernetes 前要求以下变量全部存在并通过校验：

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

测试和生产分别只接受各自域名的 BytCloud Auth 回调；默认 JWT、空数据库密码、开发回调、无效服务模式以及空或不可读 kubeconfig 都会阻止启动。API 业务进程以固定非 root 用户运行；镜像入口仅用于把宿主机只读挂载的 root-only kubeconfig 复制到容器内受限路径后立即降权。

## 验证与责任边界

发布前的仓库检查统一通过 Makefile：

```bash
make test
make lint
make build
make frontend-ci
make frontend-check
make workflow-lint
make docker-build
```

Astro 仓库只维护应用源码、Dockerfile、workflow 和本文档。Compose、OpenResty、DNS/TLS、数据库、SSH forced command、运行状态及回滚属于 `/root/base-workspace/applications/astro/`，不得复制到本仓库。部署失败和手动回滚均由该主机控制面按已记录 digest 处理。
