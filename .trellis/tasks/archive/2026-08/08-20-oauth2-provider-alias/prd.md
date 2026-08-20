# 隐藏 OAuth2 内部 Provider 名称

## 目标

将 BytCloud Auth 的公开 OAuth2 标识统一为 `bytcloudauth`，确保浏览器地址、前端请求、OAuth2 state 和公开 API 文档不暴露内部 Provider 名称 `authentik`，同时保留现有内部配置键与数据库身份键，避免账号迁移和身份重复。

## 已确认事实

- 当前前端回调为 `/oauth2/authentik/callback`，登录和回调 API 也将 `authentik` 作为公开 `:provider` 参数。
- 当前 OAuth2 state 的 `provider` 字段直接写入传入的 Provider 名称，解码后可看到内部键。
- `OAuthIdentity.Provider` 已使用内部 Provider 键识别既有身份，直接改名会造成历史身份无法命中。
- 后端配置 `oauth2.providers.authentik` 是内部实现细节，可以继续保留；用户指定的公开 URL 别名为小写 `bytcloudauth`。
- 生产站点为 `https://astro.bytcloud.org`。

## 需求

### R1. 公开 URL 与 API

- 本地前端回调地址必须为 `http://localhost:5173/oauth2/bytcloudauth/callback`。
- 生产前端回调地址必须记录为 `https://astro.bytcloud.org/oauth2/bytcloudauth/callback`。
- 前端必须通过以下公开 API 完成认证：
  - `GET /api/v1/oauth2/bytcloudauth/login`
  - `GET /api/v1/oauth2/bytcloudauth/callback?code=<code>&state=<state>`
- React 回调路由可以继续使用通用 `:provider` 结构，为以后新增公开 Provider 别名保留同一 URL 形状，但当前只接受 `bytcloudauth`。

### R2. 公开别名与内部 Provider 映射

- 后端在 OAuth2 边界将公开别名 `bytcloudauth` 映射到内部 Provider 键 `authentik`。
- OAuth2 state 必须记录并校验公开别名 `bytcloudauth`，不得写入内部 Provider 键。
- 配置读取、OAuth2 token/userinfo 交互和 `OAuthIdentity.Provider` 仍使用内部 Provider 键 `authentik`。
- 直接请求公开路径 `/api/v1/oauth2/authentik/...` 必须返回统一参数错误，不得继续作为兼容入口，也不得在响应中回显内部名称。
- 不修改既有 OAuthIdentity 数据，不新增数据库迁移。

### R3. 品牌与可见信息

- 普通用户界面继续只显示 `BytCloud Auth`。
- 前端源代码运行时常量、构建产物、浏览器 URL、网络请求路径和可解码 OAuth2 state 中不得出现 `authentik`。
- 内部名称仅可保留在后端配置、后端映射代码、数据库值、内部测试和技术规范中。
- 后端返回给客户端的错误信息不得包含内部 Provider 名称。

### R4. 配置与文档

- `configs/config.yaml` 的示例 `redirect_url` 改为本地公开回调地址。
- `web/README.md` 和认证规范同时记录本地及生产公开回调地址，并说明公开别名与内部键的边界。
- Swagger 中公开 Provider 示例必须使用 `bytcloudauth`，修改注解后同步生成文档。

## 验收标准

- [ ] 点击“使用 BytCloud Auth 继续”时，请求路径为 `/api/v1/oauth2/bytcloudauth/login`。
- [ ] 本地 OAuth2 回调进入 `/oauth2/bytcloudauth/callback`，生产登记地址为 `https://astro.bytcloud.org/oauth2/bytcloudauth/callback`。
- [ ] 前端回调只调用 `/api/v1/oauth2/bytcloudauth/callback`，成功后仍建立 Astro JWT 会话。
- [ ] 解码 OAuth2 state 后 Provider 值为 `bytcloudauth`，不包含 `authentik`。
- [ ] 后端使用内部键 `authentik` 读取配置并查询/创建 OAuthIdentity，既有身份无需迁移即可继续登录。
- [ ] 直接请求 `/api/v1/oauth2/authentik/login` 或 callback 会失败，统一错误响应不回显内部名称。
- [ ] `web/src` 和前端生产构建产物不包含大小写不敏感的 `authentik` 文本。
- [ ] Swagger、当前认证规范和 Web README 的公开示例均使用 `bytcloudauth`。
- [ ] `make frontend-check`、`make swagger`、`make test`、`make build` 和 `make lint` 通过。

## 范围外

- 新增第二个 OAuth2 Provider 或 Provider 选择界面。
- 将公开别名做成数据库表、管理 API 或动态配置系统。
- 重命名 `oauth2.providers.authentik`、迁移 `OAuthIdentity.Provider` 或修改现有身份唯一约束。
- BytCloud Auth 应用侧配置代办、生产部署、反向代理和统一登出。
