# 前端第一版本技术设计

## 1. 变更边界

本任务新增仓库内独立的 `web/` 单页应用，并只对根目录 `Makefile` 与 Authentik 示例回调地址做必要调整。现有 Go 路由、OAuth2 service、数据库模型和应用业务逻辑保持不变。

最小行为差距是：后端已有完整 API，但用户没有可操作的浏览器界面；OAuth2 当前直接回调 API 并返回 JSON，无法自动建立前端会话。

预计修改范围：

- `web/`：前端工程、页面、API 客户端、样式、测试与使用说明。
- `Makefile`：增加前端安装、开发、检查、测试和构建目标。
- `configs/config.yaml`：将 Authentik 示例 `redirect_url` 改为前端回调路由。
- `.trellis/spec/backend/auth-guidelines.md`：同步更新 OAuth2 回调契约示例与 BytCloud Auth / `authentik` 命名映射。

明确不做：新增 Go API、Cookie 会话、账号自动绑定、后端 CORS、生产静态资源托管、实时日志、监控和模板市场。

## 2. 技术选型

- React + TypeScript + Vite：适合本仓库从零建立可维护 SPA，开发反馈快，Node 24 环境已满足要求。
- React Router：负责认证守卫、页面 URL 和 OAuth2 回调 loader，避免在 React effect 中重复交换一次性 authorization code。
- 浏览器原生 `fetch`：现有 API 简单，不引入 Axios 或请求缓存库。
- Lucide React：统一提供导航、操作和状态图标。
- 单一 CSS 文件 + CSS 变量：使用原生 CSS 完成响应式与设计 token，不引入 Tailwind、CSS-in-JS 或组件框架。
- Vitest：只覆盖统一响应解包等关键分支；交互流程由浏览器验证覆盖。

不引入全局状态库、表单库、schema 库、数据缓存库或自建组件设计系统。认证只有一个持久会话源，页面请求量小，React 局部状态足够。

## 3. 目录组织

```text
web/
├── .env.example
├── README.md
├── index.html
├── package.json
├── package-lock.json
├── tsconfig*.json
├── vite.config.ts
├── eslint.config.js
└── src/
    ├── app/
    │   └── router.tsx              # 路由、守卫和 OAuth2 callback loader
    ├── components/
    │   ├── AppShell.tsx            # 登录后的统一导航框架
    │   └── Feedback.tsx            # 跨页面加载、空数据和错误反馈
    ├── features/
    │   ├── auth/
    │   │   ├── api.ts              # 注册、登录、OAuth2 API
    │   │   ├── session.ts          # JWT 的唯一读写边界
    │   │   └── pages/              # 登录、注册、OAuth2 回调页
    │   └── apps/
    │       ├── api.ts              # 应用 CRUD、生命周期和日志 API
    │       ├── components/
    │       │   └── StatusBadge.tsx # 应用状态语义映射
    │       ├── types.ts            # App 与请求类型
    │       └── pages/              # 列表、创建、详情页
    ├── lib/
    │   ├── api.ts                  # 统一响应、Bearer 与错误处理
    │   └── api.test.ts
    ├── main.tsx
    └── styles.css                  # token、基础样式与页面布局
```

页面留在所属 feature，只有跨 feature 复用的界面进入 `components/`。不会为仅使用一次的元素创建组件。

## 4. 路由与导航

| 路由 | 访问规则 | 用途 |
|---|---|---|
| `/` | 自动判断 | 有会话转 `/apps`，否则转 `/login` |
| `/login` | 公开 | BytCloud Auth 首选入口与本地账号兜底 |
| `/register` | 公开 | 本地账号兜底注册 |
| `/oauth2/:provider/callback` | 公开 | 校验查询参数、交换 JWT、建立会话 |
| `/apps` | 需 JWT | 应用列表 |
| `/apps/new` | 需 JWT | 创建应用 |
| `/apps/:id` | 需 JWT | 应用详情、生命周期和日志 |
| `*` | 自动判断 | 返回当前可访问的主页面 |

