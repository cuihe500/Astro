# Astro Web

Astro Web 是面向用户的 React + TypeScript + Vite 单页前端。首版提供登录、注册、应用创建、生命周期管理和最近日志查看，界面认证品牌统一使用 **BytCloud Auth**。

## 命令

所有前端命令通过仓库根目录 Makefile 执行：

```bash
make frontend-install
make frontend-run
make frontend-lint
make frontend-test
make frontend-build
make frontend-check
```

`make frontend-run` 默认启动 Vite 开发服务器 `http://localhost:5173`。开发代理把 `/api` 转发到 `http://localhost:8080`，因此本地联调不需要后端 CORS。若 API 位于其他地址，可在 `web/.env.local` 设置：

```bash
VITE_API_BASE_URL=http://localhost:8080
```

配置为空时使用同源 `/api/v1`；生产部署需要同源静态托管或另行配置反向代理，首版不包含生产部署拓扑。

## OAuth2 回调

用户可见的 OAuth2 公开别名为 `bytcloudauth`，界面品牌为 BytCloud Auth。后端内部配置键和既有身份记录继续使用内部 Provider 键，不出现在浏览器 URL、前端请求或 state 中。启用 provider 时，后端配置和 Provider 注册的回调地址应使用：

```text
http://localhost:5173/oauth2/bytcloudauth/callback
```

生产环境登记地址为：

```text
https://astro.bytcloud.org/oauth2/bytcloudauth/callback
```

前端只把当前回调中的 `code` 和 `state` 交给公开 callback API；authorization code、state、client secret、外部 access token 和 Astro JWT 不写入日志或页面。首次成功登录由后端创建本地用户和身份记录，前端不复制外部 Enrollment Flow。

## 目录边界

- `src/app`：路由、登录守卫和 OAuth2 callback loader。
- `src/components`：跨业务页面复用的工作台框架和反馈状态。
- `src/features/auth`：认证 API、唯一会话读写边界和认证页面。
- `src/features/apps`：应用类型、API、状态语义和应用页面。
- `src/lib`：统一响应解包、Bearer 请求和错误类型。
- `src/styles.css`：响应式布局、颜色 token、focus 和 reduced-motion 样式。

应用页面不直接解析后端原始 JSON；所有 API 均通过 `code === 0` 判断成功，并优先展示后端 `message`。
