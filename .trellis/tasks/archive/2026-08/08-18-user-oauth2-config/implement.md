# OAuth2 主要认证实施计划

## Checklist

1. 配置层
   - 在 `pkg/config/config.go` 增加 OAuth2 配置结构。
   - 在 `configs/config.yaml` 增加 Authentik/OIDC 示例配置，secret 留空。

2. 数据模型和 Repository
   - 在 `internal/model/model.go` 增加 `OAuthIdentity`。
   - 在 `internal/repository/db.go` 加入 AutoMigrate。
   - 在 `internal/repository/user.go` 增加 OAuth identity 查询/创建、按用户 ID 查询、按 email/username 辅助查询。

3. Service
   - 新增 OAuth2 service，负责 provider 配置校验、state 生成/校验、授权 URL、callback、userinfo 解析。
   - 复用现有 JWT 生成逻辑；如有必要把 `generateToken` 继续留在 user service 内部同文件复用。
   - 新用户创建时使用 OIDC `sub/id` 做稳定身份；用户名冲突时追加短 hash。
   - Authentik email 已被本地账号占用时不自动绑定，返回明确错误。

4. Handler / Routes
   - 在 `internal/handler/user.go` 或同领域新文件中增加 OAuth2 请求/响应 DTO 和 handler。
   - 注册公开路由：
     - `GET /api/v1/oauth2/:provider/login`
     - `GET /api/v1/oauth2/:provider/callback`
   - 响应继续走 `Success` / `HandleError`。

5. 错误码和文档
   - 如新增 OAuth2 错误码，在 `pkg/errcode/code.go` 添加枚举和默认消息。
   - 增加 Swagger 注解后运行 `make swagger`。

6. 测试和校验
   - 补一个最小 `_test.go`，覆盖 state 生成/校验的成功、过期或篡改失败路径。
   - 运行 `make test`。
   - 运行 `make build`。

## Validation Commands

```bash
make swagger
make test
make build
```

如本地安装了 golangci-lint，再运行：

```bash
make lint
```

## Risk / Rollback Points

- 改 `User` 或 `OAuthIdentity` 模型前先保持最小字段，避免迁移扩大。
- `client_secret`、access token、authorization code 不得进入日志、响应和 Swagger 示例。
- OAuth2 路由是新增公开路由；如发现问题，可先移除路由注册或将 provider 配置 `enabled` 设为 `false`。
