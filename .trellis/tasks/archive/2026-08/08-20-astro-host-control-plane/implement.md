# Astro 主机部署控制面实施清单

## 实施顺序

- [x] 阅读 `/root/base-workspace/AGENTS.md`、`applications/README.md`、`docker/README.md`、`databases/README.md`、`openresty/README.md`、`firewall/README.md` 与 `backup/README.md`。
- [x] 只读复核端口、Docker 网络、MariaDB、OpenResty、DNS、证书、Authentik Provider 和现有秘密路径；不得回显秘密。
- [x] 创建此次变更的 root-only UTC 备份，包含将修改的 base-workspace 文档、MariaDB/OpenResty/证书/DNS 前态和目标缺失标记；生成并验证 manifest。

## 建立主机控制面

- [x] 创建 `/root/base-workspace/applications/astro/` 与 README，更新应用索引。
- [x] 创建 `app-astro-test-deps`、`app-astro-production-deps` external Docker 网络，并向 MariaDB Compose 显式添加稳定 `mariadb` 别名。
- [x] 在 MariaDB 创建 `astro_test` / `astro_production` 及最小授权用户；生成 root-only 各环境密码和 runtime env 文件，不输出内容。
- [x] 创建测试 kubeconfig 秘密路径；为生产创建空/不可用 gate 路径与禁用标记，不创建或假装生产集群存在。
- [x] 创建 base-workspace 内的 test/production Compose 定义：固定项目名、loopback端口、healthcheck、secret/kubeconfig read-only挂载、明确 external network、无数据库容器。
- [x] 复核并复用现有 root-only GHCR host credential；登录配置不进入 Astro 仓库或日志。
- [x] 创建 `astro-deploy`、SSH key forced command、受限 deploy/rollback 脚本、版本 state 路径与权限。
- [x] 实现 digest 验证、pull、目标项目 update、health/SNI检查、state原子写入和失败恢复。

## 公网入口和身份认证

- [x] 使用 Cloudflare DNS-01 建立 `astro-test.bytcloud.org` 与 `astro.bytcloud.org` 的独立证书；创建 DNS-only CNAME 到 `entry.bytcloud.org`。
- [x] 写两个独立 OpenResty站点文件与 symlink：HTTP redirect、PROXY listener、TLS、API/web route、SPA fallback；先语法检查后 reload。
- [x] 复核测试 Astro OAuth client/application、严格测试回调和独立 secret；保留生产对象和 secret 边界。

## 验证

- [x] GitHub Run `32462154081` 的提交 SHA、API/Web digest、OCI revision 与主机 test state 一致；两镜像均为 ARM64，两容器 healthy 且仅绑定 `127.0.0.1:18080/18081`，生产没有容器或监听。
- [x] loopback、本机 SNI、JP/KR 入口和公网测试域名的 Web `/`、SPA 深链、API `/health`、统一错误响应和 TLS 均通过；公网 OAuth2 登录 URL 使用测试 client 与严格测试回调。
- [x] 经公网 API 完成临时账号注册/登录及单副本应用创建、列表、详情、running、日志、停止、启动、重启和删除；临时应用已删除，因无公开删除接口仅记录临时本地用户、OAuth 验证用户与空命名空间残留。
- [x] 用合法格式但不可拉取 digest 验证入口安全失败；用非法环境/tag/额外参数验证拒绝；不得影响当前容器或数据库。
- [x] 从 `jp-aws-entry` 外部链路提交浮动 tag，forced command 在部署前拒绝；前后 state、容器和公网健康不变。
- [~] 首次成功 state 的 `previous` 为 `null`，未伪造前一 digest 或自动回滚结论；待下一次成功版本建立 `previous` 后再演练真实自动恢复。
- [x] 验证生产入口配置存在且 production deploy 受 enabled/kubeconfig gate 拒绝。
- [x] 更新 base-workspace README 记录所有验证及回滚步骤。

## 关键验证命令

遵循 base-workspace README，至少：

```bash
docker compose config --quiet
docker compose ps
/usr/local/openresty/bin/openresty -t
systemctl is-active openresty
ss -lntp
```

## 回滚点

- OpenResty/DNS/证书：从本任务 UTC 备份恢复，`openresty -t` 后 reload。
- 数据库/网络：先恢复受影响环境库/用户或仅移除新增资源，不操作其他业务数据库。
- 控制面：恢复 base-workspace 文件并停用 Astro Compose 项目，不执行全局 Docker 清理。
- SSH：删除新增 `astro-deploy` authorized key/账号权限，保留审计备份。
