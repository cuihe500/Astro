# Astro GitHub 发布治理实施清单

- [x] 等镜像 workflow 确定 CI status check 名称和 test/production job 名称。
- [x] 创建 `test` Environment，录入部署变量 `ASTRO_DEPLOY_HOST` / `ASTRO_DEPLOY_PORT` / `ASTRO_DEPLOY_USER` 与 secrets `ASTRO_DEPLOY_SSH_KEY` / `ASTRO_DEPLOY_KNOWN_HOSTS`；不录入运行时秘密。
- [ ] 创建 `production` Environment，录入同名独立连接信息并配置 `cuihe500` 为 required reviewer。
- [ ] 创建 repository variable `ASTRO_PRODUCTION_ENABLED=false`。
- [ ] 配置 main branch protection/ruleset：PR、required check `CI 校验`、禁止 force push、owner `cuihe500` bypass。
- [ ] 通过 GitHub API/UI 复核 Environments、variable、保护规则和 bypass 语义。
- [x] 在 main push 演练中确认 owner direct push 仍触发 test workflow，生产 job 为 skipped。
- [ ] 在正式 Release 演练中确认开关为 false 时 production job 为 skipped。
- [ ] 将配置名称、状态检查名、验证时间和生产启用前置条件记录到 Astro 发布说明；不记录 secret 值。

## 回滚

- 若 workflow/主机未准备，保留 `ASTRO_PRODUCTION_ENABLED=false` 并移除/禁用未使用的 Environment deployment key。
- 保护规则异常阻塞 owner 时，使用 owner 管理权限调整规则；不要放开 force push 或删除 CI 保护作为长期替代。

## 本轮记录（2026-08-21）

- GitHub API 已确认 `test` Environment 仅包含上述三项变量与两项 secret；未录入运行时秘密。
- 测试部署公钥已通过主机既有安装入口写入 forced-command authorized_keys；JP/KR 外部链路均验证任意命令与可变 tag 在部署前被拒绝。
- 治理配置步骤本身未触发发布；后续 main push 的 Run `32462154081` 已证明测试 Environment 可用。仍未配置 production、分支保护或生产开关，也未创建 tag/Release。
