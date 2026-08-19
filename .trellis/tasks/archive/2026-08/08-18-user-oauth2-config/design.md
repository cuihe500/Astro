# OAuth2 主要认证设计

## Architecture / Boundaries

- 保持现有认证边界：OAuth2 只负责登录获取本系统 JWT；后续 API 鉴权继续使用 `internal/middleware/auth.go`。
- 新增代码仍按现有分层：`handler -> service -> repository`。
- 不新增 OAuth2 框架；直接使用已有 `golang.org/x/oauth2` 和标准库 HTTP/JSON。
- Provider 通过配置驱动，第一版按 Authentik/OIDC 标准字段工作。

## API Contract

### GET `/api/v1/oauth2/:provider/login`

- 公开路由，无需 JWT。
- `:provider` 对应配置中的 provider key，例如 `authentik`。
- 成功响应：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "auth_url": "https://auth.example.com/application/o/authorize/?..."
  }
}
```

- `auth_url` 内包含服务端签名的 `state`。
- provider 不存在或未启用时返回统一错误响应。

### GET `/api/v1/oauth2/:provider/callback`

- 公开路由，无需 JWT。
- 必需 query：`code`、`state`。
- 校验 `state` 后，用 `code` 换 token，再请求 `userinfo_url`。
- 成功响应复用现有登录响应语义：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "token": "jwt",
    "uuid": "user-uuid"
  }
}
```

## Config Shape

```yaml
oauth2:
  providers:
    authentik:
      enabled: false
      client_id: ""
      client_secret: ""
      redirect_url: "http://localhost:8080/api/v1/oauth2/authentik/callback"
      auth_url: "https://auth.example.com/application/o/authorize/"
      token_url: "https://auth.example.com/application/o/token/"
      userinfo_url: "https://auth.example.com/application/o/userinfo/"
      scopes:
        - openid
        - email
        - profile
```

`client_secret` 只进入服务端配置，不进入响应、日志或 Swagger 示例。

## Data Model

新增最小模型 `OAuthIdentity`：

- `provider`：OAuth2 Provider key。
- `provider_user_id`：OIDC `sub`，缺失时兜底使用 `id`。
- `user_id`：关联 `users.id`。

约束：`provider + provider_user_id` 唯一；`user_id` 建索引。

`User` 保持现有字段。OAuth2 新用户使用 OIDC `email`、`preferred_username/name` 初始化；不设置本地可用密码。

## Data Flow

1. 客户端请求 `/oauth2/:provider/login`。
2. service 读取 provider 配置，生成签名 `state`，用 `oauth2.Config.AuthCodeURL` 生成 `auth_url`。
3. 客户端跳转到 Authentik。
4. Authentik 回调 `/oauth2/:provider/callback?code=...&state=...`。
5. service 校验 `state`，用 code 换 access token。
6. service 请求 `userinfo_url`，读取 `sub/id`、`email`、`preferred_username/name`。
7. repository 按 `provider + provider_user_id` 查 `OAuthIdentity`。
8. 找到则读取对应 `User` 并签发 JWT。
9. 未找到则创建 `User` 和 `OAuthIdentity`，再签发 JWT。

## State Validation

采用无状态签名方案，避免新增 session 表：

- state 内容包含 provider、过期时间、随机 nonce。
- 使用现有 JWT secret 做 HMAC-SHA256 签名。
- callback 校验签名、provider 和过期时间。

## Compatibility

- `/api/v1/register` 和 `/api/v1/login` 不删除、不改响应，作为 fallback。
- 现有 JWT claims 和中间件不变。
- 现有应用 API 不需要修改。

## Trade-offs

- 不做 OIDC discovery，端点由配置显式提供；少代码，适合自建 Authentik。
- 不按 email 自动绑定已有本地用户；避免账号接管风险，账号绑定留到后续。
- 不存 refresh token；本系统仍只签发当前 JWT。

## Rollback

- 删除 OAuth2 路由注册即可停止新入口。
- 配置 `enabled: false` 可停用 provider。
- 数据库新增表通过 AutoMigrate 创建，不影响现有用户登录和应用 API。
