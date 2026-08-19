# 前端 MVP 仓库证据

## 仓库与工具链

- 仓库当前没有 `package.json` 或前端目录，只有 Go/Gin 后端。
- 本地工具链为 Node `v24.19.0`、npm `11.17.0`，可运行当前 Vite/React 工具链。
- 根 `Makefile` 当前只有 Go 的 build/run/test/lint/swagger/clean 目标，项目规范要求所有开发和验证命令通过 Makefile 目标执行。
- `.trellis/spec/` 当前只有 backend 层，没有既有前端规范；实现需遵循根 `AGENTS.md` 的 UI、KISS、验证和中文文档要求。

## API 事实

### 统一响应

来源：`internal/handler/response.go`

- 成功与错误均返回 HTTP 200。
- JSON 结构为 `code`、`message`、可选 `data`。
- `Success(nil)` 因 `omitempty` 省略 `data`，前端类型不能要求该字段始终存在。
- 前端必须以 `code === 0` 判断业务成功。

### 本地认证与 OAuth2

来源：`internal/handler/user.go`、`internal/middleware/auth.go`

- `POST /api/v1/register`：请求 `username/password/email`，成功无 data。
- `POST /api/v1/login`：成功 data 为 `{token, uuid}`。
- `GET /api/v1/oauth2/:provider/login`：成功 data 为 `{auth_url}`。
- `GET /api/v1/oauth2/:provider/callback?code&state`：成功 data 与本地登录一致。
- 受保护请求使用精确的 `Authorization: Bearer <token>`。
- 认证失效相关 code 为 `10002`、`20011`、`20012`。

### 应用管理

来源：`internal/handler/app.go`、`internal/model/model.go`

- `POST /api/v1/apps` 请求：`name`、`image`、`replicas`、`port`。
- `replicas` 的实际 binding 同时包含 `required` 与 `min=0`，零值会被 required 拒绝；首版创建界面使用 1-10。
- 受保护路由包括列表、详情、删除、启动、停止、重启、日志。
- 日志查询参数 `lines` 默认 100；成功 data 为 `{logs}`。
- App JSON 只有 `id/name/image/replicas/status/user_id/namespace/created_at/updated_at`，不含 port，前端详情不得虚构端口。
- 已知状态包括 pending、running、stopped、starting、restarting、unknown；状态同步可能异步延迟。

## OAuth2 前端接入约束

来源：`internal/service/oauth2.go`、`configs/config.yaml`
- 产品命名映射：对外认证品牌为 `BytCloud Auth`；`authentik` 是内部 provider key，`Authentik` 只用于代码、配置、技术文档和运维上下文，不用于普通 UI 文案。

- 服务端生成带签名、provider、过期时间和 nonce 的 state，并在 callback 校验。
- `oauth2.Config.RedirectURL` 同时参与授权 URL 和 code exchange，因此 Provider 注册地址、后端配置和前端回调 URL 必须完全一致。
- 当前示例回调到 API 地址 `http://localhost:8080/api/v1/oauth2/authentik/callback`，浏览器最终只会看到 JSON，无法自动写入前端会话。
- 首版把示例地址改为 `http://localhost:5173/oauth2/authentik/callback`。前端取得 code/state 后调用现有 API callback，JWT 仍只由后端签发。
- `findOrCreateUser` 在首次 provider identity 不存在时创建 `User` 与 `OAuthIdentity`，已有 identity 直接读取用户并签发 JWT；因此 OAuth2 可以承担 Astro 注册。
- 首次 OAuth2 登录若 email 已被本地账号占用，后端返回 `ErrEmailExists`，不会自动绑定；前端必须保留本地兜底但不暗示自动合并账号。
- Provider 默认关闭且没有“查询已启用 Provider”接口。首版固定提供 BytCloud Auth 首选入口（内部 provider key 为 `authentik`），未启用或 Enrollment 被拒绝时展示错误并保留本地登录。
- authorization code、state、client secret、外部 access token 和本系统 JWT 均不得进入日志或可分享 URL。

## 浏览器与部署约束

来源：`cmd/server/main.go`

- Gin 当前没有 CORS 中间件。
- 本地开发使用 Vite 同源路径 `/api` 代理到 `http://localhost:8080`，无需修改后端 CORS。
- 首版只保证本地开发与前端生产构建；生产静态托管、反向代理和跨域策略明确延期。

## UI/UX 本地研究

来源：`.agents/skills/ui-ux-pro-max` 本地数据检索

- “Minimalism & Swiss Style”适合工具和工作台，性能成本低，可访问性风险低；采用清晰网格、高对比、少装饰。
- 检索结果中的营销 Hero 模式与本任务不匹配，已明确舍弃；第一屏直接提供登录或应用管理。
- 使用浅中性背景、绿色主操作、蓝色信息、琥珀警告和红色危险操作，避免单一蓝色系。
- 远程 Fira 字体和 GSAP 动效没有用户价值，首版使用系统字体与原生过渡。
- React 指南要求以 TypeScript 定义 props，并通过 form `onSubmit` 处理提交。
- UX 指南要求删除前确认、操作成功反馈、稳定的加载状态、可见 hover/focus 状态。
- 验收视口为 375、768、1024、1440px；需要键盘可达、4.5:1 文本对比、可见 focus ring 与 reduced-motion 支持。
