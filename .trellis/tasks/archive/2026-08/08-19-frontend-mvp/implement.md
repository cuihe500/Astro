# 前端第一版本实施计划

## 变更原则

- 保持单一任务交付：前端工程、OAuth2 回调配置和 Makefile 目标共同组成一个可验证的用户闭环，不拆分父子任务。
- 先建立最小工程和 API 边界，再逐页实现；不得先引入通用组件或第三方库等待未来使用。
- 只修改 `web/`、`Makefile`、`configs/config.yaml`、`.trellis/spec/backend/auth-guidelines.md` 及任务规划文档。
- 若必须新增 Go 路由、数据库迁移或生产部署逻辑，停止实施并回到规划阶段确认范围。

## 实施清单

### 1. 建立前端工程

- [ ] 创建 `web/` 的 React + TypeScript + Vite 工程及 npm lockfile。
- [ ] 仅安装 React Router、Lucide React 与 Vitest 等已批准依赖。
- [ ] 配置 Vite `/api` 开发代理和 `VITE_API_BASE_URL` 类型声明。
- [ ] 在 `Makefile` 增加 `frontend-install`、`frontend-run`、`frontend-lint`、`frontend-test`、`frontend-build`、`frontend-check` 目标，并补全 `.PHONY`。
- [ ] 创建 `web/README.md` 与 `.env.example`，记录命令、目录边界、同源要求和 Authentik 回调配置。

阶段验证：

```bash
make frontend-install
make frontend-build
```

回滚点：删除初始 `web/` 与新增 Makefile 目标，不影响后端。

### 2. 建立 API、会话和路由边界

- [ ] 定义 `ApiResponse<T>`、`ApiError` 和统一 `request<T>`，覆盖业务 `code`、网络错误、不可解析响应及可选 `data`。
- [ ] 统一添加 Bearer Token；认证错误码 `10002/20011/20012` 必须清理会话并返回登录。
- [ ] 实现 JWT 的单一 localStorage 读写模块，不存储 authorization code、state 或密码。
- [ ] 建立公开路由、保护路由、根路径跳转与 404 回退。
- [ ] 为响应解包成功/失败和认证错误判断补最小 Vitest 测试。

阶段验证：

```bash
make frontend-test
make frontend-lint
```

回滚点：`web/src/lib/api.ts`、`web/src/features/auth/session.ts`、`web/src/app/router.tsx` 是跨页面核心边界；发现契约不一致时先修正这里，禁止在每个页面单独兼容。

### 3. 实现认证页面

- [ ] 实现实际登录页，首要按钮为“使用 BytCloud Auth 继续”，说明该入口同时支持已有用户登录和首次用户注册；普通界面不得出现 Authentik 内部名称。
- [ ] 将本地登录与注册入口作为次要兜底流程，包含密码显示/隐藏、密码管理器兼容和字段错误。
- [ ] 实现本地注册页，提供可见标签、原生邮箱校验和提交中状态。
- [ ] 实现 OAuth2 callback loader：校验 provider/code/state、调用现有 callback、保存 JWT、跳转应用列表。
- [ ] 展示首次 BytCloud Auth 登录自动创建 Astro 账号的结果；不在前端复制 Authentik Enrollment Flow。
- [ ] 实现 OAuth2 Provider 未启用、Enrollment 被拒绝、缺少参数和回调失败页面，始终提供返回本地登录路径。
- [ ] 将 `configs/config.yaml` 的示例 Authentik `redirect_url` 改为 `http://localhost:5173/oauth2/authentik/callback`，并在认证规范中同步回调示例以及 `BytCloud Auth` 对外名称 / `authentik` 内部 key 的映射。

阶段验证：优先使用浏览器 Mock API 验证 BytCloud Auth 首选按钮、首次 OAuth2 注册/已有用户登录、Provider/Enrollment 失败降级；再验证本地登录和注册。

回滚点：恢复 OAuth2 redirect URL 后，本地认证仍应独立可用。

### 4. 实现应用管理闭环

- [ ] 实现应用领域类型与 API 函数，字段严格匹配现有 Go JSON 模型。
- [ ] 实现应用列表：加载、空数据、失败重试、刷新和创建入口。
- [ ] 实现创建页：应用名称、镜像、副本和端口校验，成功后进入新应用详情。
- [ ] 实现应用详情：基础信息、状态、更新时间与返回路径。
- [ ] 按状态显示启动、停止、重启操作，提交期间禁用重复触发并在完成后刷新详情。
- [ ] 使用原生 dialog 实现删除确认，成功后返回列表。
- [ ] 实现最近 100 行日志加载、空数据、失败重试和重新加载。

