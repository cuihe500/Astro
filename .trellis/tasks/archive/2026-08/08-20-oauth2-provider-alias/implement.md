# OAuth2 公开 Provider 别名实施计划

## 变更原则

- 只增加一个固定公开别名到内部键的映射，不引入 Provider 注册表、管理 API 或新配置层。
- 公开别名用于 URL、API 参数和 state；内部键只用于配置与 OAuthIdentity。
- 旧公开内部键路径直接失败，不提供会继续泄露内部名称的兼容路由。
- 不修改数据库模型、迁移或既有 OAuthIdentity。

## 实施清单

### 1. 建立后端别名边界

- [ ] 在 `internal/service/oauth2.go` 定义 `bytcloudauth` 公开别名和 `authentik` 内部键。
- [ ] 实现最小 `resolveOAuth2Provider`，仅接受公开别名，并返回不含输入值的统一错误。
- [ ] `BuildAuthURL` 使用内部键读取配置，但用公开别名生成 state。
- [ ] `Callback` 用公开别名校验 state，再使用内部键交换 token 和查询/创建 OAuthIdentity。
- [ ] 保持 repository 与模型不变。

验证：

```bash
make test
```

回滚点：若既有身份无法命中，优先检查 `findOrCreateUser` 是否仍收到内部键，禁止迁移数据库作为补救。

### 2. 补齐后端回归测试与 Swagger

- [ ] 增加别名解析成功测试。
- [ ] 增加内部键作为公开别名被拒绝的测试。
- [ ] state 测试改为公开别名，并验证 provider 不匹配、篡改和过期行为。
- [ ] Swagger provider 示例改为 `bytcloudauth`，参数说明改为“公开 Provider 别名”。
- [ ] 执行 `make swagger` 更新 `docs/`。

验证：

```bash
make swagger
make test
```

### 3. 更新前端公开路径

- [ ] 将认证常量改为 `BYTCLOUD_PROVIDER_ALIAS = "bytcloudauth"`。
- [ ] 登录 API 请求 `/oauth2/bytcloudauth/login`。
- [ ] callback loader 只接受 `bytcloudauth`，并调用对应 callback API。
- [ ] 删除前端源代码中的内部 Provider 文本与错误替换逻辑。
- [ ] 必要时补充最小前端测试，确保公开路径不会回退到内部键。

验证：

```bash
make frontend-check
```

回滚点：前端与后端公开别名必须同批修改，禁止只改回调页面造成登录入口与 callback 不一致。

### 4. 同步配置与文档

- [ ] 将 `configs/config.yaml` 示例 redirect URL 改为 `http://localhost:5173/oauth2/bytcloudauth/callback`。
- [ ] 更新 `web/README.md`，同时记录本地和生产回调地址。
- [ ] 更新 `.trellis/spec/backend/auth-guidelines.md` 的公开别名、内部键、API、state、DB 和错误边界。
- [ ] 检查当前非归档文档的公开示例不再使用内部键。

### 5. 全量验收

- [ ] 使用浏览器 Mock API 验证登录按钮请求公开别名 API。
- [ ] 验证成功回调 URL 为 `/oauth2/bytcloudauth/callback`，且只交换一次 code。
- [ ] 验证错误回调页面只显示 BytCloud Auth。
- [ ] 解码测试 state，确认 Provider 为 `bytcloudauth`。
- [ ] 搜索 `web/src` 和 `web/dist`，确认不含大小写不敏感的内部名称。
- [ ] 检查 git diff，无数据库迁移、真实凭据或范围外文件。

最终命令：

```bash
make frontend-check
make swagger
make test
make build
make lint
```

## 风险与未验证项

- Provider 控制台必须同步登记新的本地/生产回调地址，否则 code exchange 会因 redirect URI 不一致失败。
- 无真实 BytCloud Auth 凭据时只能验证契约、单元测试与 Mock 浏览器流程；真实 Provider 联调作为部署时验证项记录。

## 执行记录

- 已完成公开别名到内部 Provider 键的服务层映射；OAuth2 state 使用 `bytcloudauth`，配置与 OAuthIdentity 保持内部键，无数据库迁移。
- 已将旧浏览器回调别名安全重定向至公开错误 URL，且不携带原始 `code` 或 `state`。
- 已通过 `make frontend-check`、`make swagger`、`make test`、`make build`、`make lint` 和任务上下文校验。
- 浏览器 Mock API 已验证公开登录/回调各请求一次、JWT 建立、旧浏览器 URL 清理、无内部名称页面/网络泄露；真实 Provider、MariaDB 既有身份和生产反向代理仍未联调。
