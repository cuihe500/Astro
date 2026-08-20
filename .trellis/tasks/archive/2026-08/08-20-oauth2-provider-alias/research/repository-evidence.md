# OAuth2 Provider 别名仓库证据

## 当前泄露面

- `internal/handler/user.go` 的公开路由为 `/oauth2/:provider/login` 和 `/oauth2/:provider/callback`，Swagger 示例直接使用内部键。
- `web/src/features/auth/api.ts` 将内部键作为运行时常量，登录 API、回调 API 和前端 URL 都会出现该值。
- `internal/service/oauth2.go` 直接用传入 Provider 同时完成配置读取、state 生成/校验和 OAuthIdentity 查询/创建，因此当前没有公开别名与内部键的边界。
- OAuth2 state payload 包含 `provider` 字段，并采用 Base64URL 编码而非加密；任何拿到 state 的浏览器用户都可解码该字段。
- `configs/config.yaml` 与 `web/README.md` 的回调示例仍使用内部键路径。
- `docs/` 的 Swagger 生成文件公开展示内部 Provider 示例。

## 兼容性证据

- `internal/service/oauth2.go` 的 `findOrCreateUser` 使用 `provider + provider_user_id` 查询 `OAuthIdentity`，并将 provider 写入新身份。
- `OAuthIdentity` 的唯一约束和既有数据依赖内部 Provider 键；直接把数据库值改为公开别名会导致旧身份无法命中并可能尝试重复建号。
- `oauth2.providers` 配置映射以内部 Provider 键索引；保留该键可避免配置结构和运维参数迁移。
- 因此最小安全改动是在 service 入口解析公开别名：state 使用公开别名，配置与数据库继续使用内部键。

## 验证依据

- 现有 `internal/service/oauth2_test.go` 已覆盖 state 成功、篡改、Provider 不匹配和过期，可直接改用公开别名并增加映射测试。
- 修改 `internal/handler/user.go` Swagger 注解后，项目规范要求执行 `make swagger`。
- 前端已有 `make frontend-check`，包含 ESLint、Vitest 和生产构建。
- 最终还需通过 `make test`、`make build` 和 `make lint`，并用浏览器 Mock API 检查公开登录/回调路径。