阶段验证：使用浏览器 Mock API 走通列表 -> 创建 -> 详情 -> 启停/重启 -> 日志 -> 删除。

回滚点：页面不得绕过 `features/apps/api.ts` 直接解析原始响应；契约问题统一在 API/类型层处理。

### 5. 完成视觉与响应式

- [ ] 在单一 `styles.css` 中建立 primitive、semantic、component token，并完成认证、导航、列表、表单、详情、dialog、反馈和日志样式。
- [ ] 使用 Lucide 图标；图标按钮具备 `aria-label` 和 `title`，装饰图标从辅助技术隐藏。
- [ ] 检查普通 UI 文案只显示 `BytCloud Auth`，`Authentik` 仅保留在代码、配置和技术文档。
- [ ] 提供 hover、focus、disabled、loading 状态，动态消息使用 `aria-live`。
- [ ] 在 375、768、1024、1440px 检查布局；移动端不得出现整页横向滚动，日志只在自身区域滚动。
- [ ] 尊重 `prefers-reduced-motion`，避免动画改变布局尺寸。

阶段验证：生成桌面与 375px 移动截图，并检查文本截断、按钮溢出、重叠、空白页面和控制台错误。

### 6. 全量质量检查

- [ ] 执行前端 lint、测试和生产构建。
- [ ] 执行后端现有测试、构建和 lint，确认配置与 Makefile 修改无回归。
- [ ] 使用 Mock API 完成关键浏览器流程，并检查未登录守卫、Token 过期跳转和双击防护。
- [ ] 检查 Git diff，只保留任务范围内文件；确认没有 Token、client secret 或真实 Provider 地址进入代码和截图。
- [ ] 对照 PRD 逐项确认验收标准，记录未验证的真实 Authentik/K8s 环境风险。

最终命令：

```bash
make frontend-check
make test
make build
make lint
```

若 `make lint` 因环境缺少 `golangci-lint` 无法运行，必须明确记录；不得用单独命令替代 Makefile 目标。

## 浏览器验收矩阵

| 场景 | 预期 |
|---|---|
| 无会话访问 `/apps` | 跳转 `/login` |
| 认证页初始状态 | BytCloud Auth 为主要操作，账号密码入口明确但次要 |
| BytCloud Auth 首次登录且允许 Enrollment | 完成 OAuth2 后自动创建 Astro 用户并进入 `/apps` |
| BytCloud Auth 已有身份登录 | 完成 OAuth2 后复用身份并进入 `/apps` |
| BytCloud Auth 未启用或 Enrollment 被拒绝 | 展示对外品牌错误信息，本地账号兜底入口仍可用 |
| 本地登录/注册成功 | 保存 JWT 并进入 `/apps` |
| API 返回认证错误码 | 清理 JWT 并返回 `/login` |
| OAuth2 callback 成功 | 只交换一次 code，保存 JWT 并进入 `/apps` |
| 应用列表为空 | 显示空状态和创建入口 |
| 创建连续点击 | 只发出一次请求 |
| 运行中应用 | 可停止、重启，不显示启动 |
| 已停止应用 | 可启动，不显示停止/重启 |
| 过渡/未知状态 | 只提供刷新与删除 |
| 删除 | 先确认，成功后回列表 |
| 日志长行 | 日志区域内部滚动，页面不横向溢出 |
| 375px 视口 | 无重叠、无整页横向滚动、主要控件可操作 |

## 评审门槛

启动实现前必须确认：

- `prd.md` 没有开放问题并已通过收敛检查。
- `design.md` 与本计划完整覆盖 OAuth2 回调、统一响应和响应式边界。
- `implement.jsonl` 与 `check.jsonl` 至少包含真实项目规范和 API 契约上下文。
- 用户已在看到最终规划摘要后明确批准实施。

## 执行记录

- 已完成 `web/` React + TypeScript + Vite SPA、BytCloud Auth OAuth2 回调、本地账号兜底、JWT 会话、应用管理闭环和响应式样式。
- 已通过 `make frontend-check`、`make test`、`make build` 和 `make lint`。
- 已使用 Playwright Mock API 验证登录守卫、Bearer 请求、创建防重复提交、生命周期、日志、删除确认、认证过期跳转、OAuth2 错误降级和 375px 无横向溢出。
- 未连接真实 Authentik、MariaDB 或 Kubernetes 环境；生产静态托管、CORS 和 Cookie 会话仍属于范围外。
- `make test` 保持仓库原有的 `go test -v ./...` 递归语义；安装前端依赖后会额外发现依赖目录中的 Go 示例包，但不影响测试结果。
