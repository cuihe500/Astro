# 建立 Astro 主机部署控制面

## Goal

仅在 `/root/base-workspace` 首次建立 Astro 测试/生产运行控制面，使受限 CI 能用 API/Web image digest 部署测试环境，并为生产保留隔离资源与明确禁用 gate。

## Requirements

- 创建 `/root/base-workspace/applications/astro/` 的应用文档、Compose 定义、受限部署/回滚入口和版本状态。
- 测试和生产必须隔离 Docker 网络、MariaDB 数据库/用户、秘密、kubeconfig、Compose 项目、端口和状态。
- 测试使用现有测试 K8s；生产没有独立 K8s 时不部署生产。
- 建立 `astro-test.bytcloud.org` 和 `astro.bytcloud.org` 的 DNS、单域名证书、OpenResty 路由；仅 loopback 暴露 API/Web。
- 受限入口只接受 `test|production` 和精确 GHCR API/Web SHA256 digest，能够健康检查、记录版本并在失败时恢复前一个版本。
- 所有秘密只进入 `/root/.secrets/astro/<environment>/`；备份只进入 `/root/base-workspace/backup/`。

## Acceptance Criteria

- [ ] base-workspace 应用文档完整记录路径、网络、端口、秘密、DNS/TLS、部署、回滚、验证和生产 K8s 限制。
- [ ] 测试 API/Web 仅监听 `127.0.0.1:18080/18081`，生产预留 `127.0.0.1:18083/18084`。
- [ ] 每个环境使用独立 MariaDB 数据库、用户、密码、Docker external 网络、Compose 项目和状态文件。
- [ ] 测试站点经 OpenResty/SNI/公网链路验证；生产站点已建立但不启动生产 API/Web，直到独立生产 K8s 就绪。
- [ ] 受限 SSH 部署只接受白名单 digest，禁止任意 shell/标签/源码拉取，并可恢复上一成功版本。
- [ ] 主机控制面不复制进 Astro Git 仓库。

## Out of Scope

- 在 Astro 仓库添加 Dockerfile、GitHub Actions 或运行时配置代码。
- 创建独立生产 Kubernetes 集群或启用生产部署。
- 日后从 Astro CI 同步/覆盖 Compose、OpenResty 或主机脚本。

## Dependencies

- 此子任务先于 `08-20-astro-image-release-pipeline` 的首次自动测试部署。
- 产出固定部署命令、环境名、端口和 digest 形状契约供应用 workflow 使用。