路由 loader 从 `session.ts` 读取会话并执行重定向。OAuth2 callback loader 在组件渲染前只执行一次交换流程，避免 React Strict Mode 导致 authorization code 被重复消费。

## 5. API 与认证数据流

### 5.1 统一响应

`lib/api.ts` 定义 `ApiResponse<T>` 与 `ApiError`。每次请求按以下顺序处理：

1. 使用 `VITE_API_BASE_URL` 与 `/api/v1` 组成请求 URL；变量为空时使用同源路径。
2. 受保护请求从 `session.ts` 读取 JWT，并添加 `Authorization: Bearer <token>`。
3. 解析 JSON 后以 `code === 0` 判断成功，不能只看 HTTP 状态。
4. 失败时抛出携带 `code/message` 的 `ApiError`；无法连接或响应不可解析时返回固定、可恢复的中文错误。
5. 收到 `10002`、`20011` 或 `20012` 时清理会话并跳转登录页。

`data` 为可选字段，因为后端 `Success(nil)` 会省略它。

### 5.2 BytCloud Auth 首选认证（内部 Provider：Authentik）

登录页的首要操作是“使用 BytCloud Auth 继续”，不区分“OAuth 登录”和“OAuth 注册”。已有 Authentik 身份由后端识别并登录；首次成功回调由后端 `findOrCreateUser` 创建 Astro `User` 与 `OAuthIdentity`，因此 BytCloud Auth 同时承担 Astro 注册。用户界面只显示 BytCloud Auth，不显示内部 Provider 名称。

BytCloud Auth 是否展示注册/Enrollment 页面由内部 Authentik Provider 配置决定。Astro 前端只负责跳转和回调，不在本地重复实现外部账号注册；Provider 不允许注册时，页面展示 BytCloud Auth 错误并保留本地兜底入口。

```text
登录页点击“使用 BytCloud Auth 继续”
  -> GET /oauth2/authentik/login
  -> 浏览器跳转 data.auth_url
  -> BytCloud Auth 完成登录或 Enrollment Flow
  -> 回调 http://localhost:5173/oauth2/authentik/callback?code&state
  -> 前端 loader 调用 GET /oauth2/authentik/callback?code&state
  -> 后端校验 state、查找或创建 Astro 用户并签发 JWT
  -> 前端保存 JWT 并进入 /apps
```

`code` 与 `state` 只在当前回调请求中使用，不写入存储、日志或页面。Provider 未启用、Enrollment 被拒绝或回调失败时，回调页显示错误与“返回登录”操作；本地登录始终保留。

### 5.3 本地账号兜底

本地登录与注册页面仍然存在，但在认证页中以次要操作呈现，用于 BytCloud Auth 不可用、Provider 暂停或用户已有本地账号的情况。登录成功后只持久化 JWT，不展示或记录 Token。首版使用 `localStorage` 满足刷新后恢复会话；退出或认证错误时立即删除。UUID 暂无界面用途，不建立额外用户状态。

### 5.4 API 与开发代理

`lib/api.ts` 定义 `ApiResponse<T>` 与 `ApiError`。每次请求按以下顺序处理：

1. 使用 `VITE_API_BASE_URL` 与 `/api/v1` 组成请求 URL；变量为空时使用同源路径。
2. 受保护请求从 `session.ts` 读取 JWT，并添加 `Authorization: Bearer <token>`。
3. 解析 JSON 后以 `code === 0` 判断成功，不能只看 HTTP 状态。
4. 失败时抛出携带 `code/message` 的 `ApiError`；无法连接或响应不可解析时返回固定、可恢复的中文错误。
5. 收到 `10002`、`20011` 或 `20012` 时清理会话并跳转登录页。

开发时 Vite 将 `/api` 代理到 `http://localhost:8080`，因此无需扩大后端 CORS 范围。生产环境需同源部署或单独设计 CORS/反向代理，不属于首版。

