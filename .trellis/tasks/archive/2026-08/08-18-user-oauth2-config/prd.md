# 规划用户体系 OAuth2 主要认证

## Goal

把 Astro 的用户登录入口调整为 OAuth2 优先：用户通过自建 Authentik（通用 OAuth2/OIDC 配置）登录，服务端完成回调后签发当前系统已有的 JWT，后续 API 仍复用现有 JWT 中间件鉴权。

## Background / Confirmed Facts

- 项目已有本地用户注册、登录和 JWT 认证：`internal/handler/user.go`、`internal/service/user.go`、`internal/middleware/auth.go`。
- 当前公开路由是 `/api/v1/register`、`/api/v1/login`；应用管理路由已使用 JWT 中间件保护：`cmd/server/main.go`。
- 当前配置入口是 `pkg/config/config.go`，默认配置文件是 `configs/config.yaml`。
- `go.mod` 已间接包含 `golang.org/x/oauth2`，可复用现有依赖，不新增 OAuth2 框架。
- 现有用户模型在 `internal/model/model.go`，字段仅覆盖本地账号：UUID、Username、Password、Email、Status；尚无 OAuth2 身份字段或第三方账号绑定模型。
- 项目约束要求：所有 API 统一响应，handler 层参数校验，敏感信息不能出现在日志中，新增错误码必须写入 `pkg/errcode`。

## Requirements

- R1：新增通用 OAuth2/OIDC Provider 配置结构，首个目标是自建 Authentik，支持 `client_id`、`client_secret`、`redirect_url`、`auth_url`、`token_url`、`userinfo_url`、`scopes`、`enabled`。
- R2：配置通过现有 Viper 配置加载路径读取，保持与 `configs/config.yaml` 风格一致。
- R3：新增 OAuth2 登录入口，返回统一响应 JSON，`data.auth_url` 是可跳转到 Authentik 的授权 URL。
- R4：OAuth2 登录入口生成并嵌入 `state`，callback 必须校验 `state`，避免 CSRF。
- R5：新增 OAuth2 callback 处理，使用授权码换取 access token，并通过 OIDC UserInfo 获取用户基础信息。
- R6：OAuth2 登录成功后用 `provider + sub/id` 创建或找到对应用户，并签发现有 JWT，避免重写 API 鉴权体系。
- R7：OAuth2 作为主要认证方式；现有 `/register` 和 `/login` 保留为兼容/兜底入口。
- R8：callback 成功后后端直接返回 `{token, uuid}` JSON，响应结构与现有 `/login` 对齐。
- R9：必须避免在日志、响应或 Swagger 示例中暴露 `client_secret`、access token、authorization code。
- R10：新增最小 OAuth2 身份存储，用于保存 `provider`、`provider_user_id`、`user_id`，不做完整账号绑定管理。
- R11：不引入新的 OAuth2 框架或复杂抽象，优先复用现有 config/handler/service/repository 分层。
- R12：第一版按 OIDC 标准字段读取用户信息：`sub` 作为稳定 ID，`email` 作为邮箱，`preferred_username` 或 `name` 作为用户名来源。

## Acceptance Criteria

- [ ] `config.Load` 后可以读取 Authentik/OIDC Provider 配置，并保留现有 JWT、数据库、K8s 配置行为。
- [ ] 示例配置包含 Authentik/OIDC Provider 模板，但不包含真实 secret。
- [ ] `GET /api/v1/oauth2/:provider/login` 返回统一响应，`data.auth_url` 可跳转到 Authentik，且 URL 包含服务端可校验的 `state`。
- [ ] `GET /api/v1/oauth2/:provider/callback` 校验 `code` 和 `state`，成功后按 `provider + sub/id` 创建/匹配本地用户并返回 `{token, uuid}` JSON。
- [ ] 使用 OAuth2 登录得到的 JWT 可以访问现有受保护应用 API。
- [ ] `/api/v1/register` 和 `/api/v1/login` 继续可用，作为兼容/兜底入口。
- [ ] 任何日志和面向客户端的返回值都不包含 `client_secret`、access token、authorization code。
- [ ] 如有新错误码，均使用 `pkg/errcode` 枚举并配置默认消息。
- [ ] Swagger 文档与新增 OAuth2 接口同步。
- [ ] `make test` 通过。

## Key Decisions

- OAuth2/OIDC 是主认证方式，本地用户名密码登录保留为 fallback。
- 首个目标 Provider 是自建 Authentik，但实现按通用 OAuth2/OIDC 端点配置，不写死 Authentik 域名。
- 用户唯一身份使用 `provider + sub/id`，不只按 email 匹配。
- callback 成功直接返回 JSON Token，不做前端重定向。
- 登录入口返回 JSON 授权 URL，不做服务端 302 跳转。

## Out of Scope

- 前端页面。
- 多租户级别的 Provider 管理。
- 在线配置 OAuth2 Provider 的 CRUD 管理后台。
- refresh token、logout、会话撤销。
- 多个第三方账号绑定同一个本地用户的完整管理能力。
- 非 OIDC 的特殊用户信息字段映射。

## Risks / Deferred Items

- 现有 `users.email` 是唯一索引；若 Authentik email 已被本地账号占用，本阶段不自动绑定，优先返回明确错误，账号绑定留到后续。
- OAuth2-created 用户不设置本地可用密码，因此不能通过本地 `/login` 登录；这是 OAuth2 主认证的预期行为。
