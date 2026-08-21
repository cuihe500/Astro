# Astro 镜像与发布流水线实施清单

## 配置安全

- [x] 执行 `trellis-before-dev`，读取配置全部调用者和现有 config test。
- [x] 定义最小显式环境覆盖键和 `ASTRO_RUNTIME_ENV` 语义。
- [x] 实现 strict runtime validate，确保在 DB/K8s 初始化前返回错误。
- [x] 添加配置单元测试：开发保留默认、test/production 合法配置通过、缺失/默认/开发回调/空 kubeconfig 失败。

## 镜像

- [x] 添加 API Dockerfile、多阶段构建、非 root runtime、OCI labels。
- [x] 添加 Web Dockerfile、多阶段构建、SPA fallback 和 OCI labels。
- [x] 添加 `.dockerignore`，确认没有 secret/kubeconfig/host基础设施进入 build context。
- [x] 本地构建或 inspect `linux/arm64` 镜像；确认 API startup 以 runtime env 和只读 kubeconfig 工作。

## GitHub Actions

- [x] 添加 workflow：push main/tag、release published；明确 checkout SHA。
- [x] 运行 Makefile 检查后才配置 Buildx/push GHCR。
- [x] 输出 API/Web digest，使用 commit SHA tag；正式 Release 补可读 tag。
- [x] 使用 `test` Environment SSH 调用受限 deploy contract。
- [x] 使用 release/prerelease/`ASTRO_PRODUCTION_ENABLED` 条件设计 production job；未启用要显式 skipped。
- [x] 加环境 concurrency、防止重复测试部署；限定 permissions。

## 文档与验证

- [x] 更新 README/web README 或新增应用发布说明，描述事件、GitHub Environment 名称、生产启用条件及不归属本仓库的基础设施边界。
- [x] 运行 `make test`、`make lint`、`make build`、`make frontend-check`。
- [x] 静态检查 workflow（actionlint）和 Dockerfile；检查 build context 与镜像层不含 host/control plane/secret。
- [ ] 推送后从 GitHub Actions 验证 main、普通 tag、prerelease Release、正式 Release skip 行为。

## 本地验证记录（2026-08-21）

- `make test`、`make lint`、`make build`、`make frontend-ci`、`make frontend-check`、`make workflow-lint` 全部通过。
- `make docker-build` 成功生成 ARM64 API/Web 镜像；OCI source/revision、healthcheck 和运行命令已 inspect。
- API 入口已用主机测试 env 和只读 kubeconfig 烟测：业务命令 UID `10001`，复制后 kubeconfig 为 `0400`；缺少 runtime env 时拒绝启动。
- Web 深链返回 SPA `index.html`，带模拟 OAuth2 `code/state` 的请求未在 Nginx 日志中出现 query；Gin formatter 单元测试同样通过。

## 回滚

- 配置改动失败：回退到 YAML 开发配置行为；不得弱化 test/production严格校验。
- 镜像/workflow失败：撤回 Astro仓库相关文件；主机已运行版本不受影响。
- 生产 gate异常：保持 `ASTRO_PRODUCTION_ENABLED=false`，不触及主机控制面。
