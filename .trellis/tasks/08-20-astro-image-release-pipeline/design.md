# Astro 镜像与发布流水线设计

## 运行时配置

在 `pkg/config` 保留 YAML 基础配置。仅当 `ASTRO_RUNTIME_ENV` 为 `test` 或 `production` 时，明确覆盖 database、JWT、OAuth2、server、log 和 kubeconfig 字段，并运行 `ValidateRuntime`。开发环境不启用严格 gate。

校验至少拒绝：空数据库密码、默认 JWT、空/示例 OAuth client secret、localhost OAuth redirect、空 kubeconfig、无效 server mode。失败发生在 `main` 初始化 DB/K8s 前。

## 镜像

- API Dockerfile：Go 构建阶段 → 小型运行阶段；复制 binary、默认 `configs/config.yaml`、必要 docs；以非 root 用户执行；不包含 secret/kubeconfig。
- Web Dockerfile：Node 构建 Vite → Nginx/Caddy 等小型静态服务；配置 history fallback 到 `index.html`；不含 API URL 与运行时 secret。
- `.dockerignore` 排除 `.git`、`.trellis`、`bin`、`web/node_modules`、日志、本地配置、覆盖文件和秘密文件。

## Workflow

统一 workflow 响应 push 和 release published：

1. checkout 目标 SHA；对 release 显式解析 tag commit。
2. 运行四个 Makefile目标。
3. 使用 Buildx 构建/push `linux/arm64` API/Web，输出 digest；以 SHA 标记，正式 Release 额外使用 tag。
4. `test` Environment job 以 SSH 调用固定强制命令，严格传 `<test> <api digest> <web digest>`。
5. `production` job 仅在 release published、非 prerelease、test成功和 `ASTRO_PRODUCTION_ENABLED == 'true'` 时运行，使用 production Environment；其他情形通过 job if 显示 skipped。

测试部署用 concurrency group 防止 tag/release 重叠。workflow 仅有 `contents: read`、`packages: write`；SSH secret 每个 Environment 隔离。

## 外部部署接口

workflow 不知道 Compose/OpenResty/数据库/DNS 的文件路径。唯一接口：

```text
ssh $ASTRO_DEPLOY_USER@$ASTRO_DEPLOY_HOST \
  deploy test ghcr.io/cuihe500/astro-api@sha256:... ghcr.io/cuihe500/astro-web@sha256:...
```

主机强制命令验证请求并完成实际操作。

## 兼容性

本地 `make run`、`make frontend-run` 与 API URL 语义不变。没有 host/CI secrets 时，本地开发不受严格 production/test gate 影响。
