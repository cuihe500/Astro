# 认证规范

> 本文件记录 Astro 后端认证相关的可执行契约。实现时仍需同时遵循错误处理、数据库、日志和质量规范。

---

## OAuth2/OIDC 主要认证

### 1. Scope / Trigger

- 触发条件：新增或修改第三方登录、OAuth2/OIDC Provider 配置、登录回调、JWT 签发、用户身份映射。
- 目标边界：OAuth2/OIDC 只负责把外部身份换成本系统 JWT；受保护 API 继续使用 `internal/middleware/auth.go` 的 JWT 中间件。
- 当前主 Provider 目标是自建 Authentik，但实现不得写死 Authentik 域名，必须通过配置提供端点。

### 2. Signatures

#### API

- `GET /api/v1/oauth2/:provider/login`
  - 公开路由。
  - 返回统一响应，`data.auth_url` 是客户端可跳转的授权 URL。
- `GET /api/v1/oauth2/:provider/callback?code=<code>&state=<state>`
  - 公开路由。
  - 成功返回统一响应，`data.token` 和 `data.uuid` 与本地 `/login` 对齐。

#### 前端回调

- 用户可见的认证品牌为 `BytCloud Auth`；`authentik` 是内部 provider key，只出现在代码、配置、技术文档和运维上下文。
- 本地开发前端回调地址为 `http://localhost:5173/oauth2/authentik/callback`。后端 `redirect_url`、Provider 注册地址和 code exchange 使用的地址必须完全一致。
- 前端 callback loader 只把当前 URL 中的一次性 `code`、`state` 交给后端 callback API，成功后保存后端签发的 JWT；不保存 authorization code 或 state。
- Provider 未启用、Enrollment Flow 拒绝或 callback 失败时，前端展示 BytCloud Auth 的可理解错误并保留账号密码备用入口，不在前端复制外部注册流程。

#### 前端 API 消费

- 前端所有请求必须解析 `code`、`message` 和可选 `data`；只有 `code === 0` 才视为成功，不能只依据 HTTP 200。
- 受保护请求必须发送 `Authorization: Bearer <JWT>`；JWT 只保存在浏览器会话存储边界，不写入日志或页面。
- 收到 `10002`、`20011` 或 `20012` 时，前端必须删除本地 JWT 并回到登录页；其他业务错误优先展示后端 `message`。

#### Config

```yaml
oauth2:
  providers:
    authentik:
      enabled: false
      client_id: ""
      client_secret: ""
      redirect_url: "http://localhost:5173/oauth2/authentik/callback"
      auth_url: "https://auth.example.com/application/o/authorize/"
      token_url: "https://auth.example.com/application/o/token/"
      userinfo_url: "https://auth.example.com/application/o/userinfo/"
      scopes:
        - openid
        - email
        - profile
```

#### DB

- `OAuthIdentity.Provider`：Provider key，例如 `authentik`。
- `OAuthIdentity.ProviderUserID`：OIDC `sub`，缺失时才兜底使用 `id`。
- `OAuthIdentity.UserID`：本地 `users.id`。
- 唯一约束：`provider + provider_user_id`。

### 3. Contracts

- 登录入口必须生成可校验的 `state`，并把它放入授权 URL。
- `state` 至少包含 provider、过期时间、随机 nonce，并使用服务端 secret 做 HMAC 签名。
- callback 必须校验 `state` 的签名、provider 和过期时间。
- UserInfo 第一版只依赖 OIDC 标准字段：`sub`、`email`、`preferred_username`、`name`；特殊字段映射不属于当前基础契约。
- OAuth2 用户唯一身份必须使用 `provider + sub/id`，不能只按 email 识别。
- OAuth2 创建的新用户不能拥有可用本地密码；本地 `/login` 只是 fallback，不自动登录 OAuth2-only 用户。
- `client_secret`、access token、authorization code 不得进入日志、响应和 Swagger 示例。
- 生产环境 JWT secret 必须通过环境变量或 Secret 管理系统注入，不得使用仓库中的示例/默认值；配置加载或启动时发现 secret 为空、仍为默认值或强度不足必须失败。

### 4. Validation & Error Matrix

| 条件 | 行为 |
|------|------|
| Provider 不存在、未启用或关键端点缺失 | 返回 `ErrBadRequest` |
| callback 缺少 `provider`、`code` 或 `state` | handler 返回 `ErrBadRequest` |
| `state` 签名错误、provider 不匹配或过期 | 返回 `ErrUnauthorized` |
| 换 token 失败、UserInfo 请求失败或 UserInfo 缺少 `sub/id` | 返回 `ErrLoginFailed` |
| `provider + sub/id` 已存在 | 读取本地用户并签发 JWT |
| 首次 OAuth2 登录且 email 已被本地账号占用 | 返回 `ErrEmailExists`，不自动绑定 |
| Repository 返回非 NotFound 错误 | service 转为 `ErrDatabase` |

### 5. Good / Base / Bad Cases

- Good：Authentik 返回 `sub`、`email`、`preferred_username`，系统创建 User + OAuthIdentity，并返回 `{token, uuid}`。
- Base：Authentik 不返回 email，系统使用本地占位邮箱，仍以 `provider + sub` 识别用户。
- Bad：只按 email 匹配用户，会造成账号接管风险，禁止这样实现。

### 6. Tests Required

- state 生成/校验成功路径。
- state 被篡改、provider 不匹配、过期失败路径。
- 涉及 UserInfo 解析或用户创建逻辑时，至少覆盖 `sub` 缺失、email 冲突、已有 identity 命中之一。
- 改 handler 注解后运行 `make swagger`；提交前至少运行 `make test` 和 `make build`，可用时运行 `make lint`。

### 7. Wrong vs Correct

#### Wrong

```go
// 只按 email 查找用户，可能把外部账号绑定到错误的本地账号。
user, err := repo.GetUserByEmail(userInfo.Email)
```

#### Correct

```go
identity, err := repo.GetOAuthIdentity(provider, userInfo.Sub)
// 未找到 identity 时创建新 User + OAuthIdentity；email 冲突不自动绑定。
```