## 6. 页面状态与交互

- 列表、详情、日志均使用页面局部状态，不做全局缓存和后台轮询。
- 所有提交和生命周期请求使用 `pending` 标记禁用触发控件。
- 列表和详情提供显式刷新按钮；生命周期操作完成后重新请求详情。
- `running` 提供停止与重启，`stopped` 提供启动；`pending/starting/restarting/unknown` 只提供刷新，避免在过渡状态重复操作。
- 删除使用原生 `<dialog>` 二次确认；删除成功后返回列表。
- 日志默认请求最近 100 行，以 `<pre>` 保留换行并只允许日志区域横向滚动。
- 成功反馈短暂显示，错误反馈保留到用户处理或重试，不用颜色作为唯一状态信息。

## 7. 表单验证

- 登录：用户名、密码必填。
- 注册：用户名、密码、邮箱必填，邮箱使用原生 `type="email"`；不在前端复制后端尚未执行的长度规则。
- 创建应用：名称、镜像、副本必填；名称按容器平台实际要求限制为小写字母、数字和短横线且首尾为字母或数字；副本范围为 1-10；端口可空，否则为 1-65535。
- HTML 原生约束负责基础校验，服务端 `message` 负责业务错误；字段均有可见 label 和就近错误反馈。

前端校验只改善体验，不替代后端信任边界校验。

## 8. 视觉系统

界面是工作台，不建立营销首页。桌面使用紧凑的列表布局，移动端将同一列表重排为纵向信息行；不使用卡片嵌套。

- 字体：系统无衬线字体；日志使用系统等宽字体，不加载远程字体。
- 色彩：浅中性背景、白色工作面、深色正文；绿色主操作、蓝色信息、琥珀色警告、红色危险操作，避免单一色系。
- 圆角：控件与边界最多 8px；不使用渐变、装饰光斑或大面积深色背景。
- token：在一个 CSS 文件中按 primitive -> semantic -> component 分段，组件只引用语义变量。
- 品牌：认证页和应用框架首屏明确显示 `Astro` 与 Lucide Orbit 图标；认证主按钮显示 `BytCloud Auth`，不创建非官方品牌图形。
- 命名边界：`BytCloud Auth` 是唯一用户可见的认证品牌；`Authentik` 只允许出现在代码、配置、技术文档和运维错误上下文，不出现在普通 UI 文案。
- 可访问性：16px 基础字号、可见 focus ring、文本对比度至少 4.5:1、图标按钮有 `aria-label/title`，动态反馈使用 `aria-live`，并尊重 `prefers-reduced-motion`。
- 响应式检查视口：375、768、1024、1440px；整页不能横向滚动。

## 9. 兼容性与风险

- JWT 存在 `localStorage` 会暴露于同源 XSS。现有后端只接受 Bearer Token，首版不扩大为 Cookie 会话；当系统进入公网生产部署时，应优先评估 HttpOnly/Secure/SameSite Cookie 的 BFF 方案。
- 后端没有查询已启用 OAuth2 Provider 的接口，因此首版固定展示 BytCloud Auth 首选入口（内部 provider key 为 `authentik`）；其是否允许新用户注册由 Authentik Enrollment Flow 决定，未配置或失败时使用本地账号兜底。新增多 Provider 时再引入 Provider 列表契约。
- 应用 API 返回模型不含端口，因此端口只在创建时输入，详情不虚构该字段。
- 状态同步可能延迟，首版使用手动刷新和操作后重取，不增加轮询。

## 10. 验证与回滚

验证包含前端 lint、单元测试、生产构建、后端现有测试/构建/lint，以及使用浏览器 Mock API 检查认证、列表、创建、详情、操作、日志和 OAuth2 回调错误状态。桌面与移动截图必须检查内容重叠、空白画面和横向溢出。

回滚时删除 `web/` 与 Makefile 前端目标，并恢复 Authentik 示例回调地址即可；现有后端 API 与数据无需迁移。
