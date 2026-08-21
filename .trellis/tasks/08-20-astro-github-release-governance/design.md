# Astro GitHub 发布治理设计

## Environments

| Environment | 内容 | 保护 |
|---|---|---|
| `test` | variables：`ASTRO_DEPLOY_HOST`、`ASTRO_DEPLOY_PORT`、`ASTRO_DEPLOY_USER`；secrets：`ASTRO_DEPLOY_SSH_KEY`、`ASTRO_DEPLOY_KNOWN_HOSTS` | 无人工审批，允许 main/tag/Release 的测试 job |
| `production` | 同名生产部署连接信息 | required reviewer 审批；只允许正式 Release production job |

变量/secret 名称在两个环境保持一致，值独立。`ASTRO_DEPLOY_KNOWN_HOSTS` 保存与主机和端口匹配的固定 host key 记录，workflow 必须启用严格校验，禁止运行时 `ssh-keyscan`。仅保存 SSH 连接信息，不保存运行时秘密或 GHCR拉取 token。

## 生产开关

repository variable `ASTRO_PRODUCTION_ENABLED` 初值 `false`。workflow 的 production job 必须在 `true` 时才创建；false/缺失表示 job skipped。将来只有生产 K8s、kubeconfig、主机控制面 preflight 和审批策略均准备好后才变更为 true。

## main 保护

使用 GitHub branch protection 或 ruleset，目标 `main`：

- 普通协作者需 pull request 和 workflow 定义的 CI 状态检查；
- force push 禁止；
- 管理员限制不包含 owner，或 ruleset 为 `cuihe500` 定义 bypass，允许其直接 push；
- 任何 direct push 仍触发 push workflow，旁路不跳过测试部署。

由于当前仓库只有 owner，最终 UI/API 验证需明确记录采用的 GitHub 表达方式及状态检查名称，避免未来 workflow 改名导致保护失效。
