# Astro 主机部署控制面设计

## 固定目录和所有权

```text
/root/base-workspace/applications/astro/
  README.md
  compose/test.yaml
  compose/production.yaml
  bin/deploy
  bin/rollback
  state/test.json
  state/production.json

/root/.secrets/astro/
  registry.env
  test/runtime.env
  test/kubeconfig
  production/runtime.env
  production/kubeconfig
```

控制面仅属于 base-workspace。Astro Git 仓库和 GitHub Actions 不写这些文件。

## 服务模型

每个环境固定两个服务：API 与 Web。测试端口 API/Web 为 `127.0.0.1:18080/18081`，生产为 `127.0.0.1:18083/18084`。API 与 Web 只加入该环境的 external Docker network；数据库容器同时加入该网络并以 `mariadb` 提供别名。

Compose 中 image 值由受限 `deploy` 读取当前请求 digest 后临时/环境变量注入；Compose 不允许通过浮动 tag 运行。API 配置与 kubeconfig 都只读挂载，Web 无秘密挂载。

## 部署入口

部署参数：

```text
deploy <test|production> <api ghcr digest> <web ghcr digest>
```

执行顺序：验证参数 → 读取环境固定文件 → `docker pull` 两个 digest → `docker compose up -d` 仅该项目 → 等 health → 回环 API `/health` → OpenResty 本机 SNI → 原子记录 state。失败时读取 state 上一对 digest 并重新部署；无上一状态时停止新容器，保留数据。

拒绝：额外参数、非 `ghcr.io/cuihe500/astro-{api,web}@sha256:<64hex>`、`latest`、URL、shell 元字符、未知环境。不得运行 Git、`docker compose down`、`--remove-orphans`、prune 或数据库删除。

SSH key 使用 forced command 调用该入口，禁用 pty、端口/agent/X11 转发。部署账号不具有 root shell；root 初始化完成后仅让入口拥有其所需的最小 Docker 权限和固定文件读权限。

## OpenResty

两个独立站点，均使用既有 80/10080 HTTPS 跳转、443/10443 TLS 与 proxy protocol 结构：

- `/api/`、`/health` 反代同环境 API；
- `/` 反代同环境 Web；
- 每域名单独 Certbot DNS-01 证书；
- DNS-only CNAME → `entry.bytcloud.org`，走既有 JP/KR 入口。

执行前按 base-workspace 规则备份；`openresty -t` 成功后 reload。生产站点配置/证书可建立，但生产 Compose 不能在生产 kubeconfig gate 未满足时启动。

## 数据、秘密和 K8s

测试和生产各自数据库、用户、密码文件、网络、JWT secret、OAuth client/secret、runtime env、kubeconfig 和 state。测试 OAuth2 需独立 client 与 `astro-test` 严格回调。生产运行时仅在独立生产 K8s kubeconfig 可用时启用。

数据库创建、用户授权和网络连接均先创建 base-workspace 规定的备份；不操作其他业务库。

## 验证与回滚

- `docker compose config --quiet`；容器 health；loopback API/Web；OpenResty SNI；JP/KR/公网验证。
- 记录 Docker image digest、网络、端口、数据库与证书状态。
- 单次部署失败自动恢复上一 state；DNS/OpenResty/数据库变更采用各自已验证备份恢复，先检查语法/数据再 reload。
