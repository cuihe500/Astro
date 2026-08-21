# Astro GitHub 发布治理实施清单

- [ ] 等镜像 workflow 确定 CI status check 名称和 test/production job 名称。
- [ ] 创建 `test` Environment，录入部署变量 `ASTRO_DEPLOY_HOST` / `ASTRO_DEPLOY_PORT` / `ASTRO_DEPLOY_USER` 与 secrets `ASTRO_DEPLOY_SSH_KEY` / `ASTRO_DEPLOY_KNOWN_HOSTS`；不录入运行时秘密。
- [ ] 创建 `production` Environment，录入同名独立连接信息并配置 `cuihe500` 为 required reviewer。
- [ ] 创建 repository variable `ASTRO_PRODUCTION_ENABLED=false`。
- [ ] 配置 main branch protection/ruleset：PR、required check `CI 校验`、禁止 force push、owner `cuihe500` bypass。
- [ ] 通过 GitHub API/UI 复核 Environments、variable、保护规则和 bypass 语义。
- [ ] 在 push/Release 演练中确认 owner direct push 仍触发 test workflow，正式 Release 且开关 false 的 production job 为 skipped。
- [ ] 将配置名称、状态检查名、验证时间和生产启用前置条件记录到 Astro 发布说明；不记录 secret 值。

## 回滚

- 若 workflow/主机未准备，保留 `ASTRO_PRODUCTION_ENABLED=false` 并移除/禁用未使用的 Environment deployment key。
- 保护规则异常阻塞 owner 时，使用 owner 管理权限调整规则；不要放开 force push 或删除 CI 保护作为长期替代。
